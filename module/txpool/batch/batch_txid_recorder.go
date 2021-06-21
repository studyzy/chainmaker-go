/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"sync"

	txpoolPb "chainmaker.org/chainmaker/pb-go/txpool"
)

type batchTxIdRecorder struct {
	m sync.Map // format: key:batchId, value: map[txId]txIndex
}

func newBatchTxIdRecorder() *batchTxIdRecorder {
	return &batchTxIdRecorder{m: sync.Map{}}
}

func (r *batchTxIdRecorder) AddRecordWithBatch(batch *txpoolPb.TxBatch) {
	if txsMap := batch.GetTxIdsMap(); len(txsMap) > 0 {
		r.m.Store(batch.GetBatchId(), txsMap)
	}
}

func (r *batchTxIdRecorder) RemoveRecordWithBatch(batch *txpoolPb.TxBatch) {
	batchId := batch.GetBatchId()
	r.m.Delete(batchId)
}

func (r *batchTxIdRecorder) FindBatchIdWithTxId(txId string) (batchId int32, txIndex int32, ok bool) {
	batchId = -1
	r.m.Range(func(key, value interface{}) bool {
		txsMapInfo := value.(map[string]int32)
		if index, exist := txsMapInfo[txId]; exist {
			ok = true
			txIndex = index
			batchId = key.(int32)
			return false
		}
		return true
	})
	return
}
