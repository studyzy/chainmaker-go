/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/common/queue/lockfreequeue"
	commonpb "chainmaker.org/chainmaker/pb-go/common"

	"github.com/stretchr/testify/require"
)

func TestBatchTxPool_PopTxsFromQueue(t *testing.T) {
	for i := 0; i < 100; i++ {
		pool := NewBatchTxPool("nodeId", "test-chain", nil, nil, nil, nil)
		pool.txQueue = lockfreequeue.NewQueue(uint32(pool.batchMaxSize))
		for i := 0; i < int(pool.batchMaxSize); i++ {
			pool.txQueue.Push(&commonpb.Transaction{Payload: &commonpb.Payload{TxId: utils.GetRandTxId()}})
		}
		pool.batchCreateTimeout = time.Second
		txs, txIdToIndex := pool.popTxsFromQueue()
		require.EqualValues(t, len(txs), len(txIdToIndex))
		for txId, index := range txIdToIndex {
			require.EqualValues(t, txId, txs[index].Payload.TxId, "txId should be equal")
		}
	}
}
