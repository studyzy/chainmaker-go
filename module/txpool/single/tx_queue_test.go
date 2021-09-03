/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func mockValidateInQueue(queue *txQueue, blockChainStore protocol.BlockchainStore) txValidateFunc {
	return func(tx *commonPb.Transaction, source protocol.TxSource) error {
		if _, ok := queue.pendingCache.Load(tx.Payload.TxId); ok {
			return fmt.Errorf("tx exist in txpool")
		}
		if queue.commonTxQueue.queue.Get(tx.Payload.TxId) != nil {
			return fmt.Errorf("tx exist in txpool")
		}
		if queue.configTxQueue.queue.Get(tx.Payload.TxId) != nil {
			return fmt.Errorf("tx exist in txpool")
		}
		if blockChainStore != nil {
			if exist, _ := blockChainStore.TxExists(tx.Payload.TxId); exist {
				return fmt.Errorf("tx exist in blockchain")
			}
		}
		return nil
	}
}

var testQueueLogName = "test_tx_queue"

func TestAddTxsToConfigQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, internalTxs := generateTxsBySource(10, true)

	// 1. put txs to config queue
	queue.addTxsToConfigQueue(rpcTxs)
	queue.addTxsToConfigQueue(p2pTxs)
	queue.addTxsToConfigQueue(internalTxs)
	require.EqualValues(t, 30, queue.configTxsCount())

	// 2. repeat put txs to config queue failed when source = [RPC,P2P]
	queue.addTxsToConfigQueue(rpcTxs)
	queue.addTxsToConfigQueue(p2pTxs)
	queue.addTxsToConfigQueue(internalTxs)
	require.EqualValues(t, 30, queue.configTxsCount())
	require.EqualValues(t, 0, queue.commonTxsCount())

	// 3. repeat put txs to common queue failed due to txIds exist in config queue
	for _, tx := range rpcTxs.txs {
		tx.Payload.TxType = commonPb.TxType_INVOKE_CONTRACT
	}
	queue.addTxsToCommonQueue(rpcTxs)
	queue.addTxsToCommonQueue(p2pTxs)
	require.EqualValues(t, 30, queue.configTxsCount())
	require.EqualValues(t, 0, queue.commonTxsCount())
}
func changeTx2ConfigTx(tx *commonPb.Transaction) {
	payload := tx.Payload
	payload.ContractName = syscontract.SystemContract_CHAIN_CONFIG.String()
}
func TestAddTxsToCommonQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, internalTxs := generateTxsBySource(10, false)

	// 1. put txs to queue
	queue.addTxsToCommonQueue(rpcTxs)
	queue.addTxsToCommonQueue(p2pTxs)
	queue.addTxsToCommonQueue(internalTxs)
	require.EqualValues(t, 30, queue.commonTxsCount())

	// 2. repeat put txs to queue failed when source = [RPC,P2P]
	queue.addTxsToCommonQueue(rpcTxs)
	queue.addTxsToCommonQueue(p2pTxs)
	queue.addTxsToCommonQueue(internalTxs)
	require.EqualValues(t, 30, queue.commonTxsCount())
	require.EqualValues(t, 0, queue.configTxsCount())

	// 3. repeat put txs to config queue failed due to txIds exist in common queue
	for _, tx := range rpcTxs.txs {
		//tx.Payload.TxType = commonPb.TxType_INVOKE_CONTRACT
		changeTx2ConfigTx(tx)
	}
	queue.addTxsToConfigQueue(rpcTxs)
	queue.addTxsToConfigQueue(p2pTxs)
	require.EqualValues(t, 0, queue.configTxsCount())
	require.EqualValues(t, 30, queue.commonTxsCount())
}

func TestGetInQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, internalTxs := generateTxsBySource(10, false)

	// 1. put txs to queue and check existence
	queue.addTxsToCommonQueue(rpcTxs)
	for _, tx := range rpcTxs.txs {
		txInPool, inBlockHeight := queue.get(tx.Payload.TxId)
		require.EqualValues(t, txInPool, tx)
		require.EqualValues(t, 0, inBlockHeight)
	}

	// 2. check not existence
	for _, tx := range internalTxs.txs {
		txInPool, inBlockHeight := queue.get(tx.Payload.TxId)
		require.Nil(t, txInPool)
		require.EqualValues(t, -1, inBlockHeight)
	}
	for _, tx := range p2pTxs.txs {
		txInPool, inBlockHeight := queue.get(tx.Payload.TxId)
		require.Nil(t, txInPool)
		require.EqualValues(t, -1, inBlockHeight)
	}

	// 3. modify p2pTxs txType to commonPb.TxType_INVOKE_CONTRACT
	for _, tx := range p2pTxs.txs {
		//tx.Payload.TxType = commonPb.TxType_INVOKE_CONTRACT
		changeTx2ConfigTx(tx)
	}

	// 4. put txs to config queue and check existence
	queue.addTxsToConfigQueue(p2pTxs)
	for _, tx := range p2pTxs.txs {
		txInPool, inBlockHeight := queue.get(tx.Payload.TxId)
		require.EqualValues(t, txInPool, tx)
		require.EqualValues(t, 0, inBlockHeight)
	}
}

func TestHasInQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, internalTxs := generateTxsBySource(10, false)

	// 1. put txs to queue and check existence
	queue.addTxsToCommonQueue(rpcTxs)
	for _, tx := range rpcTxs.txs {
		require.True(t, queue.has(tx, true))
	}

	// 2. check not existence
	for _, tx := range internalTxs.txs {
		require.False(t, queue.has(tx, true))
	}
	for _, tx := range p2pTxs.txs {
		require.False(t, queue.has(tx, true))
	}

	// 3. modify p2pTxs txType to commonPb.TxType_INVOKE_CONTRACT
	for _, tx := range p2pTxs.txs {
		//tx.Payload.TxType = commonPb.TxType_INVOKE_CONTRACT
		changeTx2ConfigTx(tx)
	}

	// 4. put txs to config queue and check existence
	queue.addTxsToConfigQueue(p2pTxs)
	for _, tx := range p2pTxs.txs {
		require.True(t, queue.has(tx, true))
	}
}

func TestDeleteConfigTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, _ := generateTxsBySource(10, true)

	// 1. put txs to queue
	queue.addTxsToConfigQueue(rpcTxs)

	// 2. delete txs in common queue and check existence
	queue.deleteCommonTxs(getTxIds(rpcTxs.txs))
	for _, tx := range rpcTxs.txs {
		require.True(t, queue.has(tx, true))
	}
	require.EqualValues(t, 10, queue.configTxsCount())

	// 3. delete txs in config queue and check existence
	queue.deleteConfigTxs(getTxIds(rpcTxs.txs))
	for _, tx := range rpcTxs.txs {
		require.False(t, queue.has(tx, true))
	}
	require.EqualValues(t, 0, queue.configTxsCount())

	// 4. delete not exist txs and check existence
	queue.deleteConfigTxs(getTxIds(p2pTxs.txs))
	require.EqualValues(t, 0, queue.configTxsCount())
}

func TestDeleteCommonTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, _ := generateTxsBySource(10, false)

	// 1. put txs to queue and check existence
	queue.addTxsToCommonQueue(rpcTxs)

	// 2. delete txs in common queue and check existence
	queue.deleteConfigTxs(getTxIds(rpcTxs.txs))
	for _, tx := range rpcTxs.txs {
		require.True(t, queue.has(tx, true))
	}
	require.EqualValues(t, 10, queue.commonTxsCount())

	// 3. delete txs in config queue and check existence
	queue.deleteCommonTxs(getTxIds(rpcTxs.txs))
	for _, tx := range rpcTxs.txs {
		require.False(t, queue.has(tx, true))
	}
	require.EqualValues(t, 0, queue.commonTxsCount())

	// 4. delete not exist txs and check existence
	queue.deleteConfigTxs(getTxIds(p2pTxs.txs))
	require.EqualValues(t, 0, queue.commonTxsCount())
}

func TestAppendTxsToPendingCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, _ := generateTxsBySource(10, false)

	// 1. put txs to queue and check appendTxsToPendingCache
	queue.addTxsToCommonQueue(rpcTxs)
	queue.appendTxsToPendingCache(rpcTxs.txs, 100, false)
	//require.EqualValues(t, 10, queue.commonTxQueue.pendingCache.Size())

	// 3. repeat appendTxsToPendingCache txs
	queue.appendTxsToPendingCache(rpcTxs.txs, 100, false)
	//require.EqualValues(t, 10, queue.commonTxQueue.pendingCache.Size())

	// 4. modify p2pTxs txType to commonPb.TxType_INVOKE_CONTRACT
	for _, tx := range p2pTxs.txs {
		//tx.Payload.TxType = commonPb.TxType_INVOKE_CONTRACT
		changeTx2ConfigTx(tx)
	}

	// 5. add txs to config queue and check appendTxsToPendingCache
	queue.addTxsToCommonQueue(rpcTxs)
	queue.appendTxsToPendingCache(p2pTxs.txs[:1], 101, false)
	//require.EqualValues(t, 11, queue.configTxQueue.pendingCache.Size())

	// 6. append >1 config txs
	queue.appendTxsToPendingCache(p2pTxs.txs[1:], 101, false)
	//require.EqualValues(t, 11, queue.configTxQueue.pendingCache.Size())
}

func TestFetchInQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	queue := newQueue(blockChainStore.store, logger.GetLogger(testQueueLogName), nil)
	queue.validate = mockValidateInQueue(queue, blockChainStore.store)
	rpcTxs, p2pTxs, _ := generateTxsBySource(10, false)

	// 1. put txs to queue and check appendTxsToPendingCache
	queue.addTxsToCommonQueue(rpcTxs)
	fetchTxs := queue.fetch(100, 99, nil)
	require.EqualValues(t, rpcTxs.txs, fetchTxs)
	//require.EqualValues(t, len(rpcTxs.txs), queue.configTxQueue.pendingCache.Size())

	// 2. fetch txs nil
	fetchTxs = queue.fetch(100, 99, nil)
	require.EqualValues(t, 0, len(fetchTxs))

	// 3. modify p2pTxs txType to commonPb.TxType_INVOKE_CONTRACT and push txs to config queue
	for _, tx := range p2pTxs.txs {
		//tx.Payload.TxType = commonPb.TxType_INVOKE_CONTRACT
		changeTx2ConfigTx(tx)
	}
	queue.addTxsToConfigQueue(p2pTxs)

	// 4. fetch config tx
	fetchTxs = queue.fetch(100, 100, nil)
	require.EqualValues(t, p2pTxs.txs[:1], fetchTxs)
	//require.EqualValues(t, 11, queue.configTxQueue.pendingCache.Size())

	// 5. next fetch
	fetchTxs = queue.fetch(100, 101, nil)
	require.EqualValues(t, p2pTxs.txs[1:2], fetchTxs)
	//require.EqualValues(t, 12, queue.configTxQueue.pendingCache.Size())
}
