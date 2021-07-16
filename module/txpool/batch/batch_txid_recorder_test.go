/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	txpoolPb "chainmaker.org/chainmaker/pb-go/txpool"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker-go/utils"
)

func TestBatchTxIdRecorder_FindBatchIdWithTxId(t *testing.T) {
	recorder := newBatchTxIdRecorder()

	// 1. add batch to recorder
	batch9 := txpoolPb.TxBatch{
		BatchId:  9,
		Txs:      make([]*commonPb.Transaction, 1000),
		TxIdsMap: make(map[string]int32),
	}
	for i := 0; i < 1000; i++ {
		txId := utils.GetRandTxId()
		batch9.TxIdsMap[txId] = int32(i)
		batch9.Txs[i] = &commonPb.Transaction{Payload: &commonPb.Payload{TxId: txId}}
	}
	recorder.AddRecordWithBatch(&batch9)

	batch10 := txpoolPb.TxBatch{
		BatchId:  10,
		Txs:      make([]*commonPb.Transaction, 1000),
		TxIdsMap: make(map[string]int32, 1000),
	}
	for i := 0; i < 1000; i++ {
		txId := utils.GetRandTxId()
		batch10.TxIdsMap[txId] = int32(i)
		batch10.Txs[i] = &commonPb.Transaction{Payload: &commonPb.Payload{TxId: txId}}
	}
	recorder.AddRecordWithBatch(&batch10)

	// 2. check existence in recorder
	for i, tx := range batch9.Txs {
		batchId, txIndex, ok := recorder.FindBatchIdWithTxId(tx.Payload.TxId)
		require.True(t, ok)
		require.EqualValues(t, i, txIndex)
		require.EqualValues(t, 9, batchId)
	}
	for i, tx := range batch10.Txs {
		batchId, txIndex, ok := recorder.FindBatchIdWithTxId(tx.Payload.TxId)
		require.True(t, ok)
		require.EqualValues(t, i, txIndex)
		require.EqualValues(t, 10, batchId)
	}

	// 3. remove batch from recorder and check existence
	recorder.RemoveRecordWithBatch(&batch9)
	for _, tx := range batch9.Txs {
		batchId, _, ok := recorder.FindBatchIdWithTxId(tx.Payload.TxId)
		require.False(t, ok)
		require.EqualValues(t, -1, batchId)
	}
	for i, tx := range batch10.Txs {
		batchId, txIndex, ok := recorder.FindBatchIdWithTxId(tx.Payload.TxId)
		require.True(t, ok)
		require.EqualValues(t, i, txIndex)
		require.EqualValues(t, 10, batchId)
	}
}
