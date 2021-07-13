/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker/protocol/test"

	"chainmaker.org/chainmaker-go/chainconf"
	commonErrors "chainmaker.org/chainmaker/common/errors"
	"chainmaker.org/chainmaker-go/localconf"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configpb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker-go/utils"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var log = &test.GoLogger{}

func TestNewTxPoolImpl(t *testing.T) {
	chainConf, _ := chainconf.NewChainConf(nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	txPool, err := NewTxPoolImpl("", newMockBlockChainStore(ctrl).store, newMockMessageBus(ctrl), chainConf, newMockAccessControlProvider(ctrl), newMockNet(ctrl), log)
	require.Nil(t, txPool)
	require.EqualError(t, fmt.Errorf("no chainId in create txpool"), err.Error())

	txPool, err = NewTxPoolImpl("test_chain", newMockBlockChainStore(ctrl).store, newMockMessageBus(ctrl), chainConf, newMockAccessControlProvider(ctrl), newMockNet(ctrl), log)
	require.NotNil(t, txPool)
	require.NoError(t, err)
}

type testPool struct {
	txPool protocol.TxPool
	extTxs map[string]*commonPb.Transaction
}

func newTestPool(t *testing.T, txCount uint32) (*testPool, func()) {
	chainConf, _ := chainconf.NewChainConf(nil)
	chainConf.ChainConf = &configpb.ChainConfig{
		Block:    &configpb.BlockConfig{},
		Contract: &configpb.ContractConfig{},
	}
	localconf.ChainMakerConfig.TxPoolConfig.MaxTxPoolSize = txCount
	localconf.ChainMakerConfig.TxPoolConfig.MaxConfigTxPoolSize = 1000
	localconf.ChainMakerConfig.TxPoolConfig.CacheFlushTicker = 1
	localconf.ChainMakerConfig.TxPoolConfig.CacheThresholdCount = 1
	localconf.ChainMakerConfig.LogConfig.SystemLog.LogLevels = make(map[string]string)
	localconf.ChainMakerConfig.LogConfig.SystemLog.LogLevels["txpool"] = "ERROR"
	ctrl := gomock.NewController(t)
	mockStore := newMockBlockChainStore(ctrl)
	txPool, _ := NewTxPoolImpl("test_chain", mockStore.store, newMockMessageBus(ctrl), chainConf, newMockAccessControlProvider(ctrl), newMockNet(ctrl), log)
	_ = txPool.Start()
	return &testPool{
			txPool: txPool,
			extTxs: mockStore.txs,
		}, func() {
			ctrl.Finish()
			txPool.Stop()
		}
}

func TestTxPoolImpl_AddTx(t *testing.T) {
	commonTxs := generateTxs(30, false)
	configTxs := generateTxs(30, true)
	testPool, fn := newTestPool(t, 20)
	txPool := testPool.txPool
	defer fn()

	// 2. add config txs
	for _, tx := range configTxs[:10] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}

	// 1. add common txs
	for _, tx := range commonTxs[:20] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}

	// 3. sleep to wait for the txPool to fill up
	time.Sleep(time.Second)

	// 4. check pool is full
	require.EqualError(t, commonErrors.ErrTxPoolLimit, txPool.AddTx(commonTxs[20], protocol.RPC).Error())
	require.EqualError(t, commonErrors.ErrTxPoolLimit, txPool.AddTx(configTxs[10], protocol.RPC).Error())
	imPool := txPool.(*txPoolImpl)
	require.EqualValues(t, 20, imPool.queue.commonTxsCount())
	require.EqualValues(t, 10, imPool.queue.configTxsCount())

	// 5. repeat add same txs
	localconf.ChainMakerConfig.TxPoolConfig.MaxTxPoolSize = 30
	localconf.ChainMakerConfig.TxPoolConfig.MaxConfigTxPoolSize = 11
	require.EqualError(t, commonErrors.ErrTxIdExist, txPool.AddTx(commonTxs[0], protocol.RPC).Error())
	require.EqualError(t, commonErrors.ErrTxIdExist, txPool.AddTx(configTxs[0], protocol.RPC).Error())

	// 6. add txs to blockchain
	for _, tx := range commonTxs[20:25] {
		testPool.extTxs[tx.Header.TxId] = tx
	}

	// 7. add txs[20:25] failed due to txs exist in blockchain
	for _, tx := range commonTxs[20:25] {
		// here because not check existence in blockchain, The check will
		// only be performed when the flush transaction reaches the db
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}
	//  sleep to wait for the flush
	time.Sleep(time.Second)
	require.EqualValues(t, 20, imPool.queue.commonTxsCount())

}

func TestFlushOrAddTxsToCache(t *testing.T) {
	testPool, fn := newTestPool(t, 20)
	txPool := testPool.txPool
	defer fn()
	rpcConfigTxs, _, _ := generateTxsBySource(10, true)
	rpcCommonTxs, p2pCommonTxs, _ := generateTxsBySource(10, false)
	imlPool := txPool.(*txPoolImpl)

	// 1. add config txs
	imlPool.flushOrAddTxsToCache(rpcConfigTxs)
	require.EqualValues(t, len(rpcConfigTxs.txs), imlPool.queue.configTxsCount())

	// 2. repeat add config txs
	imlPool.flushOrAddTxsToCache(rpcConfigTxs)
	require.EqualValues(t, len(rpcConfigTxs.txs), imlPool.queue.configTxsCount())

	// 3. add common txs
	imlPool.flushOrAddTxsToCache(rpcCommonTxs)
	require.EqualValues(t, len(rpcCommonTxs.txs), imlPool.queue.commonTxsCount())
	require.EqualValues(t, 0, imlPool.cache.totalCount)

	// 4. repeat add common txs due to not flush, so size will be *2
	imlPool.flushOrAddTxsToCache(rpcCommonTxs)
	require.EqualValues(t, len(rpcCommonTxs.txs), imlPool.queue.commonTxsCount())
	require.EqualValues(t, 0, imlPool.cache.totalCount)

	// 5. modify flushThreshold in cache and add common txs to queue
	imlPool.cache.flushThreshold = 0
	p2pCommonTxs.source = protocol.RPC
	imlPool.flushOrAddTxsToCache(p2pCommonTxs)
	fmt.Println(imlPool.cache.isFlushByTxCount(p2pCommonTxs), imlPool.queue.configTxsCount(), imlPool.queue.commonTxsCount())
	require.EqualValues(t, 20, imlPool.queue.commonTxsCount())
}

func TestTxPoolImpl_AddTxsToPendingCache(t *testing.T) {
	testPool, fn := newTestPool(t, 200)
	txPool := testPool.txPool
	defer fn()
	imlPool := txPool.(*txPoolImpl)
	commonTxs := generateTxs(50, false)

	// 1. add common txs
	for _, tx := range commonTxs[:20] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}
	// wait time to flush txs to queue，execute in order by adding txs to queue and adding txs to cache
	time.Sleep(time.Millisecond * 1500)
	require.EqualValues(t, 20, imlPool.queue.commonTxsCount())
	// 1.1 add txs[0:20] to pending cache
	txPool.AddTxsToPendingCache(commonTxs[:20], 99)
	require.EqualValues(t, 0, imlPool.queue.commonTxsCount())
	//require.EqualValues(t, 20, imlPool.queue.commonTxQueue.pendingCache)

	// 2. add common txs
	for _, tx := range commonTxs[20:45] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}
	// 2.1 add txs[20:40] to pending cache
	txPool.AddTxsToPendingCache(commonTxs[20:40], 100)
	// wait time to flush txs to queue with failed due to txs has exist in pending cache，parallel execution by adding txs to queue and adding txs to pending cache
	time.Sleep(time.Second * 3)
	require.EqualValues(t, 5, imlPool.queue.commonTxsCount())
	//require.EqualValues(t, 40, imlPool.queue.commonTxQueue.pendingCache.Size())

	// 3. only add txs to pending cache
	txPool.AddTxsToPendingCache(commonTxs[45:], 101)
	require.EqualValues(t, 5, imlPool.queue.commonTxsCount())
	//require.EqualValues(t, 45, imlPool.queue.commonTxQueue.pendingCache.Size())
}

func TestTxPoolImpl_GetTxByTxId(t *testing.T) {
	testPool, fn := newTestPool(t, 20)
	txPool := testPool.txPool
	defer fn()
	imlPool := txPool.(*txPoolImpl)
	commonTxs := generateTxs(50, false)

	// 1. add common txs
	for _, tx := range commonTxs[:20] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}
	time.Sleep(time.Millisecond * 1500)
	require.EqualValues(t, 20, imlPool.queue.commonTxsCount())

	// 2. check txs[:20] existence
	for _, tx := range commonTxs[:20] {
		txInPool, inBlockHeight := txPool.GetTxByTxId(tx.Header.TxId)
		require.EqualValues(t, tx, txInPool)
		require.EqualValues(t, 0, inBlockHeight)
		require.True(t, txPool.TxExists(tx))
	}

	// 3. check txs[20:50] not existence
	for _, tx := range commonTxs[20:] {
		txInPool, inBlockHeight := txPool.GetTxByTxId(tx.Header.TxId)
		require.Nil(t, txInPool)
		require.EqualValues(t, -1, inBlockHeight)
		require.False(t, txPool.TxExists(tx))
	}

	// 4. add txs[20:30] to pendingCache
	for _, tx := range commonTxs[20:30] {
		imlPool.queue.commonTxQueue.pendingCache.Store(tx.Header.TxId, &valInPendingCache{tx: tx, inBlockHeight: 99})
	}

	// 5. check txs[:20] existence
	for _, tx := range commonTxs[20:30] {
		txInPool, inBlockHeight := txPool.GetTxByTxId(tx.Header.TxId)
		require.EqualValues(t, tx, txInPool)
		require.EqualValues(t, 99, inBlockHeight)
		require.True(t, txPool.TxExists(tx))
	}
}

func TestTxPoolImpl_GetTxsByTxIds(t *testing.T) {
	testPool, fn := newTestPool(t, 20)
	txPool := testPool.txPool
	defer fn()
	imlPool := txPool.(*txPoolImpl)
	commonTxs := generateTxs(50, false)

	// 1. add common txs
	for _, tx := range commonTxs[:20] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}
	time.Sleep(time.Millisecond * 1500)
	require.EqualValues(t, 20, imlPool.queue.commonTxsCount())

	// 2. check txs[:20] existence
	txsInPool, txsHeightInPool := txPool.GetTxsByTxIds(getTxIds(commonTxs[:20]))
	for _, tx := range commonTxs[:20] {
		require.EqualValues(t, tx, txsInPool[tx.Header.TxId])
		require.EqualValues(t, 0, txsHeightInPool[tx.Header.TxId])
	}

	// 3. check txs[20:50] not existence
	txsInPool, txsHeightInPool = txPool.GetTxsByTxIds(getTxIds(commonTxs[20:]))
	for _, tx := range commonTxs[20:50] {
		require.Nil(t, txsInPool[tx.Header.TxId])
		_, exist := txsHeightInPool[tx.Header.TxId]
		require.False(t, exist)
	}

	// 4. add txs[20:30] to pendingCache
	for _, tx := range commonTxs[20:30] {
		imlPool.queue.commonTxQueue.pendingCache.Store(tx.Header.TxId, &valInPendingCache{tx: tx, inBlockHeight: 99})
	}

	// 5. check txs[:20] existence
	txsInPool, txsHeightInPool = txPool.GetTxsByTxIds(getTxIds(commonTxs[20:30]))
	for _, tx := range commonTxs[20:30] {
		require.EqualValues(t, tx, txsInPool[tx.Header.TxId])
		require.EqualValues(t, 99, txsHeightInPool[tx.Header.TxId])
	}
}

func TestTxPoolImpl_FetchTxBatch(t *testing.T) {
	testPool, fn := newTestPool(t, 2000)
	txPool := testPool.txPool
	defer fn()
	imlPool := txPool.(*txPoolImpl)
	commonTxs := generateTxs(100, false)

	// 1. add common txs
	for _, tx := range commonTxs[:50] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}

	// 2. sleep to wait txs flush
	time.Sleep(time.Millisecond * 100)
	txsInPool := txPool.FetchTxBatch(99)
	require.EqualValues(t, commonTxs[:50], txsInPool)
	require.EqualValues(t, 0, imlPool.queue.commonTxsCount())
}

func TestTxPoolImpl_RetryAndRemoveTxs(t *testing.T) {
	testPool, fn := newTestPool(t, 2000)
	txPool := testPool.txPool
	defer fn()
	imlPool := txPool.(*txPoolImpl)
	commonTxs := generateTxs(100, false)

	// 1. add common txs
	for _, tx := range commonTxs[:50] {
		require.NoError(t, txPool.AddTx(tx, protocol.RPC))
	}
	time.Sleep(time.Millisecond * 100)
	require.EqualValues(t, 50, imlPool.queue.commonTxsCount())

	// 2. retry nil and remove txs[50:60]
	txPool.RetryAndRemoveTxs(nil, commonTxs[50:60])
	require.EqualValues(t, 50, imlPool.queue.commonTxsCount())

	// 3. retry nil and remove txs[0:50]
	txPool.RetryAndRemoveTxs(nil, commonTxs[:50])
	require.EqualValues(t, 0, imlPool.queue.commonTxsCount())

	// 4. retry txs[0:50] and remove txs[0:50]
	txPool.RetryAndRemoveTxs(commonTxs[:50], commonTxs[:50])
	require.EqualValues(t, 0, imlPool.queue.commonTxsCount())

	// 5. retry txs[0:80] and remove txs[0:50]
	txPool.RetryAndRemoveTxs(commonTxs[:80], commonTxs[:50])
	require.EqualValues(t, 30, imlPool.queue.commonTxsCount())
	txsInPool, _ := txPool.GetTxsByTxIds(getTxIds(commonTxs[50:80]))
	require.EqualValues(t, 30, len(txsInPool))

	// 6. Add txs[:50] to pendingCache, and retry txs[:50] and delRetry = true
	for _, tx := range commonTxs[:50] {
		imlPool.queue.pendingCache.Store(tx.Header.TxId, &valInPendingCache{tx: tx, inBlockHeight: 999})
	}
	//require.EqualValues(t, 50, imlPool.queue.pendingCache.Size())
	txPool.RetryAndRemoveTxs(commonTxs[:50], nil)
	require.EqualValues(t, 80, imlPool.queue.commonTxsCount())
	//require.EqualValues(t, 0, imlPool.queue.pendingCache.Size())
}

func TestPoolImplConcurrencyInvoke(t *testing.T) {
	testPool, fn := newTestPool(t, 2000000)
	txPool := testPool.txPool
	defer fn()
	imlPool := txPool.(*txPoolImpl)
	commonTxs := generateTxs(500000, false)
	txIds := make([]string, 0, len(commonTxs))
	for _, tx := range commonTxs {
		txIds = append(txIds, tx.Header.TxId)
	}

	// 1. Concurrent Adding Transactions to txPool
	addBegin := utils.CurrentTimeMillisSeconds()
	workerNum := 100
	peerWorkerTxNum := len(commonTxs) / workerNum
	wg := sync.WaitGroup{}
	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go func(i int, txs []*commonPb.Transaction) {
			for _, tx := range txs {
				txPool.AddTx(tx, protocol.RPC)
			}
			wg.Done()
			imlPool.log.Debugf("add txs done")
		}(i, commonTxs[i*peerWorkerTxNum:(i+1)*peerWorkerTxNum])
	}

	// 2. Simulate the logic for generating blocks
	fetchTimes := make([]int64, 0, 100)
	go func() {
		height := int64(100)
		fetchTicker := time.NewTicker(time.Millisecond * 100)
		fetchTimer := time.NewTimer(2 * time.Minute)
		defer func() {
			fetchTimer.Stop()
			fetchTicker.Stop()
		}()

	Loop:
		for {
			select {
			case <-fetchTicker.C:
				begin := utils.CurrentTimeMillisSeconds()
				txs := txPool.FetchTxBatch(height)
				fetchTimes = append(fetchTimes, utils.CurrentTimeMillisSeconds()-begin)
				imlPool.log.Debugf("fetch txs num: ", len(txs))
			case <-fetchTimer.C:
				break Loop
			}
		}
		imlPool.log.Debugf("time used: fetch txs: %v ", fetchTimes)
	}()

	// 3. Simulation validates the logic of the block
	getTimes := make([]int64, 0, 100)
	go func() {
		getTicker := time.NewTicker(time.Millisecond * 80)
		getTimer := time.NewTimer(2 * time.Minute)
		defer func() {
			getTimer.Stop()
			getTicker.Stop()
		}()

	Loop:
		for {
			select {
			case <-getTicker.C:
				start := rand.Intn(len(txIds) - 1000)
				begin := utils.CurrentTimeMillisSeconds()
				getTxs, _ := txPool.GetTxsByTxIds(txIds[start : start+1000])
				getTimes = append(getTimes, utils.CurrentTimeMillisSeconds()-begin)
				imlPool.log.Debugf("get txs num: ", len(getTxs))
			case <-getTimer.C:
				break Loop
			}
		}
		imlPool.log.Debugf("time used: get txs: %v ", getTimes)
	}()

	wg.Wait()
	addEnd := utils.CurrentTimeMillisSeconds()
	imlPool.log.Debugf("time used: add txs: %d, txPool state: %s\n", addEnd-addBegin, imlPool.queue.status())
}
