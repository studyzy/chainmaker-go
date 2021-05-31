/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package committer

import (
	"bytes"
	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/monitor"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/statedb/statesqldb"
	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

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
	log                   protocol.Logger            // logger
	msgBus                msgbus.MessageBus           // message bus
	mu                    sync.Mutex                  // lock, to avoid concurrent block commit
	subscriber            *subscriber.EventSubscriber // subscriber
	verifier              protocol.BlockVerifier      // block verifier
	commonCommit          *common.CommitBlock
	metricBlockSize       *prometheus.HistogramVec // metric block size
	metricBlockCounter    *prometheus.CounterVec   // metric block counter
	metricTxCounter       *prometheus.CounterVec   // metric transaction counter
	metricBlockCommitTime *prometheus.HistogramVec // metric block commit time
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

	cbConf := &common.CommitBlockConf{
		Store:                 config.BlockchainStore,
		Log:                   blockchain.log,
		SnapshotManager:       config.SnapshotManager,
		TxPool:                config.TxPool,
		LedgerCache:           config.LedgerCache,
		ChainConf:             config.ChainConf,
		MsgBus:                config.MsgBus,
		MetricBlockCommitTime: blockchain.metricBlockCommitTime,
		MetricBlockCounter:    blockchain.metricBlockCounter,
		MetricBlockSize:       blockchain.metricBlockSize,
		MetricTxCounter:       blockchain.metricTxCounter,
	}
	blockchain.commonCommit = common.NewCommitBlock(cbConf)

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
		if err == nil {
			return
		}
		// rollback sql
		chain.log.Error(err)
		if chain.chainConf.ChainConfig().Contract.EnableSqlSupport {
			txKey := block.GetTxKey()
			_ = chain.blockchainStore.RollbackDbTransaction(txKey)
			// drop database if create contract fail
			if len(block.Txs) == 0 && utils.IsManageContractAsConfigTx(block.Txs[0], true) {
				var payload commonpb.ContractMgmtPayload
				if err := proto.Unmarshal(block.Txs[0].RequestPayload, &payload); err == nil {
					if payload.ContractId != nil {
						dbName := statesqldb.GetContractDbName(chain.chainId, payload.ContractId.ContractName)
						chain.blockchainStore.ExecDdlSql(payload.ContractId.ContractName, "drop database "+dbName)
					}
				}
			}
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
