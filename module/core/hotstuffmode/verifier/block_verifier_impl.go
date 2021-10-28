/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"encoding/hex"
	"fmt"
	"sync"

	"chainmaker.org/chainmaker-go/consensus"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// BlockVerifierImpl implements BlockVerifier interface.
// Verify block and transactions.
//nolint: structcheck,unused
type BlockVerifierImpl struct {
	chainId         string                   // chain id, to identity this chain
	msgBus          msgbus.MessageBus        // message bus
	txScheduler     protocol.TxScheduler     // scheduler orders tx batch into DAG form and returns a block
	snapshotManager protocol.SnapshotManager // snapshot manager
	ledgerCache     protocol.LedgerCache     // ledger cache
	blockchainStore protocol.BlockchainStore // blockchain store

	reentrantLocks *common.ReentrantLocks         // reentrant lock for avoid concurrent verify block
	proposalCache  protocol.ProposalCache         // proposal cache
	chainConf      protocol.ChainConf             // chain config
	ac             protocol.AccessControlProvider // access control manager
	log            protocol.Logger                // logger
	txPool         protocol.TxPool                // tx pool to check if tx is duplicate
	mu             sync.Mutex                     // to avoid concurrent map modify
	verifierBlock  *common.VerifierBlock
	storeHelper    conf.StoreHelper

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
	VmMgr           protocol.VmManager
	StoreHelper     conf.StoreHelper
}

func NewBlockVerifier(config BlockVerifierConfig, log protocol.Logger) (protocol.BlockVerifier, error) {
	v := &BlockVerifierImpl{
		chainId:         config.ChainId,
		msgBus:          config.MsgBus,
		txScheduler:     config.TxScheduler,
		snapshotManager: config.SnapshotManager,
		ledgerCache:     config.LedgerCache,
		blockchainStore: config.BlockchainStore,
		reentrantLocks: &common.ReentrantLocks{
			ReentrantLocks: make(map[string]interface{}),
		},
		proposalCache: config.ProposedCache,
		chainConf:     config.ChainConf,
		ac:            config.AC,
		log:           log,
		txPool:        config.TxPool,
		storeHelper:   config.StoreHelper,
	}

	conf := &common.VerifierBlockConf{
		ChainConf:       v.chainConf,
		Log:             v.log,
		LedgerCache:     v.ledgerCache,
		Ac:              v.ac,
		SnapshotManager: v.snapshotManager,
		TxPool:          v.txPool,
		BlockchainStore: v.blockchainStore,
		ProposalCache:   v.proposalCache,
		VmMgr:           config.VmMgr,
		StoreHelper:     config.StoreHelper,
		TxScheduler:     config.TxScheduler,
	}
	v.verifierBlock = common.NewVerifierBlock(conf)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}

	return v, nil
}

// VerifyBlock, to check if block is valid
func (v *BlockVerifierImpl) VerifyBlock(block *commonpb.Block, mode protocol.VerifyMode) (err error) {
	startTick := utils.CurrentTimeMillisSeconds()
	if err = utils.IsEmptyBlock(block); err != nil {
		v.log.Error(err)
		return err
	}

	v.log.Debugf("verify receive [%d](%x,%d,%d), from sync %d",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs), mode)
	// avoid concurrent verify, only one block hash can be verified at the same time
	if !v.reentrantLocks.Lock(string(block.Header.BlockHash)) {
		v.log.Warnf("block(%d,%x) concurrent verify, yield", block.Header.BlockHeight, block.Header.BlockHash)
		return commonErrors.ErrConcurrentVerify
	}
	defer v.reentrantLocks.Unlock(string(block.Header.BlockHash))

	var isValid bool
	var contractEventMap map[string][]*commonpb.ContractEvent
	// to check if the block has verified before
	b, txRwSet, eventMap := v.proposalCache.GetProposedBlock(block)
	//contractEventMap = eventMap

	notSolo := consensuspb.ConsensusType_SOLO != v.chainConf.ChainConfig().Consensus.Type
	if b != nil {
		isSqlDb := v.chainConf.ChainConfig().Contract.EnableSqlSupport
		if notSolo || isSqlDb {
			elapsed := utils.CurrentTimeMillisSeconds() - startTick
			// the block has verified before
			v.log.Infof("verify success repeat [%d](%x), total: %d", block.Header.BlockHeight, block.Header.BlockHash, elapsed)
			isValid = true
			if protocol.CONSENSUS_VERIFY == mode {
				// consensus mode, publish verify result to message bus
				v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid, txRwSet))
			}
			lastBlock, _ := v.proposalCache.GetProposedBlockByHashAndHeight(
				block.Header.PreBlockHash, block.Header.BlockHeight-1)
			if lastBlock == nil {
				v.log.Debugf(
					"no pre-block be found, preHeight:%d, preBlockHash:%x",
					block.Header.BlockHeight-1,
					block.Header.PreBlockHash,
				)
				return nil
			}
			cutBlocks := v.proposalCache.KeepProposedBlock(lastBlock.Header.BlockHash, lastBlock.Header.BlockHeight)
			if len(cutBlocks) > 0 {
				v.log.Infof(
					"cut block block hash: %s, height: %v",
					hex.EncodeToString(lastBlock.Header.BlockHash),
					lastBlock.Header.BlockHeight,
				)
				v.cutBlocks(cutBlocks, lastBlock)
			}
			err = v.proposalCache.SetProposedBlock(
				block, txRwSet, eventMap, v.proposalCache.IsProposedAt(block.Header.BlockHeight))
			return err
		}
	}

	// avoid to recover the committed block.
	lastBlock, err := v.verifierBlock.FetchLastBlock(block)
	if err != nil {
		return err
	}

	startPoolTick := utils.CurrentTimeMillisSeconds()
	newBlock, err := common.RecoverBlock(block, mode, v.chainConf, v.txPool, v.log)
	if err != nil {
		return err
	}
	lastPool := utils.CurrentTimeMillisSeconds() - startPoolTick

	txRWSetMap, contractEventMap, timeLasts, err := v.validateBlock(newBlock, lastBlock)
	if err != nil {
		v.log.Warnf("verify failed [%d](%x),preBlockHash:%x, %s",
			newBlock.Header.BlockHeight, newBlock.Header.BlockHash, newBlock.Header.PreBlockHash, err.Error())
		if protocol.CONSENSUS_VERIFY == mode {
			v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(newBlock, isValid, txRWSetMap))
		}

		// rollback sql
		if sqlErr := v.storeHelper.RollBack(newBlock, v.blockchainStore); sqlErr != nil {
			v.log.Errorf("block [%d] rollback sql failed: %s", newBlock.Header.BlockHeight, sqlErr)
		}
		return err
	}

	// sync mode, need to verify consensus vote signature
	beginConsensCheck := utils.CurrentTimeMillisSeconds()
	if protocol.SYNC_VERIFY == mode {
		if err = v.verifyVoteSig(newBlock); err != nil {
			v.log.Warnf("verify failed [%d](%x), votesig %s",
				newBlock.Header.BlockHeight, newBlock.Header.BlockHash, err.Error())
			return err
		}
	}
	consensusCheckUsed := utils.CurrentTimeMillisSeconds() - beginConsensCheck

	if notSolo {
		// verify success, cache block and read write set
		v.log.Debugf("set proposed block(%d,%x)", newBlock.Header.BlockHeight, newBlock.Header.BlockHash)
		if err = v.proposalCache.SetProposedBlock(newBlock, txRWSetMap, contractEventMap, false); err != nil {
			return err
		}
	}

	// mark transactions in block as pending status in txpool
	v.txPool.AddTxsToPendingCache(newBlock.Txs, newBlock.Header.BlockHeight)

	isValid = true
	if protocol.CONSENSUS_VERIFY == mode {
		v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(newBlock, isValid, txRWSetMap))
	}
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	v.log.Infof("verify success [%d,%x](%v,pool: %d,consensusCheckUsed: %d, total: %d)", newBlock.Header.BlockHeight,
		newBlock.Header.BlockHash, timeLasts, lastPool, consensusCheckUsed, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime.WithLabelValues(v.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}

func (v *BlockVerifierImpl) validateBlock(block, lastBlock *commonpb.Block) (map[string]*commonpb.TxRWSet,
	map[string][]*commonpb.ContractEvent, []int64, error) {
	hashType := v.chainConf.ChainConfig().Crypto.Hash
	timeLasts := make([]int64, 0)
	var err error
	txCapacity := v.chainConf.ChainConfig().Block.BlockTxCapacity
	if block.Header.TxCount > txCapacity {
		return nil, nil, timeLasts, fmt.Errorf("txcapacity expect <= %d, got %d)", txCapacity, block.Header.TxCount)
	}

	if err = common.IsTxCountValid(block); err != nil {
		return nil, nil, timeLasts, err
	}

	// proposed height == proposing height - 1
	proposedHeight := lastBlock.Header.BlockHeight
	// check if this block height is 1 bigger than last block height
	lastBlockHash := lastBlock.Header.BlockHash
	err = v.checkPreBlock_HOTSTUFF(block, lastBlock, err, lastBlockHash, proposedHeight)
	if err != nil {
		return nil, nil, timeLasts, err
	}

	return v.verifierBlock.ValidateBlock(block, lastBlock, hashType, timeLasts)
}

func (v *BlockVerifierImpl) checkPreBlock_HOTSTUFF(block *commonpb.Block, lastBlock *commonpb.Block, err error,
	lastBlockHash []byte, proposedHeight uint64) error {

	if block.Header.BlockHeight == lastBlock.Header.BlockHeight+1 {
		if err = common.IsPreHashValid(block, lastBlock.Header.BlockHash); err != nil {
			return err
		}
	} else {
		// for chained bft consensus type
		proposedBlock, _ := v.proposalCache.GetProposedBlockByHashAndHeight(
			block.Header.PreBlockHash, block.Header.BlockHeight-1)
		if proposedBlock == nil {
			return fmt.Errorf(
				"no last block found [%d](%x) %s",
				block.Header.BlockHeight-1,
				block.Header.PreBlockHash,
				err,
			)
		}
	}

	// remove unconfirmed block from proposal cache and txpool
	cutBlocks := v.proposalCache.KeepProposedBlock(lastBlockHash, lastBlock.Header.BlockHeight)
	if len(cutBlocks) > 0 {
		cutTxs := make([]*commonpb.Transaction, 0)
		for _, cutBlock := range cutBlocks {
			cutTxs = append(cutTxs, cutBlock.Txs...)
		}
		v.txPool.RetryAndRemoveTxs(cutTxs, nil)
	}
	return nil
}

func (v *BlockVerifierImpl) verifyVoteSig(block *commonpb.Block) error {
	return consensus.VerifyBlockSignatures(v.chainConf, v.ac, v.blockchainStore, block, v.ledgerCache)
}

func parseVerifyResult(block *commonpb.Block, isValid bool,
	txsRwSet map[string]*commonpb.TxRWSet) *consensuspb.VerifyResult {
	verifyResult := &consensuspb.VerifyResult{
		VerifiedBlock: block,
		TxsRwSet:      txsRwSet,
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

func (v *BlockVerifierImpl) cutBlocks(blocksToCut []*commonpb.Block, blockToKeep *commonpb.Block) {
	cutTxs := make([]*commonpb.Transaction, 0)
	txMap := make(map[string]interface{})
	for _, tx := range blockToKeep.Txs {
		txMap[tx.Payload.TxId] = struct{}{}
	}
	for _, blockToCut := range blocksToCut {
		v.log.Infof("cut block block hash: %s, height: %v", blockToCut.Header.BlockHash, blockToCut.Header.BlockHeight)
		for _, txToCut := range blockToCut.Txs {
			if _, ok := txMap[txToCut.Payload.TxId]; ok {
				// this transaction is kept, do NOT cut it.
				continue
			}
			v.log.Debugf("cut tx hash: %s", txToCut.Payload.TxId)
			cutTxs = append(cutTxs, txToCut)
		}
	}
	if len(cutTxs) > 0 {
		v.txPool.RetryAndRemoveTxs(cutTxs, nil)
	}
}
