/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"fmt"
	"math"
	"sync"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/common/v2/linkedhashmap"
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"github.com/prometheus/client_golang/prometheus"
)

// txList Structure of store transactions in memory
type txList struct {
	log              protocol.Logger
	blockchainStore  protocol.BlockchainStore
	metricTxPoolSize *prometheus.GaugeVec

	rwLock       sync.RWMutex
	queue        *linkedhashmap.LinkedHashMap // Orderly store TXS: txs
	pendingCache *sync.Map                    // A place where transactions are stored after Fetch
}

func newTxList(log protocol.Logger, pendingCache *sync.Map, blockchainStore protocol.BlockchainStore) *txList {
	list := &txList{
		log:             log,
		blockchainStore: blockchainStore,
		rwLock:          sync.RWMutex{},
		queue:           linkedhashmap.NewLinkedHashMap(),
		pendingCache:    pendingCache,
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		list.metricTxPoolSize = monitor.NewGaugeVec(monitor.SUBSYSTEM_TXPOOL, "metric_tx_pool_size",
			"tx pool size", "chainId", "poolType")
	}
	return list
}

// Put Add transaction to the txList
func (l *txList) Put(txs []*commonPb.Transaction, source protocol.TxSource, validate txValidateFunc) {
	if len(txs) == 0 {
		return
	}

	for _, tx := range txs {
		l.addTxs(tx, source, validate)
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		if utils.IsConfigTx(txs[0]) {
			go l.metricTxPoolSize.WithLabelValues(txs[0].Payload.ChainId, "config").Set(float64(l.queue.Size()))
		} else {
			go l.metricTxPoolSize.WithLabelValues(txs[0].Payload.ChainId, "normal").Set(float64(l.queue.Size()))
		}
	}
}

func (l *txList) addTxs(tx *commonPb.Transaction, source protocol.TxSource, validate txValidateFunc) {
	l.rwLock.Lock()
	defer l.rwLock.Unlock()
	if validate == nil || validate(tx, source) == nil {
		if source != protocol.INTERNAL {
			if val, ok := l.pendingCache.Load(tx.Payload.TxId); ok && val != nil {
				return
			}
		}
		if l.queue.Get(tx.Payload.TxId) != nil {
			return
		}
		l.queue.Add(tx.Payload.TxId, tx)
	}
}

// Delete Delete transactions from TXList by the txIds
func (l *txList) Delete(txIds []string) {
	l.rwLock.Lock()
	defer l.rwLock.Unlock()
	l.log.Debugf("remove txIds num: %d", len(txIds))
	for _, txId := range txIds {
		l.queue.Remove(txId)
		l.pendingCache.Delete(txId)
	}

}

// Fetch Gets a list of stored transactions
func (l *txList) Fetch(count int, validate func(tx *commonPb.Transaction) error,
	blockHeight uint64) ([]*commonPb.Transaction, []string) {
	queueLen := l.queue.Size()
	if queueLen < count {
		count = queueLen
	}

	var (
		txs      []*commonPb.Transaction
		txIds    []string
		errKeys  []string
		cacheKVs []*valInPendingCache
	)
	l.rwLock.Lock()
	defer func() {
		if len(txs) > 0 {
			l.monitor(txs[0], l.queue.Size())
		}
		begin := utils.CurrentTimeMillisSeconds()
		for _, txId := range errKeys {
			l.queue.Remove(txId)
		}
		for _, val := range cacheKVs {
			l.queue.Remove(val.tx.Payload.TxId)
			l.pendingCache.Store(val.tx.Payload.TxId, val)
		}
		l.rwLock.Unlock()
		l.log.Debugf("eliminate data, elapse time: %d", utils.CurrentTimeMillisSeconds()-begin)
	}()

	l.log.Debugw("txList Fetch", "count", count, "queueLen", queueLen)
	if queueLen > 0 {
		cacheKVs, txs, txIds, errKeys = l.getTxsFromQueue(count, blockHeight, validate)
		l.log.Debugw("txList Fetch txsNormal", "count", count, "queueLen", queueLen,
			"txsLen", len(txs), "errKeys", len(errKeys), "cacheKeys", len(cacheKVs))
	}
	return txs, txIds
}

func (l *txList) getTxsFromQueue(count int, blockHeight uint64, validate func(tx *commonPb.Transaction) error) (
	cacheKVs []*valInPendingCache, txs []*commonPb.Transaction, txIds []string, errKeys []string) {

	txs = make([]*commonPb.Transaction, 0, count)
	txIds = make([]string, 0, count)
	errKeys = make([]string, 0, count)
	cacheKVs = make([]*valInPendingCache, 0, count)
	node := l.queue.GetLinkList().Front()
	for node != nil && count > 0 {
		txId, ok := node.Value.(string)
		if !ok {
			l.log.Errorf("interface value transfer into string failed")
		}
		tx, ok1 := l.queue.Get(txId).(*commonPb.Transaction)
		if !ok1 {
			l.log.Errorf("interface val transfer into *commonPb.Transaction failed")
		}
		if validate != nil && validate(tx) != nil {
			errKeys = append(errKeys, txId)
		} else {
			txs = append(txs, tx)
			txIds = append(txIds, txId)
			cacheKVs = append(cacheKVs, &valInPendingCache{
				tx:            tx,
				inBlockHeight: blockHeight,
			})
			if val, ok := l.pendingCache.Load(txId); ok {
				l.log.Errorf("tx:%s duplicate to package block, txInPoolHeight: %d", txId, val.(*valInPendingCache).inBlockHeight)
			}
		}
		count--
		node = node.Next()
	}
	return
}

func (l *txList) monitor(tx *commonPb.Transaction, len int) {
	chainId := tx.Payload.ChainId
	isConfigTx := utils.IsConfigTx(tx)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled && chainId != "" {
		if isConfigTx {
			go l.metricTxPoolSize.WithLabelValues(chainId, "config").Set(float64(len))
		} else {
			go l.metricTxPoolSize.WithLabelValues(chainId, "normal").Set(float64(len))
		}
	}
}

// Has Determine if the transaction exists in the txList
func (l *txList) Has(txId string, checkPending bool) (exist bool) {
	if checkPending {
		if val, ok := l.pendingCache.Load(txId); ok && val != nil {
			return true
		}
	}
	l.rwLock.RLock()
	defer l.rwLock.RUnlock()
	return l.queue.Get(txId) != nil
}

// Get Retrieve the transaction from txList by the txId
// inBlockHeight: return -1 when the transaction does not exist,
// return 0 when the transaction is in the queue to wait to be generate block,
// return positive integer, indicating that the tx is in an unchained block.
func (l *txList) Get(txId string) (tx *commonPb.Transaction, inBlockHeight uint64) {
	if pendingVal, ok := l.pendingCache.Load(txId); ok && pendingVal != nil {
		l.log.Debugw(fmt.Sprintf("txList Get Transaction by txId = %s in pendingCache", txId), "exist", true)
		val, ok := pendingVal.(*valInPendingCache)
		if !ok {
			l.log.Errorf("interface pendingVal transfer into *valInPendingCache failed")
		}
		return val.tx, val.inBlockHeight
	}

	l.rwLock.RLock()
	defer l.rwLock.RUnlock()
	if val := l.queue.Get(txId); val != nil {
		l.log.Debugw(fmt.Sprintf("txList Get Transaction by txId = %s in queue", txId), "exist", true)
		return val.(*commonPb.Transaction), 0
	}
	l.log.Debugw(fmt.Sprintf("txList Get Transaction by txId = %s", txId), "exist", false)
	return nil, math.MaxUint64
}

func (l *txList) appendTxsToPendingCache(txs []*commonPb.Transaction, blockHeight uint64) {
	l.rwLock.Lock()
	defer l.rwLock.Unlock()
	for _, tx := range txs {
		l.pendingCache.Store(tx.Payload.TxId, &valInPendingCache{tx: tx, inBlockHeight: blockHeight})
		l.queue.Remove(tx.Payload.TxId)
	}
}

// Size Gets the number of transactions stored in the txList
func (l *txList) Size() int {
	l.rwLock.RLock()
	defer l.rwLock.RUnlock()
	return l.queue.Size()
}
