/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"encoding/hex"
	"fmt"
	"sync"

	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/consensus"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/monitor"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/prometheus/client_golang/prometheus"
)

const LOCKED = "LOCKED" // LOCKED mark

// BlockVerifierImpl implements BlockVerifier interface.
// Verify block and transactions.
type BlockVerifierImpl struct {
	chainId         string                   // chain id, to identity this chain
	msgBus          msgbus.MessageBus        // message bus
	txScheduler     protocol.TxScheduler     // scheduler orders tx batch into DAG form and returns a block
	snapshotManager protocol.SnapshotManager // snapshot manager
	ledgerCache     protocol.LedgerCache     // ledger cache
	blockchainStore protocol.BlockchainStore // blockchain store

	reentrantLocks *reentrantLocks                // reentrant lock for avoid concurrent verify block
	proposalCache  protocol.ProposalCache         // proposal cache
	chainConf      protocol.ChainConf             // chain config
	ac             protocol.AccessControlProvider // access control manager
	log            *logger.CMLogger               // logger
	txPool         protocol.TxPool                // tx pool to check if tx is duplicate
	mu             sync.Mutex                     // to avoid concurrent map modify

	blockValidator *BlockValidator //block validator
	txValidator    *TxValidator    //tx validator

	metricBlockVerifyTime *prometheus.HistogramVec // metrics monitor
}

type BlockVerifierConfig struct {
	ChainId         string
	MsgBus          msgbus.MessageBus
	SnapshotManager protocol.SnapshotManager
	BlockchainStore protocol.BlockchainStore
	LedgerCache     protocol.LedgerCache
	TxScheduler     protocol.TxScheduler
	ProposedCache   protocol.ProposalCache
	ChainConf       protocol.ChainConf
	AC              protocol.AccessControlProvider
	TxPool          protocol.TxPool
}

func NewBlockVerifier(config BlockVerifierConfig) (protocol.BlockVerifier, error) {
	v := &BlockVerifierImpl{
		chainId:         config.ChainId,
		msgBus:          config.MsgBus,
		txScheduler:     config.TxScheduler,
		snapshotManager: config.SnapshotManager,
		ledgerCache:     config.LedgerCache,
		blockchainStore: config.BlockchainStore,
		reentrantLocks: &reentrantLocks{
			reentrantLocks: make(map[string]interface{}),
		},
		proposalCache: config.ProposedCache,
		chainConf:     config.ChainConf,
		ac:            config.AC,
		log:           logger.GetLoggerByChain(logger.MODULE_CORE, config.ChainId),
		txPool:        config.TxPool,
	}

	v.blockValidator = NewBlockValidator(v.chainId, v.chainConf.ChainConfig().Crypto.Hash)
	v.txValidator = NewTxValidator(v.log, v.chainId, v.chainConf.ChainConfig().Crypto.Hash,
		v.chainConf.ChainConfig().Consensus.Type, v.blockchainStore, v.txPool, v.ac)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}

	return v, nil
}

// verifyStat, statistic for verify steps
type verifyStat struct {
	totalCount  int
	dbLasts     int64
	sigLasts    int64
	othersLasts int64
	sigCount    int
}

// VerifyBlock, to check if block is valid
func (v *BlockVerifierImpl) VerifyBlock(block *commonpb.Block, mode protocol.VerifyMode) error {
	startTick := utils.CurrentTimeMillisSeconds()
	var err error
	if err = utils.IsEmptyBlock(block); err != nil {
		v.log.Error(err)
		return err
	}

	v.log.Debugf("verify receive [%d](%x,%d,%d), from sync %d",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs), mode)
	// avoid concurrent verify, only one block hash can be verified at the same time
	if !v.reentrantLocks.lock(string(block.Header.BlockHash)) {
		v.log.Warnf("block(%d,%x) concurrent verify, yield", block.Header.BlockHeight, block.Header.BlockHash)
		return commonErrors.ErrConcurrentVerify
	}
	defer v.reentrantLocks.unlock(string(block.Header.BlockHash))

	var isValid bool
	var contractEventMap map[string][]*commonpb.ContractEvent
	// to check if the block has verified before
	if b, txRwSet := v.proposalCache.GetProposedBlock(block); b != nil &&
		consensuspb.ConsensusType_SOLO != v.chainConf.ChainConfig().Consensus.Type {
		// the block has verified before
		v.log.Infof("verify success repeat [%d](%x)", block.Header.BlockHeight, block.Header.BlockHash)
		isValid = true
		if protocol.CONSENSUS_VERIFY == mode {
			// consensus mode, publish verify result to message bus
			v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
		}
		lastBlock, _ := v.proposalCache.GetProposedBlockByHashAndHeight(block.Header.PreBlockHash, block.Header.BlockHeight-1)
		if lastBlock == nil {
			v.log.Debugf("no pre-block be found, preHeight:%d, preBlockHash:%x", block.Header.BlockHeight-1, block.Header.PreBlockHash)
			return nil
		}
		cutBlocks := v.proposalCache.KeepProposedBlock(lastBlock.Header.BlockHash, lastBlock.Header.BlockHeight)
		if len(cutBlocks) > 0 {
			v.log.Infof("cut block block hash: %s, height: %v", hex.EncodeToString(lastBlock.Header.BlockHash), lastBlock.Header.BlockHeight)
			v.cutBlocks(cutBlocks, lastBlock)
		}
		err := v.proposalCache.SetProposedBlock(block, txRwSet, v.proposalCache.IsProposedAt(block.Header.BlockHeight))
		return err
	}

	txRWSetMap, contractEventMap, timeLasts, err := v.validateBlock(block)
	if err != nil {
		v.log.Warnf("verify failed [%d](%x),preBlockHash:%x, %s",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, err.Error())
		if protocol.CONSENSUS_VERIFY == mode {
			v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
		}
		return err
	}

	// sync mode, need to verify consensus vote signature
	if protocol.SYNC_VERIFY == mode {
		if err = v.verifyVoteSig(block); err != nil {
			v.log.Warnf("verify failed [%d](%x), votesig %s",
				block.Header.BlockHeight, block.Header.BlockHash, err.Error())
			return err
		}
	}

	// verify success, cache block and read write set
	v.log.Debugf("set proposed block(%d,%x)", block.Header.BlockHeight, block.Header.BlockHash)
	if err = v.proposalCache.SetProposedBlock(block, txRWSetMap, contractEventMap, false); err != nil {
		return err
	}

	// mark transactions in block as pending status in txpool
	v.txPool.AddTxsToPendingCache(block.Txs, block.Header.BlockHeight)

	isValid = true
	if protocol.CONSENSUS_VERIFY == mode {
		v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
	}
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	v.log.Infof("verify success [%d,%x](%v,%d)", block.Header.BlockHeight, block.Header.BlockHash,
		timeLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime.WithLabelValues(v.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}

func (v *BlockVerifierImpl) verifyVoteSig(block *commonpb.Block) error {
	return consensus.VerifyBlockSignatures(v.chainConf, v.ac, v.blockchainStore, block, v.ledgerCache)
}

func parseVerifyResult(block *commonpb.Block, isValid bool) *consensuspb.VerifyResult {
	verifyResult := &consensuspb.VerifyResult{
		VerifiedBlock: block,
	}
	if isValid {
		verifyResult.Code = consensuspb.VerifyResult_SUCCESS
		verifyResult.Msg = "OK"
	} else {
		verifyResult.Msg = "FAIL"
		verifyResult.Code = consensuspb.VerifyResult_FAIL
	}
	return verifyResult
}

// validateBlock, validate block and transactions
func (v *BlockVerifierImpl) validateBlock(block *commonpb.Block) (map[string]*commonpb.TxRWSet, map[string][]*commonpb.ContractEvent, []int64, error) {
	hashType := v.chainConf.ChainConfig().Crypto.Hash
	timeLasts := make([]int64, 0)
	var err error
	var lastBlock *commonpb.Block
	txCapacity := int64(v.chainConf.ChainConfig().Block.BlockTxCapacity)
	if block.Header.TxCount > txCapacity {
		return nil, nil, timeLasts, fmt.Errorf("txcapacity expect <= %d, got %d)", txCapacity, block.Header.TxCount)
	}

	if err = v.blockValidator.IsTxCountValid(block); err != nil {
		return nil, nil, timeLasts, err
	}

	lastBlock, err = v.fetchLastBlock(block, lastBlock)
	if err != nil {
		return nil, nil, timeLasts, err
	}
	// proposed height == proposing height - 1
	proposedHeight := lastBlock.Header.BlockHeight
	// check if this block height is 1 bigger than last block height
	lastBlockHash := lastBlock.Header.BlockHash
	err = v.checkPreBlock(block, lastBlock, err, lastBlockHash, proposedHeight)
	if err != nil {
		return nil, nil, timeLasts, err
	}

	if err = v.blockValidator.IsBlockHashValid(block); err != nil {
		return nil, nil, timeLasts, err
	}

	// verify block sig and also verify identity and auth of block proposer
	startSigTick := utils.CurrentTimeMillisSeconds()

	v.log.Debugf("verify block \n %s", utils.FormatBlock(block))
	if ok, err := utils.VerifyBlockSig(hashType, block, v.ac); !ok || err != nil {
		return nil, nil, timeLasts, fmt.Errorf("(%d,%x - %x,%x) [signature]",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.Proposer, block.Header.Signature)
	}
	sigLasts := utils.CurrentTimeMillisSeconds() - startSigTick
	timeLasts = append(timeLasts, sigLasts)

	err = v.checkVacuumBlock(block)
	if err != nil {
		return nil, nil, timeLasts, err
	}
	if len(block.Txs) == 0 {
		return nil, nil, timeLasts, nil
	}

	// verify if txs are duplicate in this block
	if v.blockValidator.IsTxDuplicate(block.Txs) {
		err := fmt.Errorf("tx duplicate")
		return nil, nil, timeLasts, err
	}

	// simulate with DAG, and verify read write set
	startVMTick := utils.CurrentTimeMillisSeconds()
	snapshot := v.snapshotManager.NewSnapshot(lastBlock, block)
	txRWSetMap, txResultMap, err := v.txScheduler.SimulateWithDag(block, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - startVMTick
	timeLasts = append(timeLasts, vmLasts)
	if err != nil {
		return nil, nil, timeLasts, fmt.Errorf("simulate %s", err)
	}
	if block.Header.TxCount != int64(len(txRWSetMap)) {
		err = fmt.Errorf("simulate txcount expect %d, got %d", block.Header.TxCount, len(txRWSetMap))
		return nil, nil, timeLasts, err
	}

	// 2.transaction verify
	startTxTick := utils.CurrentTimeMillisSeconds()
	txHashes, _, errTxs, err := v.txValidator.VerifyTxs(block, txRWSetMap, txResultMap)
	txLasts := utils.CurrentTimeMillisSeconds() - startTxTick
	timeLasts = append(timeLasts, txLasts)
	if err != nil {
		// verify failed, need to put transactions back to txpool
		if len(errTxs) > 0 {
			v.log.Warn("[Duplicate txs] delete the err txs")
			v.txPool.RetryAndRemoveTxs(nil, errTxs)
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
	err = v.checkBlockDigests(block, txHashes, hashType)
	if err != nil {
		return txRWSetMap, contractEventMap, timeLasts, err
	}
	rootsLast := utils.CurrentTimeMillisSeconds() - startRootsTick
	timeLasts = append(timeLasts, rootsLast)

	return txRWSetMap, contractEventMap, timeLasts, nil
}

func (v *BlockVerifierImpl) checkVacuumBlock(block *commonpb.Block) error {
	if 0 == block.Header.TxCount {
		if utils.CanProposeEmptyBlock(v.chainConf.ChainConfig().Consensus.Type) {
			// for consensus that allows empty block, skip txs verify
			return nil
		} else {
			// for consensus that NOT allows empty block, return error
			return fmt.Errorf("tx must not empty")
		}
	}
	return nil
}

func (v *BlockVerifierImpl) checkBlockDigests(block *commonpb.Block, txHashes [][]byte, hashType string) error {
	if err := v.blockValidator.IsMerkleRootValid(block, txHashes); err != nil {
		v.log.Error(err)
		return err
	}
	// verify DAG hash
	if err := v.blockValidator.IsDagHashValid(block); err != nil {
		v.log.Error(err)
		return err
	}
	// verify read write set, check if simulate result is equal with rwset in block header
	if err := v.blockValidator.IsRWSetHashValid(block); err != nil {
		v.log.Error(err)
		return err
	}
	return nil
}

func (v *BlockVerifierImpl) checkPreBlock(block *commonpb.Block, lastBlock *commonpb.Block, err error,
	lastBlockHash []byte, proposedHeight int64) error {
	if consensuspb.ConsensusType_HOTSTUFF != v.chainConf.ChainConfig().Consensus.Type {
		if err = v.blockValidator.IsHeightValid(block, proposedHeight); err != nil {
			return err
		}
		// check if this block pre hash is equal with last block hash
		return v.blockValidator.IsPreHashValid(block, lastBlockHash)
	}

	if block.Header.BlockHeight == lastBlock.Header.BlockHeight+1 {
		if err := v.blockValidator.IsPreHashValid(block, lastBlock.Header.BlockHash); err != nil {
			return err
		}
	} else {
		// for chained bft consensus type
		proposedBlock, _ := v.proposalCache.GetProposedBlockByHashAndHeight(block.Header.PreBlockHash, block.Header.BlockHeight-1)
		if proposedBlock == nil {
			return fmt.Errorf("no last block found [%d](%x) %s", block.Header.BlockHeight-1, block.Header.PreBlockHash, err)
		}
	}

	// remove unconfirmed block from proposal cache and txpool
	cutBlocks := v.proposalCache.KeepProposedBlock(lastBlockHash, lastBlock.Header.BlockHeight)
	if len(cutBlocks) > 0 {
		v.log.Infof("cut block block hash: %s, height: %v", hex.EncodeToString(lastBlockHash), lastBlock.Header.BlockHeight)
		v.cutBlocks(cutBlocks, lastBlock)
	}
	return nil
}

func (v *BlockVerifierImpl) cutBlocks(blocksToCut []*commonpb.Block, blockToKeep *commonpb.Block) {
	cutTxs := make([]*commonpb.Transaction, 0)
	txMap := make(map[string]interface{})
	for _, tx := range blockToKeep.Txs {
		txMap[tx.Header.TxId] = struct{}{}
	}
	for _, blockToCut := range blocksToCut {
		v.log.Infof("cut block block hash: %s, height: %v", blockToCut.Header.BlockHash, blockToCut.Header.BlockHeight)
		for _, txToCut := range blockToCut.Txs {
			if _, ok := txMap[txToCut.Header.TxId]; ok {
				// this transaction is kept, do NOT cut it.
				continue
			}
			v.log.Infof("cut tx hash: %s", txToCut.Header.TxId)
			cutTxs = append(cutTxs, txToCut)
		}
	}
	if len(cutTxs) > 0 {
		v.txPool.RetryAndRemoveTxs(cutTxs, nil)
	}
}

func (v *BlockVerifierImpl) fetchLastBlock(block *commonpb.Block, lastBlock *commonpb.Block) (*commonpb.Block, error) {
	currentHeight, _ := v.ledgerCache.CurrentHeight()
	if currentHeight >= block.Header.BlockHeight {
		return nil, commonErrors.ErrBlockHadBeenCommited
	}

	if currentHeight+1 == block.Header.BlockHeight {
		lastBlock = v.ledgerCache.GetLastCommittedBlock()
	} else {
		lastBlock, _ = v.proposalCache.GetProposedBlockByHashAndHeight(block.Header.PreBlockHash, block.Header.BlockHeight-1)
	}
	if lastBlock == nil {
		return nil, fmt.Errorf("no pre block found [%d](%x)", block.Header.BlockHeight-1, block.Header.PreBlockHash)
	}
	return lastBlock, nil
}
