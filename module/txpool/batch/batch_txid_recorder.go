/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	txpoolPb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"sync"
)

type batchTxIdRecorder struct {
	m sync.Map // key为batchId,value为sync.Map(txId,struct{})
}

func newBatchTxIdRecorder() *batchTxIdRecorder {
	return &batchTxIdRecorder{m: sync.Map{}}
}

func (r *batchTxIdRecorder) AddRecordWithBatch(batch *txpoolPb.TxBatch) {
	batchId := batch.GetBatchId()
	txIdMap := sync.Map{}
	if txsMap := batch.GetTxIdsMap(); len(txsMap) > 0 {
		txIdMap.Store(batchId, txsMap)
	}
	r.m.Store(batchId, &txIdMap)
}

func (r *batchTxIdRecorder) RemoveRecordWithBatch(batch *txpoolPb.TxBatch) {
	batchId := batch.GetBatchId()
	r.m.Delete(batchId)
}

func (r *batchTxIdRecorder) FindBatchIdWithTxId(txId string) (batchId int32, ok bool) {
	r.m.Range(func(key, value interface{}) bool {
		batchId = key.(int32)
		m := value.(*sync.Map)
		if val, exist := m.Load(batchId); exist {
			if txsMap, exist := val.(map[string]int32); exist {
				if _, ok = txsMap[txId]; ok {
					return false
				}
			}
		}
		return true
	})
	return
}
