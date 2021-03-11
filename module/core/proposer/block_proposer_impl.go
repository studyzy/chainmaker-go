/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	"bytes"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/monitor"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/prometheus/client_golang/prometheus"

	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/logger"
	txpoolpb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"chainmaker.org/chainmaker-go/protocol"
)

// BlockProposerImpl implements BlockProposer interface.
// In charge of propose a new block.
type BlockProposerImpl struct {
	chainId string // chain id, to identity this chain

	txPool          protocol.TxPool          // tx pool provides tx batch
	txScheduler     protocol.TxScheduler     // scheduler orders tx batch into DAG form and returns a block
	snapshotManager protocol.SnapshotManager // snapshot manager
	identity        protocol.SigningMember   // identity manager
	ledgerCache     protocol.LedgerCache     // ledger cache
	msgBus          msgbus.MessageBus        // channel to give out proposed block
	ac              protocol.AccessControlProvider
	blockchainStore protocol.BlockchainStore

	isProposer   bool        // whether current node can propose block now
	idle         bool        // whether current node is proposing or not
	proposeTimer *time.Timer // timer controls the proposing periods

	canProposeC   chan bool                   // channel to handle propose status change from consensus module
	txPoolSignalC chan *txpoolpb.TxPoolSignal // channel to handle propose signal from tx pool
	exitC         chan bool                   // channel to stop proposing loop
	proposalCache protocol.ProposalCache

	chainConf protocol.ChainConf // chain config

	idleMu         sync.Mutex   // for proposeBlock reentrant lock
	statusMu       sync.Mutex   // for propose status change lock
	proposerMu     sync.RWMutex // for isProposer lock, avoid race
	log            *logger.CMLogger
	finishProposeC chan bool // channel to receive signal to yield propose block

	metricBlockPackageTime *prometheus.HistogramVec
	proposer               []byte // this node identity
}

type BlockProposerConfig struct {
	ChainId         string
	TxPool          protocol.TxPool
	SnapshotManager protocol.SnapshotManager
	MsgBus          msgbus.MessageBus
	Identity        protocol.SigningMember
	LedgerCache     protocol.LedgerCache
	TxScheduler     protocol.TxScheduler
	ProposalCache   protocol.ProposalCache
	ChainConf       protocol.ChainConf
	AC              protocol.AccessControlProvider
	BlockchainStore protocol.BlockchainStore
}

const (
	DEFAULTDURATION = 1000     // default proposal duration, millis seconds
	DEFAULTVERSION  = "v1.0.0" // default version of chain
)

func NewBlockProposer(config BlockProposerConfig) (protocol.BlockProposer, error) {
	blockProposerImpl := &BlockProposerImpl{
		chainId:         config.ChainId,
		isProposer:      false, // not proposer when initialized
		idle:            true,
		msgBus:          config.MsgBus,
		blockchainStore: config.BlockchainStore,
		canProposeC:     make(chan bool),
		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
		exitC:           make(chan bool),
		txPool:          config.TxPool,
		snapshotManager: config.SnapshotManager,
		txScheduler:     config.TxScheduler,
		identity:        config.Identity,
		ledgerCache:     config.LedgerCache,
		proposalCache:   config.ProposalCache,
		chainConf:       config.ChainConf,
		ac:              config.AC,
		log:             logger.GetLoggerByChain(logger.MODULE_CORE, config.ChainId),
		finishProposeC:  make(chan bool),
	}

	var err error
	blockProposerImpl.proposer, err = blockProposerImpl.identity.Serialize(true)
	if err != nil {
		blockProposerImpl.log.Warnf("identity serialize failed, %s", err)
		return nil, err
	}

	// start propose timer
	blockProposerImpl.proposeTimer = time.NewTimer(blockProposerImpl.getDuration())
	if !blockProposerImpl.isSelfProposer() {
		blockProposerImpl.proposeTimer.Stop()
	}

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		blockProposerImpl.metricBlockPackageTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_PROPOSER, "metric_block_package_time",
			"block package time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}

	return blockProposerImpl, nil
}

// Start, start proposer
func (bp *BlockProposerImpl) Start() error {
	defer bp.log.Info("block proposer starts")

	go bp.startProposingLoop()

	return nil
}

// Stop, stop proposing loop
func (bp *BlockProposerImpl) Stop() error {
	defer bp.log.Infof("block proposer stoped")
	bp.exitC <- true
	return nil
}

// Start, start proposing loop
func (bp *BlockProposerImpl) startProposingLoop() {
	for {
		select {
		case <-bp.proposeTimer.C:
			if !bp.isSelfProposer() {
				break
			}
			go bp.proposeBlock()

		case signal := <-bp.txPoolSignalC:
			if !bp.isSelfProposer() {
				break
			}
			if signal.SignalType != txpoolpb.SignalType_BLOCK_PROPOSE {
				break
			}
			go bp.proposeBlock()
			bp.log.Infof("trigger proposal from signal, height[%d], signal %d", bp.ledgerCache.GetLastCommittedBlock().Header.BlockHeight, signal.SignalType)

		case <-bp.exitC:
			bp.proposeTimer.Stop()
			bp.log.Info("block proposer loop stoped")
			return
		}
	}
}

/*
 * shouldProposeByBFT, check if node should propose new block
 * Only for *BFT consensus
 * if node is proposer, and node is not propose right now, and last proposed block is committed, then return true
 */
func (bp *BlockProposerImpl) shouldProposeByBFT(height int64) bool {
	if !bp.isIdle() {
		// concurrent control, proposer is proposing now
		bp.log.Debugf("proposer is busy, not propose [%d] ", height)
		return false
	}
	committedBlock := bp.ledgerCache.GetLastCommittedBlock()
	if committedBlock == nil {
		bp.log.Errorf("no committed block found")
		return false
	}
	currentHeight := committedBlock.Header.BlockHeight
	// proposing height must higher than current height
	if currentHeight+1 != height {
		return false
	}
	if bp.proposalCache.IsProposedAt(height) {
		// this node is proposer and has proposed at this round before
		bp.log.Debugf("proposer has proposed at [%d] ", height)
		return false
	}
	return true
}

// proposeBlock, to check if proposer can propose block right now
// if so, start proposing
func (bp *BlockProposerImpl) proposeBlock() {
	defer func() {
		if bp.isSelfProposer() {
			bp.proposeTimer.Reset(bp.getDuration())
		}
	}()
	lastBlock := bp.ledgerCache.GetLastCommittedBlock()
	proposingHeight := lastBlock.Header.BlockHeight + 1
	if !bp.shouldProposeByBFT(proposingHeight) {
		return
	}
	if !bp.setNotIdle() {
		bp.log.Infof("concurrent propose block [%d], yield!", proposingHeight)
		return
	}
	defer bp.setIdle()

	go bp.proposing(proposingHeight, lastBlock.Header.BlockHash)
	// #DEBUG MODE#
	if localconf.ChainMakerConfig.DebugConfig.IsHaltPropose {
		go func() {
			bp.OnReceiveYieldProposeSignal(true)
		}()
	}

	<-bp.finishProposeC
}

// proposing, propose a block in new height
func (bp *BlockProposerImpl) proposing(height int64, preHash []byte) *commonpb.Block {
	startTick := utils.CurrentTimeMillisSeconds()
	defer bp.yieldProposing()

	selfProposedBlock := bp.proposalCache.GetSelfProposedBlockAt(height)
	if selfProposedBlock != nil {
		// Repeat propose block if node has proposed before at the same height
		bp.proposalCache.SetProposedAt(height)
		bp.msgBus.Publish(msgbus.ProposedBlock, selfProposedBlock)
		bp.log.Infof("proposer success repeat [%d](txs:%d,hash:%x)",
			selfProposedBlock.Header.BlockHeight, selfProposedBlock.Header.TxCount, selfProposedBlock.Header.BlockHash)
		return nil
	}

	// retrieve tx batch from tx pool
	startFetchTick := utils.CurrentTimeMillisSeconds()
	checkedBatch := bp.txPool.FetchTxBatch(height)
	fetchLasts := utils.CurrentTimeMillisSeconds() - startFetchTick
	bp.log.Debugf("begin proposing block[%d], fetch tx num[%d]", height, len(checkedBatch))

	startDupTick := utils.CurrentTimeMillisSeconds()
	//checkedBatch := bp.txDuplicateCheck(txBatch)
	dupLasts := utils.CurrentTimeMillisSeconds() - startDupTick
	if !utils.CanProposeEmptyBlock(bp.chainConf.ChainConfig().Consensus.Type) &&
		(checkedBatch == nil || len(checkedBatch) == 0) {
		// can not propose empty block and tx batch is empty, then yield proposing.
		bp.log.Debugf("no txs in tx pool, proposing block stoped")
		return nil
	}

	txCapacity := int(bp.chainConf.ChainConfig().Block.BlockTxCapacity)
	if len(checkedBatch) > txCapacity {
		// check if checkedBatch > txCapacity, if so, strict block tx count according to  config, and put other txs back to txpool.
		txRetry := checkedBatch[txCapacity:]
		checkedBatch = checkedBatch[:txCapacity]
		bp.txPool.RetryAndRemoveTxs(txRetry, nil)
		bp.log.Warnf("txbatch oversize expect <= %d, got %d", txCapacity, len(checkedBatch))
	}

	block, timeLasts, err := bp.generateNewBlock(height, preHash, checkedBatch)
	if err != nil {
		bp.txPool.RetryAndRemoveTxs(checkedBatch, nil) // put txs back to txpool
		bp.log.Warnf("generate new block failed, %s", err.Error())
		return nil
	}
	bp.msgBus.Publish(msgbus.ProposedBlock, block)
	//bp.log.Debugf("finalized block \n%s", utils.FormatBlock(block))
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bp.log.Infof("proposer success [%d](txs:%d), time used(fetch:%d,dup:%d,vm:%v,total:%d)",
		block.Header.BlockHeight, block.Header.TxCount,
		fetchLasts, dupLasts, timeLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		bp.metricBlockPackageTime.WithLabelValues(bp.chainId).Observe(float64(elapsed) / 1000)
	}
	return block
}

// txDuplicateCheck, to check if transactions that are about to proposing are double spenting.
func (bp *BlockProposerImpl) txDuplicateCheck(batch []*commonpb.Transaction) []*commonpb.Transaction {
	if batch == nil || len(batch) == 0 {
		return nil
	}
	checked := make([]*commonpb.Transaction, 0)
	verifyBatchs := utils.DispatchTxVerifyTask(batch)
	results := make([][]*commonpb.Transaction, 0)
	workerCount := len(verifyBatchs)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func(index int, b []*commonpb.Transaction) {
			defer wg.Done()
			result := make([]*commonpb.Transaction, 0)
			for _, tx := range b {
				exist, err := bp.blockchainStore.TxExists(tx.Header.TxId)
				if err == nil && !exist {
					result = append(result, tx)
				}
			}
			results[index] = result
		}(i, verifyBatchs[i])
	}
	wg.Wait()
	for _, result := range results {
		checked = append(checked, result...)
	}
	return checked
}

// OnReceiveTxPoolSignal, receive txpool signal and deliver to chan txpool signal
func (bp *BlockProposerImpl) OnReceiveTxPoolSignal(txPoolSignal *txpoolpb.TxPoolSignal) {
	bp.txPoolSignalC <- txPoolSignal
}

/*
 * OnReceiveProposeStatusChange, to update isProposer status when received proposeStatus from consensus
 * if node is proposer, then reset the timer, otherwise stop the timer
 */
func (bp *BlockProposerImpl) OnReceiveProposeStatusChange(proposeStatus bool) {
	bp.log.Debugf("OnReceiveProposeStatusChange(%t)", proposeStatus)
	bp.statusMu.Lock()
	defer bp.statusMu.Unlock()
	if proposeStatus == bp.isSelfProposer() {
		// 状态一致，忽略
		return
	}
	height, _ := bp.ledgerCache.CurrentHeight()
	bp.proposalCache.ResetProposedAt(height) // proposer status changed, reset this round proposed status
	bp.setIsSelfProposer(proposeStatus)
	if !bp.isSelfProposer() {
		bp.yieldProposing() // try to yield if proposer self is proposing right now.
		bp.log.Debug("current node is not proposer ")
		return
	} else {
		bp.proposeTimer.Reset(bp.getDuration())
		bp.log.Debugf("current node is proposer, timeout period is %v", bp.getDuration())
	}
}

// OnReceiveChainedBFTProposal, to check if this proposer should propose a new block
// Only for chained bft consensus
func (bp *BlockProposerImpl) OnReceiveChainedBFTProposal(_ *interface{}) {

}

// OnReceiveYieldProposeSignal, receive yield propose signal
func (bp *BlockProposerImpl) OnReceiveYieldProposeSignal(isYield bool) {
	if !isYield {
		return
	}
	if bp.yieldProposing() {
		// halt scheduler execution
		bp.txScheduler.Halt()
		height, _ := bp.ledgerCache.CurrentHeight()
		bp.proposalCache.ResetProposedAt(height)
	}
}

// yieldProposing, to yield proposing handle
func (bp *BlockProposerImpl) yieldProposing() bool {
	// signal finish propose only if proposer is not idle
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	if !bp.idle {
		bp.finishProposeC <- true
		bp.idle = true
		return true
	}
	return false
}

// getDuration, get propose duration from config.
// If not access from config, use default value.
func (bp *BlockProposerImpl) getDuration() time.Duration {
	if bp.chainConf == nil || bp.chainConf.ChainConfig() == nil {
		return DEFAULTDURATION * time.Millisecond
	}
	chainConfig := bp.chainConf.ChainConfig()
	duration := chainConfig.Block.BlockInterval
	if duration <= 0 {
		return DEFAULTDURATION * time.Millisecond
	} else {
		return time.Duration(duration) * time.Millisecond
	}
}

// getChainVersion, get chain version from config.
// If not access from config, use default value.
func (bp *BlockProposerImpl) getChainVersion() []byte {
	if bp.chainConf == nil || bp.chainConf.ChainConfig() == nil {
		return []byte(DEFAULTVERSION)
	}
	return []byte(bp.chainConf.ChainConfig().Version)
}

// setNotIdle, set not idle status
func (bp *BlockProposerImpl) setNotIdle() bool {
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	if bp.idle {
		bp.idle = false
		return true
	} else {
		return false
	}
}

// isIdle, to check if proposer is idle
func (bp *BlockProposerImpl) isIdle() bool {
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	return bp.idle
}

// setIdle, set idle status
func (bp *BlockProposerImpl) setIdle() {
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	bp.idle = true
}

// setIsSelfProposer, set isProposer status of this node
func (bp *BlockProposerImpl) setIsSelfProposer(isSelfProposer bool) {
	bp.proposerMu.Lock()
	defer bp.proposerMu.Unlock()
	bp.isProposer = isSelfProposer
	if !bp.isProposer {
		bp.proposeTimer.Stop()
	} else {
		bp.proposeTimer.Reset(bp.getDuration())
	}
}

// isSelfProposer, return if this node is consensus proposer
func (bp *BlockProposerImpl) isSelfProposer() bool {
	bp.proposerMu.RLock()
	defer bp.proposerMu.RUnlock()
	return bp.isProposer
}

/*
 * shouldProposeByChainedBFT, check if node should propose new block
 * Only for chained bft consensus
 */
func (bp *BlockProposerImpl) shouldProposeByChainedBFT(height int64, preHash []byte) bool {
	committedBlock := bp.ledgerCache.GetLastCommittedBlock()
	if committedBlock == nil {
		bp.log.Errorf("no committed block found")
		return false
	}
	currentHeight := committedBlock.Header.BlockHeight
	// proposing height must higher than current height
	if currentHeight >= height {
		return false
	}
	if height == currentHeight+1 {
		// height follows the last committed block
		if bytes.Equal(committedBlock.Header.BlockHash, preHash) {
			return true
		} else {
			bp.log.Errorf("block pre hash error, expect %x, got %x, can not propose",
				committedBlock.Header.BlockHash, preHash)
			return false
		}
	}
	// if height not follows the last committed block, then check last proposed block
	b, _ := bp.proposalCache.GetProposedBlockByHashAndHeight(preHash, height-1)
	return b != nil
}
