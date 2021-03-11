/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	txpoolPb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"testing"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker-go/utils"
)

func TestBatchTxIdRecorder_FindBatchIdWithTxId(t *testing.T) {
	recorder := newBatchTxIdRecorder()

	batch := txpoolPb.TxBatch{
		BatchId:  9,
		TxIdsMap: make(map[string]int32),
	}
	for i := 0; i < 1000; i++ {
		txid := utils.GetRandTxId()
		batch.Txs = append(batch.Txs, &commonPb.Transaction{Header: &commonPb.TxHeader{TxId: txid}})
		batch.TxIdsMap[txid] = int32(i)
	}

	recorder.AddRecordWithBatch(&batch)
	for _, tx := range batch.Txs {
		batchId, ok := recorder.FindBatchIdWithTxId(tx.Header.TxId)
		require.EqualValues(t, 9, batchId)
		require.True(t, ok)
	}

	recorder.RemoveRecordWithBatch(&batch)
	for _, tx := range batch.Txs {
		_, ok := recorder.FindBatchIdWithTxId(tx.Header.TxId)
		require.False(t, ok)
	}

}
