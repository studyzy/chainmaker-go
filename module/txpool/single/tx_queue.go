/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"fmt"
	"math"
	"sync"

	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

type txValidateFunc func(tx *commonPb.Transaction, source protocol.TxSource) error

type txQueue struct {
	log      protocol.Logger
	validate txValidateFunc

	commonTxQueue *txList   // common transaction queue
	configTxQueue *txList   // config transaction queue
	pendingCache  *sync.Map // Caches transactions that are already in the block to be deleted
}

func newQueue(blockStore protocol.BlockchainStore, log protocol.Logger, validate txValidateFunc) *txQueue {
	pendingCache := sync.Map{}
	queue := txQueue{
		log:           log,
		validate:      validate,
		pendingCache:  &pendingCache,
		commonTxQueue: newTxList(log, &pendingCache, blockStore),
		configTxQueue: newTxList(log, &pendingCache, blockStore),
	}
	return &queue
}

func (queue *txQueue) addTxsToConfigQueue(memTxs *mempoolTxs) {
	queue.configTxQueue.Put(memTxs.txs, memTxs.source, queue.validate)
}

func (queue *txQueue) addTxsToCommonQueue(memTxs *mempoolTxs) {
	queue.commonTxQueue.Put(memTxs.txs, memTxs.source, queue.validate)
}

func (queue *txQueue) deleteTxsInPending(txIds []*commonPb.Transaction) {
	for _, tx := range txIds {
		queue.pendingCache.Delete(tx.Payload.TxId)
	}
}

func (queue *txQueue) get(txId string) (tx *commonPb.Transaction, inBlockHeight uint64) {
	if tx, inBlockHeight := queue.commonTxQueue.Get(txId); tx != nil {
		return tx, inBlockHeight
	}
	if tx, inBlockHeight := queue.configTxQueue.Get(txId); tx != nil {
		return tx, inBlockHeight
	}
	return nil, math.MaxUint64
}

func (queue *txQueue) configTxsCount() int {
	return queue.configTxQueue.Size()
}

func (queue *txQueue) commonTxsCount() int {
	return queue.commonTxQueue.Size()
}

func (queue *txQueue) deleteConfigTxs(txIds []string) {
	queue.configTxQueue.Delete(txIds)
}

func (queue *txQueue) deleteCommonTxs(txIds []string) {
	queue.commonTxQueue.Delete(txIds)
}

func (queue *txQueue) fetch(expectedCount int, blockHeight uint64,
	validateTxTime func(tx *commonPb.Transaction) error) []*commonPb.Transaction {
	// 1. fetch the config transaction
	if configQueueLen := queue.configTxsCount(); configQueueLen > 0 {
		if txs, txIds := queue.configTxQueue.Fetch(1, validateTxTime, blockHeight); len(txs) > 0 {
			queue.log.Debugw("FetchTxBatch get config txs", "txCount", 1, "configQueueLen",
				configQueueLen, "txsLen", len(txs), "txIds", txIds)
			return txs
		}
	}

	// 2. fetch the common transaction
	if txQueueLen := queue.commonTxsCount(); txQueueLen > 0 {
		if txs, txIds := queue.commonTxQueue.Fetch(expectedCount, validateTxTime, blockHeight); len(txs) > 0 {
			queue.log.Debugw("FetchTxBatch get common txs", "txCount", expectedCount, "txQueueLen",
				txQueueLen, "txsLen", len(txs), "txIds", txIds)
			return txs
		}
	}
	return nil
}

func (queue *txQueue) appendTxsToPendingCache(txs []*commonPb.Transaction, blockHeight uint64, enableSqlDB bool) {
	if (utils.IsConfigTx(txs[0]) || utils.IsManageContractAsConfigTx(txs[0], enableSqlDB)) && len(txs) == 1 {
		queue.configTxQueue.appendTxsToPendingCache(txs, blockHeight)
	} else {
		queue.commonTxQueue.appendTxsToPendingCache(txs, blockHeight)
	}
}

func (queue *txQueue) has(tx *commonPb.Transaction, checkPending bool) bool {
	if queue.commonTxQueue.Has(tx.Payload.TxId, checkPending) {
		return true
	}
	return queue.configTxQueue.Has(tx.Payload.TxId, checkPending)
}

func (queue *txQueue) status() string {
	return fmt.Sprintf("common txs len: %d, config txs len: %d", queue.commonTxQueue.Size(), queue.configTxQueue.Size())
}
