/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	netPb "chainmaker.org/chainmaker-go/pb/protogo/net"
	txpoolPb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/txpool/poolconf"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
)

var _ protocol.TxPool = (*txPoolImpl)(nil)

type txPoolImpl struct {
	chainId string

	queue          *txQueue            // the queue for store transactions
	cache          *txCache            // the cache to temporarily cache transactions
	addTxsCh       chan *mempoolTxs    // channel that receive the common transactions
	stopCh         chan struct{}       // the channel signal that stop the service
	stopAtomic     int64               // the flag that identifies whether the service has closed
	flushTicker    int                 // ticker to check whether the cache needs to be refreshed
	signalLock     sync.RWMutex        // Locker to protect signal status
	signalStatus   txpoolPb.SignalType // The current state of the transaction pool
	latestFullTime int64               // The most latest time the trading pool was full

	ac              protocol.AccessControlProvider
	log             *logger.CMLogger
	msgBus          msgbus.MessageBus        // Information interaction between modules
	chainConf       protocol.ChainConf       // chainConfig
	netService      protocol.NetService      // P2P module implementation
	blockchainStore protocol.BlockchainStore // Store module implementation
}

func NewTxPoolImpl(chainId string, blockStore protocol.BlockchainStore, msgBus msgbus.MessageBus,
	conf protocol.ChainConf, ac protocol.AccessControlProvider, net protocol.NetService) (protocol.TxPool, error) {
	if len(chainId) == 0 {
		return nil, fmt.Errorf("no chainId in create txpool")
	}

	var (
		ticker    = DefaultFlushTicker
		addChSize = DefaultChannelSize
		log       = logger.GetLoggerByChain(logger.MODULE_TXPOOL, chainId)
	)
	if localconf.ChainMakerConfig.TxPoolConfig.AddTxChannelSize > 0 {
		addChSize = int(localconf.ChainMakerConfig.TxPoolConfig.AddTxChannelSize)
	}
	if localconf.ChainMakerConfig.TxPoolConfig.CacheFlushTicker > 0 {
		ticker = int(localconf.ChainMakerConfig.TxPoolConfig.CacheFlushTicker)
	}
	txPoolQueue := &txPoolImpl{
		chainId:      chainId,
		cache:        newTxCache(),
		stopCh:       make(chan struct{}),
		addTxsCh:     make(chan *mempoolTxs, addChSize),
		flushTicker:  ticker,
		signalStatus: txpoolPb.SignalType_NO_EVENT,

		ac:              ac,
		log:             log,
		msgBus:          msgBus,
		chainConf:       conf,
		netService:      net,
		blockchainStore: blockStore,
	}
	txPoolQueue.queue = newQueue(blockStore, log, txPoolQueue.validate)
	return txPoolQueue, nil
}

func (pool *txPoolImpl) Start() (err error) {
	if pool.msgBus != nil {
		pool.msgBus.Register(msgbus.RecvTxPoolMsg, pool)
	}
	go pool.listen()
	return
}

func (pool *txPoolImpl) listen() {
	flushTicker := time.NewTicker(time.Duration(pool.flushTicker) * time.Second)
	defer flushTicker.Stop()
	for {
		select {
		case poolTxs := <-pool.addTxsCh:
			pool.flushOrAddTxsToCache(poolTxs)
		case <-flushTicker.C:
			if pool.cache.isFlushByTime() && pool.cache.txCount() > 0 {
				pool.flushCommonTxToQueue(nil)
			}
		case <-pool.stopCh:
			return
		}
	}
}

func (pool *txPoolImpl) flushOrAddTxsToCache(memTxs *mempoolTxs) {
	if memTxs == nil || len(memTxs.txs) == 0 {
		return
	}
	defer func() {
		pool.log.Debugf("txPool status: %s, cache txs num: %d", pool.queue.status(), pool.cache.txCount())
	}()

	if memTxs.isConfigTxs {
		pool.flushConfigTxToQueue(memTxs)
		return
	}

	if pool.cache.isFlushByTxCount(memTxs) {
		pool.flushCommonTxToQueue(memTxs)
	} else {
		pool.cache.addMemoryTxs(memTxs)
	}
}

func (pool *txPoolImpl) flushConfigTxToQueue(memTxs *mempoolTxs) {
	defer func() {
		pool.updateAndPublishSignal()
	}()
	pool.queue.addTxsToConfigQueue(memTxs)
}

func (pool *txPoolImpl) flushCommonTxToQueue(memTxs *mempoolTxs) {
	defer func() {
		pool.updateAndPublishSignal()
		pool.cache.reset()
	}()

	rpcTxs, p2pTxs, internalTxs := pool.cache.mergeAndSplitTxsBySource(memTxs)
	pool.queue.addTxsToCommonQueue(&mempoolTxs{txs: rpcTxs, source: protocol.RPC})
	pool.queue.addTxsToCommonQueue(&mempoolTxs{txs: p2pTxs, source: protocol.P2P})
	pool.queue.addTxsToCommonQueue(&mempoolTxs{txs: internalTxs, source: protocol.INTERNAL})
}

func (pool *txPoolImpl) Stop() error {
	if !atomic.CompareAndSwapInt64(&pool.stopAtomic, 0, 1) {
		return fmt.Errorf("txpool service has stoped")
	}
	close(pool.stopCh)
	close(pool.addTxsCh)
	pool.log.Infof("close txpool service")
	return nil
}

func (pool *txPoolImpl) AddTx(tx *commonPb.Transaction, source protocol.TxSource) error {
	if tx == nil {
		return commonErrors.ErrStructEmpty
	}
	if atomic.LoadInt64(&pool.stopAtomic) > 0 {
		pool.log.Info("AddTx TxPool has stopped")
		return errors.New("AddTx TxPool has stopped")
	}
	if source == protocol.INTERNAL {
		return commonErrors.ErrTxSource
	}
	pool.log.Debugw("AddTx", "txId", tx.Header.GetTxId(), "source", source)

	// 1. Determine if the tx pool is full or tx exist
	if pool.isFull(tx) {
		return commonErrors.ErrTxPoolLimit
	}
	if pool.TxExists(tx) {
		return commonErrors.ErrTxIdExist
	}

	// 2. store the transaction
	memTx := &mempoolTxs{isConfigTxs: false, txs: []*commonPb.Transaction{tx}, source: source}
	if utils.IsConfigTx(tx) {
		memTx.isConfigTxs = true
	}
	t := time.NewTimer(time.Second)
	defer t.Stop()
	select {
	case pool.addTxsCh <- memTx:
	case <-t.C:
		pool.log.Warnf("add transaction timeout")
		return fmt.Errorf("add transaction timeout")
	}
	// 3. broadcast the transaction
	if source == protocol.RPC {
		pool.broadcastTx(tx)
	}
	return nil
}

// isFull Check whether the transaction pool is fullnal
func (pool *txPoolImpl) isFull(tx *commonPb.Transaction) bool {
	if utils.IsConfigTx(tx) && pool.queue.configTxsCount() >= poolconf.MaxConfigTxPoolSize() {
		pool.log.Errorw("AddTx configTxPool is full", "txId", tx.Header.GetTxId(), "configQueueSize", pool.queue.configTxsCount())
		return true
	}
	if pool.queue.commonTxsCount() >= poolconf.MaxCommonTxPoolSize() {
		pool.log.Errorw("AddTx txPool is full", "txId", tx.Header.GetTxId(), "txQueueSize", pool.queue.commonTxsCount())
		return true
	}
	return false
}

func (pool *txPoolImpl) publish(signalType txpoolPb.SignalType) {
	if pool.msgBus != nil {
		pool.log.Debugw("publish", "signalType", signalType)
		pool.msgBus.Publish(msgbus.TxPoolSignal, &txpoolPb.TxPoolSignal{
			SignalType: signalType,
			ChainId:    pool.chainId,
		})
	}
}

func (pool *txPoolImpl) broadcastTx(tx *commonPb.Transaction) {
	if pool.msgBus != nil {
		pool.log.Debugw("broadcastTx", "txId", tx.Header.GetTxId())
		txMsg, err := proto.Marshal(tx)
		if err != nil {
			pool.log.Errorw("broadcastTx proto.Marshal(tx) err", "err", err)
			return
		}
		netMsg := &netPb.NetMsg{
			Payload: txMsg,
			Type:    netPb.NetMsg_TX,
		}
		pool.msgBus.Publish(msgbus.SendTxPoolMsg, netMsg)
	}
}

// updateAndPublishSignal When the number of transactions in the transaction pool is greater
// than or equal to the block can contain, update the status of the tx pool to block
// propose, otherwise update the status of tx pool to TRANSACTION_INCOME.
func (pool *txPoolImpl) updateAndPublishSignal() {
	signalType := txpoolPb.SignalType_NO_EVENT
	defer func() {
		if signalType != txpoolPb.SignalType_NO_EVENT {
			pool.log.Debugw("updateAndPublishSignal pool.publish", "signalType", signalType)
			pool.publish(signalType)
		}
		pool.setSignalStatus(signalType)
	}()

	if pool.queue.configTxsCount() > 0 || pool.queue.commonTxsCount() >= poolconf.MaxTxCount(pool.chainConf) {
		signalType = txpoolPb.SignalType_BLOCK_PROPOSE
	} else {
		signalType = txpoolPb.SignalType_TRANSACTION_INCOME
	}
}

func (pool *txPoolImpl) GetTxByTxId(txId string) (tx *commonPb.Transaction, inBlockHeight int64) {
	return pool.queue.get(txId)
}

func (pool *txPoolImpl) GetTxsByTxIds(txIds []string) (map[string]*commonPb.Transaction, map[string]int64) {
	start := utils.CurrentTimeMillisSeconds()
	var (
		txsRet       = make(map[string]*commonPb.Transaction, len(txIds))
		txsHeightRet = make(map[string]int64, len(txIds))
	)
	for _, txId := range txIds {
		if tx, inBlockHeight := pool.queue.get(txId); tx != nil {
			txsRet[txId] = tx
			txsHeightRet[txId] = inBlockHeight
		}
	}
	pool.log.Infof("GetTxsByTxIds elapse time: %d", utils.CurrentTimeMillisSeconds()-start)
	return txsRet, txsHeightRet
}

func (pool *txPoolImpl) RetryAndRemoveTxs(retryTxs []*commonPb.Transaction, removeTxs []*commonPb.Transaction) {
	start := utils.CurrentTimeMillisSeconds()
	pool.retryTxs(retryTxs)
	pool.removeTxs(removeTxs)
	pool.log.Debugf("RetryAndRemoveTxs elapse time: %d, retry txs:%d, remove txs:%d "+
		"", utils.CurrentTimeMillisSeconds()-start, len(retryTxs), len(removeTxs))
}

// retryTxs Re-add the txs to txPool
func (pool *txPoolImpl) retryTxs(txs []*commonPb.Transaction) {
	if len(txs) == 0 {
		return
	}
	start := utils.CurrentTimeMillisSeconds()
	var (
		configTxs   = make([]*commonPb.Transaction, 0)
		commonTxs   = make([]*commonPb.Transaction, 0)
		commonTxIds = make([]string, 0, len(txs))
		configTxIds = make([]string, 0, len(txs))
	)
	for _, tx := range txs {
		if utils.IsConfigTx(tx) {
			configTxs = append(configTxs, tx)
			configTxIds = append(configTxIds, tx.Header.TxId)
		} else {
			commonTxs = append(commonTxs, tx)
			commonTxIds = append(commonTxIds, tx.Header.TxId)
		}
	}

	if len(configTxs) > 0 {
		pool.log.Debugw("retryTxBatch config txs", "count", len(configTxs), "txIds", configTxIds)
		pool.queue.addTxsToConfigQueue(&mempoolTxs{txs: configTxs, source: protocol.INTERNAL})
	}
	if len(commonTxs) > 0 {
		pool.log.Debugw("retryTxBatch common txs", "count", len(commonTxs), "txIds", commonTxIds)
		pool.queue.addTxsToCommonQueue(&mempoolTxs{txs: commonTxs, source: protocol.INTERNAL})
	}
	pool.queue.deleteTxsInPending(txs)
	pool.log.Infof("retryTxs elapse time: %d", utils.CurrentTimeMillisSeconds()-start)
}

// removeTxs delete the txs from the pool
func (pool *txPoolImpl) removeTxs(txs []*commonPb.Transaction) {
	if len(txs) == 0 {
		return
	}
	defer pool.updateAndPublishSignal()
	start := utils.CurrentTimeMillisSeconds()
	configTxIds := make([]string, 0, 1)
	commonTxIds := make([]string, 0, len(txs)/2)
	for _, tx := range txs {
		if utils.IsConfigTx(tx) {
			configTxIds = append(configTxIds, tx.Header.TxId)
		} else {
			commonTxIds = append(commonTxIds, tx.Header.TxId)
		}
	}

	if len(configTxIds) > 0 {
		pool.log.Debugw("removeTxBatch config txs", "count", len(configTxIds), "txIds", configTxIds)
		pool.queue.deleteConfigTxs(configTxIds)
	}
	if len(commonTxIds) > 0 {
		pool.log.Debugw("removeTxBatch common txs", "count", len(commonTxIds), "txIds", commonTxIds)
		pool.queue.deleteCommonTxs(commonTxIds)
	}
	pool.log.Infof("removeTxs elapse time: %d", utils.CurrentTimeMillisSeconds()-start)
}

func (pool *txPoolImpl) FetchTxBatch(blockHeight int64) []*commonPb.Transaction {
	start := utils.CurrentTimeMillisSeconds()
	txs := pool.queue.fetch(pool.maxTxCount(), blockHeight, pool.validateTxTime)
	if len(txs) > 0 {
		pool.log.Infof("fetch txs from txpool, txsNum:%d, blockHeight:%d, elapse time: %d", len(txs), blockHeight, utils.CurrentTimeMillisSeconds()-start)
	}
	return txs
}

func (pool *txPoolImpl) TxExists(tx *commonPb.Transaction) bool {
	return pool.queue.has(tx, true)
}

func (pool *txPoolImpl) metrics(msg string, startTime int64, endTime int64) {
	if poolconf.IsMetrics() {
		pool.log.Infow(msg, "internal", endTime-startTime, "startTime", startTime, "endTime", endTime)
	}
}

func (pool *txPoolImpl) AddTxsToPendingCache(txs []*commonPb.Transaction, blockHeight int64) {
	if len(txs) == 0 {
		return
	}
	pool.log.Infof("add tx to pendingCache, (txs num:%d), blockHeight:%d", len(txs), blockHeight)
	pool.queue.appendTxsToPendingCache(txs, blockHeight)
}

// OnMessage Process messages from MsgBus
func (pool *txPoolImpl) OnMessage(msg *msgbus.Message) {
	if msg == nil {
		pool.log.Errorw("receiveOnMessage msg OnMessage msg is empty")
		return
	}
	if msg.Topic != msgbus.RecvTxPoolMsg {
		pool.log.Errorw("receiveOnMessage msg topic is not msgbus.RecvTxPoolMsg")
		return
	}

	var (
		tx    = commonPb.Transaction{}
		bytes = msg.Payload.(*netPb.NetMsg).Payload
	)
	if err := proto.Unmarshal(bytes, &tx); err != nil {
		pool.log.Errorw("receiveOnMessage proto.Unmarshal(bytes, tx) err", "err", err)
		return
	}
	if err := pool.AddTx(&tx, protocol.P2P); err != nil {
		pool.log.Debugw("receiveOnMessage", "txId", tx.Header.TxId, "add failed", err.Error())
	}
	pool.log.Debugw("receiveOnMessage", "txId", tx.Header.TxId, "add success", true)
}

func (pool *txPoolImpl) OnQuit() {
	// no implement
}

func (pool *txPoolImpl) getSignalStatus() txpoolPb.SignalType {
	pool.signalLock.RLock()
	defer pool.signalLock.RUnlock()
	return pool.signalStatus
}

func (pool *txPoolImpl) setSignalStatus(signal txpoolPb.SignalType) {
	pool.signalLock.Lock()
	defer pool.signalLock.Unlock()
	pool.signalStatus = signal
}
