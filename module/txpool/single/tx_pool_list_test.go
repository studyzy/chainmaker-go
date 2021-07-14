/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"fmt"
	"sync"
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/common"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/protocol"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func mockValidate(txList *txList, blockChainStore protocol.BlockchainStore) txValidateFunc {
	return func(tx *commonPb.Transaction, source protocol.TxSource) error {
		if source != protocol.INTERNAL {
			if val, ok := txList.pendingCache.Load(tx.Payload.TxId); ok && val != nil {
				return fmt.Errorf("tx exist in txpool")
			}
		}
		if txList.queue.Get(tx.Payload.TxId) != nil {
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

func generateTxs(num int, isConfig bool) []*commonPb.Transaction {
	txs := make([]*commonPb.Transaction, 0, num)
	txType := commonPb.TxType_INVOKE_CONTRACT
	for i := 0; i < num; i++ {
		contractName := syscontract.SystemContract_CHAIN_CONFIG.String()

		if !isConfig {
			contractName = "userContract1"
		}
		txs = append(txs, &commonPb.Transaction{
			Payload: &commonPb.Payload{TxId: utils.GetRandTxId(), TxType: txType,
				Method: "SetConfig", ContractName: contractName},
		},
		)
	}
	return txs
}

func getTxIds(txs []*commonPb.Transaction) []string {
	txIds := make([]string, 0, len(txs))
	for _, tx := range txs {
		txIds = append(txIds, tx.Payload.TxId)
	}
	return txIds
}

var testListLogName = "test_tx_list"

func TestTxList_Put(t *testing.T) {
	// 0. init source
	txs := generateTxs(100, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := newMockBlockChainStore(ctrl)
	list := newTxList(logger.GetLogger(testListLogName), &sync.Map{}, mockStore.store)
	validateFunc := mockValidate(list, mockStore.store)

	// 1. put 30 rpc txs and check num in txList
	list.Put(txs[:10], protocol.RPC, validateFunc)
	require.EqualValues(t, 10, list.Size())
	list.Put(txs[10:20], protocol.P2P, validateFunc)
	require.EqualValues(t, 20, list.Size())
	list.Put(txs[20:30], protocol.INTERNAL, validateFunc)
	require.EqualValues(t, 30, list.Size())

	// 2. check the tx exist in txList
	for _, tx := range txs[:30] {
		require.EqualError(t, fmt.Errorf("tx exist in txpool"), validateFunc(tx, protocol.RPC).Error())
	}
	//  check the tx not exist in blockChainStore
	for _, tx := range txs[30:] {
		require.NoError(t, validateFunc(tx, protocol.RPC))
	}

	// 3. add txs in mockBlockChainStore
	for _, tx := range txs[50:80] {
		mockStore.txs[tx.Payload.TxId] = tx
	}

	// 4. put txs[50:80] failed due to the txs has exist in blockchain when source = [RPC,P2P]
	list.Put(txs[50:60], protocol.RPC, validateFunc)
	require.EqualValues(t, 30, list.Size())
	list.Put(txs[60:70], protocol.P2P, validateFunc)
	require.EqualValues(t, 30, list.Size())
	list.Put(txs[70:80], protocol.INTERNAL, validateFunc)
	require.EqualValues(t, 30, list.Size())
	for _, tx := range txs[50:80] {
		require.EqualError(t, fmt.Errorf("tx exist in blockchain"), validateFunc(tx, protocol.RPC).Error())
	}

	// 5. put txs[80:90] succeed due to not check the tx existence in pendingCache when source = [INTERNAL]
	for _, tx := range txs[80:90] {
		list.pendingCache.Store(tx.Payload.TxId, &valInPendingCache{0, tx})
	}
	list.Put(txs[80:90], protocol.RPC, validateFunc)
	list.Put(txs[80:90], protocol.P2P, validateFunc)
	require.EqualValues(t, 30, list.Size())
	list.Put(txs[80:90], protocol.INTERNAL, validateFunc)
	require.EqualValues(t, 40, list.Size())

	// 6. repeat put txs[90:100]
	list.Put(txs[90:100], protocol.RPC, validateFunc)
	require.EqualValues(t, 50, list.Size())
	list.Put(txs[90:100], protocol.P2P, validateFunc)
	require.EqualValues(t, 50, list.Size())
	list.Put(txs[90:100], protocol.RPC, validateFunc)
	list.Put(txs[90:100], protocol.INTERNAL, validateFunc)
	require.EqualValues(t, 50, list.Size())
}

func TestTxList_Get(t *testing.T) {
	txs := generateTxs(100, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	list := newTxList(logger.GetLogger(testListLogName), &sync.Map{}, blockChainStore.store)
	validateFunc := mockValidate(list, blockChainStore.store)

	// 1. put txs[:30] txs and check existence
	list.Put(txs[:30], protocol.RPC, validateFunc)
	for _, tx := range txs[:30] {
		tx, inBlockHeight := list.Get(tx.Payload.TxId)
		require.NotNil(t, tx)
		require.EqualValues(t, 0, inBlockHeight)
	}

	// 2. check txs[30:100] not exist in txList
	for _, tx := range txs[30:100] {
		tx, inBlockHeight := list.Get(tx.Payload.TxId)
		require.Nil(t, tx)
		require.EqualValues(t, -1, inBlockHeight)
	}

	// 3. put txs[30:40] to pending cache and check txs[30:40] exist in pendingCache in the txList
	for _, tx := range txs[30:40] {
		list.pendingCache.Store(tx.Payload.TxId, &valInPendingCache{inBlockHeight: 999, tx: tx})
	}
	for _, tx := range txs[30:40] {
		txInPool, inBlockHeight := list.Get(tx.Payload.TxId)
		require.EqualValues(t, tx, txInPool)
		require.EqualValues(t, 999, inBlockHeight)
	}
}

func TestTxList_Has(t *testing.T) {
	txs := generateTxs(100, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	list := newTxList(logger.GetLogger(testListLogName), &sync.Map{}, blockChainStore.store)
	validateFunc := mockValidate(list, blockChainStore.store)

	// 1. put txs[:30] txs and check existence
	list.Put(txs[:30], protocol.RPC, validateFunc)
	for _, tx := range txs[:30] {
		require.True(t, list.Has(tx.Payload.TxId, true))
		require.True(t, list.Has(tx.Payload.TxId, false))
	}

	// 2. put txs[30:40] in pendingCache in txList and check existence
	for _, tx := range txs[30:40] {
		list.pendingCache.Store(tx.Payload.TxId, &valInPendingCache{0, tx})
	}
	for _, tx := range txs[30:40] {
		require.True(t, list.Has(tx.Payload.TxId, true))
		require.False(t, list.Has(tx.Payload.TxId, false))
	}

	// 3. check not existence in txList
	for _, tx := range txs[40:] {
		require.False(t, list.Has(tx.Payload.TxId, true))
		require.False(t, list.Has(tx.Payload.TxId, false))
	}
}

func TestTxList_Delete(t *testing.T) {
	txs := generateTxs(100, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	list := newTxList(logger.GetLogger(testListLogName), &sync.Map{}, blockChainStore.store)
	validateFunc := mockValidate(list, blockChainStore.store)

	// 1. put txs[:30]
	list.Put(txs[:30], protocol.RPC, validateFunc)

	// 2. delete txs[:10] and check correctness
	list.Delete(getTxIds(txs[:10]))
	require.EqualValues(t, 20, list.Size())
	for _, tx := range txs[:10] {
		require.False(t, list.Has(tx.Payload.TxId, true))
		require.False(t, list.Has(tx.Payload.TxId, false))
	}

	// 2. put txs[30:50] in the pendingCache in txList
	for _, tx := range txs[30:50] {
		list.pendingCache.Store(tx.Payload.TxId, &valInPendingCache{0, tx})
	}
	//require.EqualValues(t, 20, len(list.pendingCache))
	list.Delete(getTxIds(txs[30:40]))
	//require.EqualValues(t, 10, len(list.pendingCache))

	// 3. put txs[40:50] succeed due to not check existence when source = [INTERNAL]
	list.Put(txs[40:50], protocol.INTERNAL, validateFunc)
	require.EqualValues(t, 30, list.Size())

	// 4. delete txs[40:50], check pendingCache size and queue size
	list.Delete(getTxIds(txs[40:50]))
	require.EqualValues(t, 20, list.Size())
	//require.EqualValues(t, 0, len(list.pendingCache))
}

func TestTxList_Fetch(t *testing.T) {
	txs := generateTxs(100, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	list := newTxList(logger.GetLogger(testListLogName), &sync.Map{}, blockChainStore.store)
	validateFunc := mockValidate(list, blockChainStore.store)

	// 1. put txs[:30] and Fetch txs
	list.Put(txs[:30], protocol.RPC, validateFunc)

	fetchTxs, fetchTxIds := list.Fetch(100, nil, 99)
	require.EqualValues(t, 30, len(fetchTxs))
	require.EqualValues(t, 30, len(fetchTxIds))
	require.EqualValues(t, 0, list.Size())
	//require.EqualValues(t, 30, len(list.pendingCache))

	// 2. put txs[:30] failed due to exist in pendingCache
	list.Put(txs[:30], protocol.RPC, validateFunc)
	require.EqualValues(t, 0, list.Size())

	// 3. fetch txs nil due to not exist txs in txList
	fetchTxs, fetchTxIds = list.Fetch(100, nil, 99)
	require.EqualValues(t, 0, len(fetchTxs))
	require.EqualValues(t, 0, len(fetchTxIds))

	// 4. put txs[30:100] and Fetch txs with less number
	list.Put(txs[30:], protocol.RPC, validateFunc)
	fetchTxs, fetchTxIds = list.Fetch(10, nil, 111)
	require.EqualValues(t, 10, len(fetchTxs))
	require.EqualValues(t, 10, len(fetchTxIds))

	// 5. fetch all remaining txs
	fetchTxs, fetchTxIds = list.Fetch(100, nil, 112)
	require.EqualValues(t, 60, len(fetchTxs))
	require.EqualValues(t, 60, len(fetchTxIds))

	// 6. repeat put txs[30:100] with source = [INTERNAL] and fetch txs
	list.Put(txs[30:], protocol.INTERNAL, validateFunc)
	require.EqualValues(t, 70, list.Size())

	fetchTxs, fetchTxIds = list.Fetch(100, nil, 112)
	require.EqualValues(t, 70, len(fetchTxs))
	require.EqualValues(t, 70, len(fetchTxIds))
}

func TestTxList_Fetch_Bench(t *testing.T) {
	txs := generateTxs(1000000, false)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	blockChainStore := newMockBlockChainStore(ctrl)
	list := newTxList(logger.GetLogger(testListLogName), &sync.Map{}, blockChainStore.store)
	validateFunc := mockValidate(list, blockChainStore.store)

	// 1. put txs
	beginPut := utils.CurrentTimeMillisSeconds()
	list.Put(txs, protocol.RPC, validateFunc)
	fmt.Printf("put txs:%d, elapse time: %d\n", len(txs), utils.CurrentTimeMillisSeconds()-beginPut)

	// 2. fetch
	fetchNum := 100000
	for i := 0; i < len(txs)/fetchNum; i++ {
		beginFetch := utils.CurrentTimeMillisSeconds()
		fetchTxs, _ := list.Fetch(fetchNum, nil, 999)
		fmt.Printf("fetch txs:%d, elapse time: %d\n", len(fetchTxs), utils.CurrentTimeMillisSeconds()-beginFetch)
	}
	require.EqualValues(t, 0, list.queue.Size())
	//require.EqualValues(t, len(txs), len(list.pendingCache.))
}
