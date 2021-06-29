///*
//Copyright (C) BABEC. All rights reserved.
//
//SPDX-License-Identifier: Apache-2.0
//*/
//
package common

import (
	"bytes"
	"chainmaker.org/chainmaker-go/common/crypto/hash"
	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/common/scheduler"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/monitor"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

const (
	DEFAULTDURATION = 1000     // default proposal duration, millis seconds
	DEFAULTVERSION  = "v1.0.0" // default version of chain
)

type BlockBuilderConf struct {
	ChainId         string                   // chain id, to identity this chain
	TxPool          protocol.TxPool          // tx pool provides tx batch
	TxScheduler     protocol.TxScheduler     // scheduler orders tx batch into DAG form and returns a block
	SnapshotManager protocol.SnapshotManager // snapshot manager
	Identity        protocol.SigningMember   // identity manager
	LedgerCache     protocol.LedgerCache     // ledger cache
	ProposalCache   protocol.ProposalCache
	ChainConf       protocol.ChainConf // chain config
	Log             protocol.Logger
	StoreHelper     conf.StoreHelper
}

type BlockBuilder struct {
	chainId         string                   // chain id, to identity this chain
	txPool          protocol.TxPool          // tx pool provides tx batch
	txScheduler     protocol.TxScheduler     // scheduler orders tx batch into DAG form and returns a block
	snapshotManager protocol.SnapshotManager // snapshot manager
	identity        protocol.SigningMember   // identity manager
	ledgerCache     protocol.LedgerCache     // ledger cache
	proposalCache   protocol.ProposalCache
	chainConf       protocol.ChainConf // chain config
	log             protocol.Logger
	storeHelper     conf.StoreHelper
}

func NewBlockBuilder(conf *BlockBuilderConf) *BlockBuilder {
	creatorBlock := &BlockBuilder{
		chainId:         conf.ChainId,
		txPool:          conf.TxPool,
		txScheduler:     conf.TxScheduler,
		snapshotManager: conf.SnapshotManager,
		identity:        conf.Identity,
		ledgerCache:     conf.LedgerCache,
		proposalCache:   conf.ProposalCache,
		chainConf:       conf.ChainConf,
		log:             conf.Log,
		storeHelper:     conf.StoreHelper,
	}

	return creatorBlock
}

func (bb *BlockBuilder) GenerateNewBlock(proposingHeight int64, preHash []byte, txBatch []*commonpb.Transaction) (*commonpb.Block, []int64, error) {
	timeLasts := make([]int64, 0)
	currentHeight, _ := bb.ledgerCache.CurrentHeight()
	lastBlock := bb.findLastBlockFromCache(proposingHeight, preHash, currentHeight)
	if lastBlock == nil {
		return nil, nil, fmt.Errorf("no pre block found [%d] (%x)", proposingHeight-1, preHash)
	}
	block, err := InitNewBlock(lastBlock, bb.identity, bb.chainId, bb.chainConf)
	if err != nil {
		return block, timeLasts, err
	}
	if block == nil {
		bb.log.Warnf("generate new block failed, block == nil")
		return nil, timeLasts, fmt.Errorf("generate new block failed, block == nil")
	}
	//if txBatch == nil {
	//	// For ChainedBFT consensus, generate an empty block if tx batch is empty.
	//	return block, timeLasts, nil
	//}

	// validate tx and verify ACL，split into 2 slice according to result
	// validatedTxs are txs passed validate and should be executed by contract
	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	var validatedTxs = txBatch

	// txScheduler handle：
	// 1. execute transaction and fill the result, include rw set digest, and remove from txBatch
	// 2. calculate dag and fill into block
	// 3. fill txs into block
	// If only part of the txBatch is filled into the Block, consider executing it again
	ssStartTick := utils.CurrentTimeMillisSeconds()
	snapshot := bb.snapshotManager.NewSnapshot(lastBlock, block)
	vmStartTick := utils.CurrentTimeMillisSeconds()
	ssLasts := vmStartTick - ssStartTick
	bb.storeHelper.BeginDbTransaction(snapshot.GetBlockchainStore(), block.GetTxKey())
	txRWSetMap, contractEventMap, err := bb.txScheduler.Schedule(block, validatedTxs, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
	timeLasts = append(timeLasts, ssLasts, vmLasts)

	if err != nil {
		return nil, timeLasts, fmt.Errorf("schedule block(%d,%x) error %s",
			block.Header.BlockHeight, block.Header.BlockHash, err)
	}

	// deal with the special situation：
	// 1. only one tx and schedule time out
	// 2. package the empty block
	if !utils.CanProposeEmptyBlock(bb.chainConf.ChainConfig().Consensus.Type) && len(block.Txs) == 0 {
		return nil, timeLasts, fmt.Errorf("no txs in scheduled block, proposing block ends")
	}

	finalizeStartTick := utils.CurrentTimeMillisSeconds()
	err = FinalizeBlock(
		block,
		txRWSetMap,
		aclFailTxs,
		bb.chainConf.ChainConfig().Crypto.Hash,
		bb.log)
	finalizeLasts := utils.CurrentTimeMillisSeconds() - finalizeStartTick
	if err != nil {
		return nil, timeLasts, fmt.Errorf("finalizeBlock block(%d,%s) error %s",
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash), err)
	}
	timeLasts = append(timeLasts, finalizeLasts)
	// get txs schedule timeout and put back to txpool
	var txsTimeout = make([]*commonpb.Transaction, 0)
	if len(txRWSetMap) < len(txBatch) {
		// if tx not in txRWSetMap, tx should be put back to txpool
		for _, tx := range txBatch {
			if _, ok := txRWSetMap[tx.Header.TxId]; !ok {
				txsTimeout = append(txsTimeout, tx)
			}
		}
		bb.txPool.RetryAndRemoveTxs(txsTimeout, nil)
	}

	// cache proposed block
	bb.log.Debugf("set proposed block(%d,%x)", block.Header.BlockHeight, block.Header.BlockHash)
	if err = bb.proposalCache.SetProposedBlock(block, txRWSetMap, contractEventMap, true); err != nil {
		return block, timeLasts, err
	}
	bb.proposalCache.SetProposedAt(block.Header.BlockHeight)

	return block, timeLasts, nil
}

func (bb *BlockBuilder) findLastBlockFromCache(proposingHeight int64, preHash []byte, currentHeight int64) *commonpb.Block {
	var lastBlock *commonpb.Block
	if currentHeight+1 == proposingHeight {
		lastBlock = bb.ledgerCache.GetLastCommittedBlock()
	} else {
		lastBlock, _ = bb.proposalCache.GetProposedBlockByHashAndHeight(preHash, proposingHeight-1)
	}
	return lastBlock
}

func InitNewBlock(
	lastBlock *commonpb.Block,
	identity protocol.SigningMember,
	chainId string,
	chainConf protocol.ChainConf) (*commonpb.Block, error) {
	// get node pk from identity
	proposer, err := identity.Serialize(true)
	if err != nil {
		return nil, fmt.Errorf("identity serialize failed, %s", err)
	}
	preConfHeight := lastBlock.Header.PreConfHeight
	// if last block is config block, then this block.preConfHeight is last block height
	if utils.IsConfBlock(lastBlock) {
		preConfHeight = lastBlock.Header.BlockHeight
	}

	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        chainId,
			BlockHeight:    lastBlock.Header.BlockHeight + 1,
			PreBlockHash:   lastBlock.Header.BlockHash,
			BlockHash:      nil,
			PreConfHeight:  preConfHeight,
			BlockVersion:   getChainVersion(chainConf),
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: utils.CurrentTimeSeconds(),
			Proposer:       proposer,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag:            &commonpb.DAG{},
		Txs:            nil,
		AdditionalData: nil,
	}
	return block, nil
}

func FinalizeBlock(
	block *commonpb.Block,
	txRWSetMap map[string]*commonpb.TxRWSet,
	aclFailTxs []*commonpb.Transaction,
	hashType string,
	logger protocol.Logger) error {

	if aclFailTxs != nil && len(aclFailTxs) > 0 {
		// append acl check failed txs to the end of block.Txs
		block.Txs = append(block.Txs, aclFailTxs...)
	}

	// TxCount contains acl verify failed txs and invoked contract txs
	txCount := len(block.Txs)
	block.Header.TxCount = int64(txCount)

	// TxRoot/RwSetRoot
	var err error
	txHashes := make([][]byte, txCount)
	for i, tx := range block.Txs {
		// finalize tx, put rwsethash into tx.Result
		rwSet := txRWSetMap[tx.Header.TxId]
		if rwSet == nil {
			rwSet = &commonpb.TxRWSet{
				TxId:     tx.Header.TxId,
				TxReads:  nil,
				TxWrites: nil,
			}
		}
		rwSetHash, err := utils.CalcRWSetHash(hashType, rwSet)
		logger.DebugDynamic(func() string {
			return fmt.Sprintf("CalcRWSetHash rwset: %+v ,hash: %x", rwSet, rwSetHash)
		})
		if err != nil {
			return err
		}
		if tx.Result == nil {
			// in case tx.Result is nil, avoid panic
			e := fmt.Errorf("tx(%s) result == nil", tx.Header.TxId)
			logger.Error(e.Error())
			return e
		}
		tx.Result.RwSetHash = rwSetHash
		// calculate complete tx hash, include tx.Header, tx.Payload, tx.Result
		txHash, err := utils.CalcTxHash(hashType, tx)
		if err != nil {
			return err
		}
		txHashes[i] = txHash
	}

	block.Header.TxRoot, err = hash.GetMerkleRoot(hashType, txHashes)
	if err != nil {
		logger.Warnf("get tx merkle root error %s", err)
		return err
	}
	block.Header.RwSetRoot, err = utils.CalcRWSetRoot(hashType, block.Txs)
	if err != nil {
		logger.Warnf("get rwset merkle root error %s", err)
		return err
	}

	// DagDigest
	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
	if err != nil {
		logger.Warnf("get dag hash error %s", err)
		return err
	}
	block.Header.DagHash = dagHash

	return nil
}

// IsTxCountValid, to check if txcount in block is valid
func IsTxCountValid(block *commonpb.Block) error {
	if block.Header.TxCount != int64(len(block.Txs)) {
		return fmt.Errorf("txcount expect %d, got %d", block.Header.TxCount, len(block.Txs))
	}
	return nil
}

// IsHeightValid, to check if block height is valid
func IsHeightValid(block *commonpb.Block, currentHeight int64) error {
	if currentHeight+1 != block.Header.BlockHeight {
		return fmt.Errorf("height expect %d, got %d", currentHeight+1, block.Header.BlockHeight)
	}
	return nil
}

// IsPreHashValid, to check if block.preHash equals with last block hash
func IsPreHashValid(block *commonpb.Block, preHash []byte) error {
	if !bytes.Equal(preHash, block.Header.PreBlockHash) {
		return fmt.Errorf("prehash expect %x, got %x", preHash, block.Header.PreBlockHash)
	}
	return nil
}

// IsBlockHashValid, to check if block hash equals with result calculated from block
func IsBlockHashValid(block *commonpb.Block, hashType string) error {
	hash, err := utils.CalcBlockHash(hashType, block)
	if err != nil {
		return fmt.Errorf("calc block hash error")
	}
	if !bytes.Equal(hash, block.Header.BlockHash) {
		return fmt.Errorf("block hash expect %x, got %x", block.Header.BlockHash, hash)
	}
	return nil
}

// IsTxDuplicate, to check if there is duplicated transactions in one block
func IsTxDuplicate(txs []*commonpb.Transaction) bool {
	txSet := make(map[string]struct{})
	exist := struct{}{}
	for _, tx := range txs {
		if tx == nil || tx.Header == nil {
			return true
		}
		txSet[tx.Header.TxId] = exist
	}
	// length of set < length of txs, means txs have duplicate tx
	return len(txSet) < len(txs)
}

// IsMerkleRootValid, to check if block merkle root equals with simulated merkle root
func IsMerkleRootValid(block *commonpb.Block, txHashes [][]byte, hashType string) error {
	txRoot, err := hash.GetMerkleRoot(hashType, txHashes)
	if err != nil || !bytes.Equal(txRoot, block.Header.TxRoot) {
		return fmt.Errorf("txroot expect %x, got %x, err: %s", block.Header.TxRoot, txRoot, err.Error())
	}
	return nil
}

// IsDagHashValid, to check if block dag equals with simulated block dag
func IsDagHashValid(block *commonpb.Block, hashType string) error {
	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
	if err != nil || !bytes.Equal(dagHash, block.Header.DagHash) {
		return fmt.Errorf("dag expect %x, got %x", block.Header.DagHash, dagHash)
	}
	return nil
}

// IsRWSetHashValid, to check if read write set is valid
func IsRWSetHashValid(block *commonpb.Block, hashType string) error {
	rwSetRoot, err := utils.CalcRWSetRoot(hashType, block.Txs)
	if err != nil {
		return fmt.Errorf("calc rwset error, %s", err)
	}
	if !bytes.Equal(rwSetRoot, block.Header.RwSetRoot) {
		return fmt.Errorf("rwset expect %x, got %x", block.Header.RwSetRoot, rwSetRoot)
	}
	return nil
}

// getChainVersion, get chain version from config.
// If not access from config, use default value.
func getChainVersion(chainConf protocol.ChainConf) []byte {
	if chainConf == nil || chainConf.ChainConfig() == nil {
		return []byte(DEFAULTVERSION)
	}
	return []byte(chainConf.ChainConfig().Version)
}

func VerifyHeight(height int64, ledgerCache protocol.LedgerCache) error {
	currentHeight, err := ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}
	if currentHeight+1 != height {
		return fmt.Errorf("verify height fail,expected [%d]", currentHeight+1)
	}
	return nil
}

func CheckBlockDigests(block *commonpb.Block, txHashes [][]byte, hashType string, log protocol.Logger) error {
	if err := IsMerkleRootValid(block, txHashes, hashType); err != nil {
		log.Error(err)
		return err
	}
	// verify DAG hash
	if err := IsDagHashValid(block, hashType); err != nil {
		log.Error(err)
		return err
	}
	// verify read write set, check if simulate result is equal with rwset in block header
	if err := IsRWSetHashValid(block, hashType); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func CheckVacuumBlock(block *commonpb.Block, consensusType consensus.ConsensusType) error {
	if 0 == block.Header.TxCount {
		if utils.CanProposeEmptyBlock(consensusType) {
			// for consensus that allows empty block, skip txs verify
			return nil
		} else {
			// for consensus that NOT allows empty block, return error
			return fmt.Errorf("tx must not empty")
		}
	}
	return nil
}

type VerifierBlockConf struct {
	ChainConf       protocol.ChainConf
	Log             protocol.Logger
	LedgerCache     protocol.LedgerCache
	Ac              protocol.AccessControlProvider
	SnapshotManager protocol.SnapshotManager
	VmMgr           protocol.VmManager
	TxPool          protocol.TxPool
	BlockchainStore protocol.BlockchainStore
	ProposalCache   protocol.ProposalCache // proposal cache
	StoreHelper     conf.StoreHelper
}

type VerifierBlock struct {
	chainConf       protocol.ChainConf
	log             protocol.Logger
	ledgerCache     protocol.LedgerCache
	ac              protocol.AccessControlProvider
	snapshotManager protocol.SnapshotManager
	vmMgr           protocol.VmManager
	txScheduler     protocol.TxScheduler
	txPool          protocol.TxPool
	blockchainStore protocol.BlockchainStore
	proposalCache   protocol.ProposalCache // proposal cache
	storeHelper     conf.StoreHelper
}

func NewVerifierBlock(conf *VerifierBlockConf) *VerifierBlock {
	verifyBlock := &VerifierBlock{
		chainConf:       conf.ChainConf,
		log:             conf.Log,
		ledgerCache:     conf.LedgerCache,
		ac:              conf.Ac,
		snapshotManager: conf.SnapshotManager,
		vmMgr:           conf.VmMgr,
		txPool:          conf.TxPool,
		blockchainStore: conf.BlockchainStore,
		proposalCache:   conf.ProposalCache,
		storeHelper:     conf.StoreHelper,
	}
	var schedulerFactory scheduler.TxSchedulerFactory
	verifyBlock.txScheduler = schedulerFactory.NewTxScheduler(verifyBlock.vmMgr, verifyBlock.chainConf, conf.StoreHelper)
	return verifyBlock
}

func (vb *VerifierBlock) FetchLastBlock(block *commonpb.Block, lastBlock *commonpb.Block) (*commonpb.Block, error) {
	currentHeight, _ := vb.ledgerCache.CurrentHeight()
	if currentHeight >= block.Header.BlockHeight {
		return nil, commonErrors.ErrBlockHadBeenCommited
	}

	if currentHeight+1 == block.Header.BlockHeight {
		lastBlock = vb.ledgerCache.GetLastCommittedBlock()
	} else {
		lastBlock, _ = vb.proposalCache.GetProposedBlockByHashAndHeight(block.Header.PreBlockHash, block.Header.BlockHeight-1)
	}
	if lastBlock == nil {
		return nil, fmt.Errorf("no pre block found [%d](%x)", block.Header.BlockHeight-1, block.Header.PreBlockHash)
	}
	return lastBlock, nil
}

// validateBlock, validate block and transactions
func (vb *VerifierBlock) ValidateBlock(
	block, lastBlock *commonpb.Block,
	hashType string, timeLasts []int64) (map[string]*commonpb.TxRWSet, map[string][]*commonpb.ContractEvent, []int64, error) {

	if err := IsBlockHashValid(block, vb.chainConf.ChainConfig().Crypto.Hash); err != nil {
		return nil, nil, timeLasts, err
	}

	// verify block sig and also verify identity and auth of block proposer
	startSigTick := utils.CurrentTimeMillisSeconds()
	vb.log.Debugf("verify block \n %s", utils.FormatBlock(block))
	if ok, err := utils.VerifyBlockSig(hashType, block, vb.ac); !ok || err != nil {
		return nil, nil, timeLasts, fmt.Errorf("(%d,%x - %x,%x) [signature]",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.Proposer, block.Header.Signature)
	}
	sigLasts := utils.CurrentTimeMillisSeconds() - startSigTick
	timeLasts = append(timeLasts, sigLasts)

	err := CheckVacuumBlock(block, vb.chainConf.ChainConfig().Consensus.Type)
	if err != nil {
		return nil, nil, timeLasts, err
	}
	// we must new a snapshot for the vacant block,
	// otherwise the subsequent snapshot can not link to the previous snapshot.
	snapshot := vb.snapshotManager.NewSnapshot(lastBlock, block)
	if len(block.Txs) == 0 {
		return nil, nil, timeLasts, nil
	}
	// verify if txs are duplicate in this block
	if IsTxDuplicate(block.Txs) {
		return nil, nil, timeLasts, fmt.Errorf("tx duplicate")
	}

	// simulate with DAG, and verify read write set
	startVMTick := utils.CurrentTimeMillisSeconds()
	vb.storeHelper.BeginDbTransaction(snapshot.GetBlockchainStore(), block.GetTxKey())
	txRWSetMap, txResultMap, err := vb.txScheduler.SimulateWithDag(block, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - startVMTick
	timeLasts = append(timeLasts, vmLasts)
	if err != nil {
		return nil, nil, timeLasts, fmt.Errorf("simulate %s", err)
	}
	if block.Header.TxCount != int64(len(txRWSetMap)) {
		return nil, nil, timeLasts, fmt.Errorf("simulate txcount expect %d, got %d",
			block.Header.TxCount, len(txRWSetMap))
	}

	// 2.transaction verify
	startTxTick := utils.CurrentTimeMillisSeconds()
	verifierTxConf := &VerifierTxConfig{
		Block:       block,
		TxResultMap: txResultMap,
		TxRWSetMap:  txRWSetMap,
		ChainConf:   vb.chainConf,
		Log:         vb.log,
		Ac:          vb.ac,
		TxPool:      vb.txPool,
		Store:       vb.blockchainStore,
	}
	verifiertx := NewVerifierTx(verifierTxConf)
	txHashes, _, errTxs, err := verifiertx.verifierTxs(block)
	txLasts := utils.CurrentTimeMillisSeconds() - startTxTick
	timeLasts = append(timeLasts, txLasts)
	if err != nil {
		if len(errTxs) > 0 {
			vb.log.Warn("[Duplicate txs] delete the err txs")
			vb.txPool.RetryAndRemoveTxs(nil, errTxs)
		}
		return nil, nil, timeLasts, fmt.Errorf("verify failed [%d](%x), %s ",
			block.Header.BlockHeight, block.Header.PreBlockHash, err)
	}
	//if protocol.CONSENSUS_VERIFY == mode && len(newAddTx) > 0 {
	//	v.txPool.AddTrustedTx(newAddTx)
	//}

	// get contract events
	contractEventMap := make(map[string][]*commonpb.ContractEvent)
	for _, tx := range block.Txs {
		var events []*commonpb.ContractEvent
		if result, ok := txResultMap[tx.Header.TxId]; ok {
			events = result.ContractResult.ContractEvent
		}
		contractEventMap[tx.Header.TxId] = events
	}
	// verify TxRoot
	startRootsTick := utils.CurrentTimeMillisSeconds()
	err = CheckBlockDigests(block, txHashes, hashType, vb.log)
	if err != nil {
		return txRWSetMap, contractEventMap, timeLasts, err
	}
	rootsLast := utils.CurrentTimeMillisSeconds() - startRootsTick
	timeLasts = append(timeLasts, rootsLast)

	return txRWSetMap, contractEventMap, timeLasts, nil
}

func CheckPreBlock(block *commonpb.Block, lastBlock *commonpb.Block, err error,
	lastBlockHash []byte, proposedHeight int64) error {

	if err = IsHeightValid(block, proposedHeight); err != nil {
		return err
	}
	// check if this block pre hash is equal with last block hash
	return IsPreHashValid(block, lastBlockHash)
}

// BlockCommitterImpl implements BlockCommitter interface.
// To commit a block after it is confirmed by consensus module.
type BlockCommitterImpl struct {
	chainId string // chain id, to identity this chain
	// Store is a block store that will only fetch data locally
	blockchainStore protocol.BlockchainStore // blockchain store
	snapshotManager protocol.SnapshotManager // snapshot manager
	txPool          protocol.TxPool          // transaction pool
	chainConf       protocol.ChainConf       // chain config

	ledgerCache           protocol.LedgerCache        // ledger cache
	proposalCache         protocol.ProposalCache      // proposal cache
	log                   protocol.Logger             // logger
	msgBus                msgbus.MessageBus           // message bus
	mu                    sync.Mutex                  // lock, to avoid concurrent block commit
	subscriber            *subscriber.EventSubscriber // subscriber
	verifier              protocol.BlockVerifier      // block verifier
	commonCommit          *CommitBlock
	metricBlockSize       *prometheus.HistogramVec // metric block size
	metricBlockCounter    *prometheus.CounterVec   // metric block counter
	metricTxCounter       *prometheus.CounterVec   // metric transaction counter
	metricBlockCommitTime *prometheus.HistogramVec // metric block commit time
	storeHelper           conf.StoreHelper
}

type BlockCommitterConfig struct {
	ChainId         string
	BlockchainStore protocol.BlockchainStore
	SnapshotManager protocol.SnapshotManager
	TxPool          protocol.TxPool
	LedgerCache     protocol.LedgerCache
	ProposedCache   protocol.ProposalCache
	ChainConf       protocol.ChainConf
	MsgBus          msgbus.MessageBus
	Subscriber      *subscriber.EventSubscriber
	Verifier        protocol.BlockVerifier
	StoreHelper     conf.StoreHelper
}

func NewBlockCommitter(config BlockCommitterConfig, log protocol.Logger) (protocol.BlockCommitter, error) {
	blockchain := &BlockCommitterImpl{
		chainId:         config.ChainId,
		blockchainStore: config.BlockchainStore,
		snapshotManager: config.SnapshotManager,
		txPool:          config.TxPool,
		ledgerCache:     config.LedgerCache,
		proposalCache:   config.ProposedCache,
		log:             log,
		chainConf:       config.ChainConf,
		msgBus:          config.MsgBus,
		subscriber:      config.Subscriber,
		verifier:        config.Verifier,
		storeHelper:     config.StoreHelper,
	}

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		blockchain.metricBlockSize = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockSize,
			monitor.HelpCurrentBlockSizeMetric, prometheus.ExponentialBuckets(1024, 2, 12), monitor.ChainId)

		blockchain.metricBlockCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockCounter,
			monitor.HelpBlockCountsMetric, monitor.ChainId)

		blockchain.metricTxCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricTxCounter,
			monitor.HelpTxCountsMetric, monitor.ChainId)

		blockchain.metricBlockCommitTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockCommitTime,
			monitor.HelpBlockCommitTimeMetric, []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, monitor.ChainId)
	}

	cbConf := &CommitBlockConf{
		Store:                 blockchain.blockchainStore,
		Log:                   blockchain.log,
		SnapshotManager:       blockchain.snapshotManager,
		TxPool:                blockchain.txPool,
		LedgerCache:           blockchain.ledgerCache,
		ChainConf:             blockchain.chainConf,
		MsgBus:                blockchain.msgBus,
		MetricBlockCommitTime: blockchain.metricBlockCommitTime,
		MetricBlockCounter:    blockchain.metricBlockCounter,
		MetricBlockSize:       blockchain.metricBlockSize,
		MetricTxCounter:       blockchain.metricTxCounter,
	}
	blockchain.commonCommit = NewCommitBlock(cbConf)

	return blockchain, nil
}

func (chain *BlockCommitterImpl) isBlockLegal(blk *commonpb.Block) error {
	lastBlock := chain.ledgerCache.GetLastCommittedBlock()
	if lastBlock == nil {
		// 获取上一区块
		// 首次进入，从DB获取最新区块
		return fmt.Errorf("get last block == nil ")
	}

	if lastBlock.Header.BlockHeight >= blk.Header.BlockHeight {
		return commonErrors.ErrBlockHadBeenCommited
	}
	// block height verify
	if blk.Header.BlockHeight != lastBlock.Header.BlockHeight+1 {
		return fmt.Errorf("isBlockLegal() failed: Height is less than chaintip")
	}
	// block pre hash verify
	if !bytes.Equal(blk.Header.PreBlockHash, lastBlock.Header.BlockHash) {
		return fmt.Errorf("isBlockLegal() failed: PrevHash invalid (%x != %x)",
			blk.Header.PreBlockHash, lastBlock.Header.BlockHash)
	}

	blkHash, err := utils.CalcBlockHash(chain.chainConf.ChainConfig().Crypto.Hash, blk)
	if err != nil || !bytes.Equal(blk.Header.BlockHash, blkHash) {
		return fmt.Errorf("isBlockLegal() failed: BlockHash invalid (%x != %x)",
			blkHash, blk.Header.BlockHash)
	}

	return nil
}

func (chain *BlockCommitterImpl) AddBlock(block *commonpb.Block) (err error) {
	defer func() {
		panicErr := recover()
		if err == nil {
			if panicErr != nil {
				err = fmt.Errorf(fmt.Sprint(panicErr))
			} else {
				return
			}
		}
		// rollback sql
		chain.log.Error("cache add block err: ", err)
		if sqlErr := chain.storeHelper.RollBack(block, chain.blockchainStore); sqlErr != nil {
			chain.log.Errorf("block [%d] rollback sql failed: %s", block.Header.BlockHeight, sqlErr)
		}
	}()

	startTick := utils.CurrentTimeMillisSeconds()
	chain.log.Debugf("add block(%d,%x)=(%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, block.Header.TxCount, len(block.Txs))
	chain.mu.Lock()
	defer chain.mu.Unlock()

	height := block.Header.BlockHeight
	if err = chain.isBlockLegal(block); err != nil {
		chain.log.Errorf("block illegal [%d](hash:%x), %s", height, block.Header.BlockHash, err)
		return err
	}
	lastProposed, rwSetMap, conEventMap := chain.proposalCache.GetProposedBlock(block)
	if err = chain.checkLastProposedBlock(block, lastProposed, err, height, rwSetMap, conEventMap); err != nil {
		return err
	}

	checkLasts := utils.CurrentTimeMillisSeconds() - startTick

	dbLasts, snapshotLasts, confLasts, otherLasts, pubEvent, err := chain.commonCommit.CommitBlock(block, rwSetMap, conEventMap)
	if err != nil {
		chain.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
	}

	// Remove txs from txpool. Remove will invoke proposeSignal from txpool if pool size > txcount
	startPoolTick := utils.CurrentTimeMillisSeconds()
	txRetry := chain.syncWithTxPool(block, height)
	chain.log.Infof("remove txs[%d] and retry txs[%d] in add block", len(block.Txs), len(txRetry))
	chain.txPool.RetryAndRemoveTxs(txRetry, block.Txs)
	poolLasts := utils.CurrentTimeMillisSeconds() - startPoolTick

	chain.proposalCache.ClearProposedBlockAt(height)

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	chain.log.Infof("commit block [%d](count:%d,hash:%x), time used(check:%d,db:%d,ss:%d,conf:%d,pool:%d,pubConEvent:%d,other:%d,total:%d)",
		height, block.Header.TxCount, block.Header.BlockHash, checkLasts, dbLasts, snapshotLasts, confLasts, poolLasts, pubEvent, otherLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		chain.metricBlockCommitTime.WithLabelValues(chain.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}

func (chain *BlockCommitterImpl) syncWithTxPool(block *commonpb.Block, height int64) []*commonpb.Transaction {
	proposedBlocks := chain.proposalCache.GetProposedBlocksAt(height)
	txRetry := make([]*commonpb.Transaction, 0, localconf.ChainMakerConfig.TxPoolConfig.BatchMaxSize)
	chain.log.Debugf("has %d blocks in height: %d", len(proposedBlocks), height)
	keepTxs := make(map[string]struct{}, len(block.Txs))
	for _, tx := range block.Txs {
		keepTxs[tx.Header.TxId] = struct{}{}
	}
	for _, b := range proposedBlocks {
		if bytes.Equal(b.Header.BlockHash, block.Header.BlockHash) {
			continue
		}
		for _, tx := range b.Txs {
			if _, ok := keepTxs[tx.Header.TxId]; !ok {
				txRetry = append(txRetry, tx)
			}
		}
	}
	return txRetry
}

func (chain *BlockCommitterImpl) checkLastProposedBlock(block *commonpb.Block, lastProposed *commonpb.Block,
	err error, height int64, rwSetMap map[string]*commonpb.TxRWSet, conEventMap map[string][]*commonpb.ContractEvent) error {
	if lastProposed != nil {
		return nil
	}
	err = chain.verifier.VerifyBlock(block, protocol.SYNC_VERIFY)
	if err != nil {
		chain.log.Error("block verify failed [%d](hash:%x), %s", height, block.Header.BlockHash, err)
		return err
	}
	lastProposed, rwSetMap, conEventMap = chain.proposalCache.GetProposedBlock(block)
	if lastProposed == nil {
		chain.log.Error("block not verified [%d](hash:%x)", height, block.Header.BlockHash)
		return fmt.Errorf("block not verified [%d](hash:%x)", height, block.Header.BlockHash)
	}
	return nil
}
