/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/protocol/v2"
)

type txCache struct {
	txs        []*mempoolTxs // The transactions in the cache
	totalCount int           // The number of transactions in the cache

	flushThreshold int           // The threshold of the number of transactions in the cache. When the number of transactions in the cache is greater than or equal to the threshold, the transactions will be flushed to the queue.
	flushTimeOut   time.Duration // The timeOut to flush cache
	lastFlushTime  time.Time     // Time of the latest cache refresh
}

func newTxCache() *txCache {
	cache := &txCache{flushThreshold: DefaultCacheThreshold, flushTimeOut: DefaultFlushTimeOut}
	if localconf.ChainMakerConfig.TxPoolConfig.CacheThresholdCount > 0 {
		cache.flushThreshold = int(localconf.ChainMakerConfig.TxPoolConfig.CacheThresholdCount)
	}
	if localconf.ChainMakerConfig.TxPoolConfig.CacheFlushTimeOut > 0 {
		cache.flushTimeOut = time.Duration(localconf.ChainMakerConfig.TxPoolConfig.CacheFlushTimeOut) * time.Second
	}
	return cache
}

// isFlushByTxCount Whether the number of transactions in the cache reaches the refresh threshold
func (cache *txCache) isFlushByTxCount(memTxs *mempoolTxs) bool {
	if memTxs != nil {
		return len(memTxs.txs)+cache.totalCount >= cache.flushThreshold
	}
	return cache.totalCount >= cache.flushThreshold
}

// addMemoryTxs Add transactions to the cache without any checks at this time.
// When the transactions in the cache is refreshed to the queue, the validity of the transaction will be checked.
func (cache *txCache) addMemoryTxs(memTxs *mempoolTxs) {
	cache.txs = append(cache.txs, memTxs)
	cache.totalCount += len(memTxs.txs)
}

// mergeAndSplitTxsBySource Divide the transactions in the cache according to the source.
func (cache *txCache) mergeAndSplitTxsBySource(memTxs *mempoolTxs) (rpcTxs, p2pTxs, internalTxs []*commonPb.Transaction) {
	totalCount := cache.totalCount
	if memTxs != nil {
		totalCount = len(memTxs.txs) + cache.totalCount
	}
	rpcTxs = make([]*commonPb.Transaction, 0, totalCount/2)
	p2pTxs = make([]*commonPb.Transaction, 0, totalCount/2)
	internalTxs = make([]*commonPb.Transaction, 0, totalCount/2)
	for _, v := range cache.txs {
		splitTxsBySource(v, &rpcTxs, &p2pTxs, &internalTxs)
	}
	if memTxs != nil {
		splitTxsBySource(memTxs, &rpcTxs, &p2pTxs, &internalTxs)
	}
	return
}

func splitTxsBySource(memTxs *mempoolTxs, rpcTxs, p2pTxs, internalTxs *[]*commonPb.Transaction) {
	if len(memTxs.txs) == 0 {
		return
	}
	if memTxs.source == protocol.RPC {
		*rpcTxs = append(*rpcTxs, memTxs.txs...)
	} else if memTxs.source == protocol.P2P {
		*p2pTxs = append(*p2pTxs, memTxs.txs...)
	} else {
		*internalTxs = append(*internalTxs, memTxs.txs...)
	}
}

// reset Reset the cache.
func (cache *txCache) reset() {
	cache.txs = cache.txs[:0]
	cache.totalCount = 0
	cache.lastFlushTime = time.Now()
}

// isFlushByTime Whether the cache refresh time threshold is reached
func (cache *txCache) isFlushByTime() bool {
	return time.Now().After(cache.lastFlushTime.Add(cache.flushTimeOut))
}

// txCount The number of transactions in the cache.
func (cache *txCache) txCount() int {
	return cache.totalCount
}
