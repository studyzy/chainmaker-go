/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package single

import (
	"testing"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/protocol"
)

func generateTxsBySource(num int, isConfig bool) (rpcTxs, p2pTxs, internalTxs *mempoolTxs) {
	rpcTxs = &mempoolTxs{isConfigTxs: isConfig, source: protocol.RPC}
	p2pTxs = &mempoolTxs{isConfigTxs: isConfig, source: protocol.P2P}
	internalTxs = &mempoolTxs{isConfigTxs: isConfig, source: protocol.INTERNAL}
	txType := commonPb.TxType_INVOKE_CONTRACT
	//if !isConfig {
	//	txType = commonPb.TxType_INVOKE_CONTRACT
	//}

	for i := 0; i < num; i++ {

		contractName := syscontract.SystemContract_CHAIN_CONFIG.String()

		if !isConfig {
			contractName = contract
		}

		rpcTxs.txs = append(rpcTxs.txs, &commonPb.Transaction{Payload: &commonPb.Payload{TxId: utils.GetRandTxId(), TxType: txType, Method: "SetConfig", ContractName: contractName}})
		p2pTxs.txs = append(p2pTxs.txs, &commonPb.Transaction{Payload: &commonPb.Payload{TxId: utils.GetRandTxId(), TxType: txType, Method: "SetConfig", ContractName: contractName}})
		internalTxs.txs = append(internalTxs.txs, &commonPb.Transaction{Payload: &commonPb.Payload{TxId: utils.GetRandTxId(), TxType: txType, Method: "SetConfig", ContractName: contractName}})
	}
	return
}

func TestAddMemoryTxs(t *testing.T) {
	cache := newTxCache()
	rpcTxs, p2pTxs, internalTxs := generateTxsBySource(10, false)
	cache.addMemoryTxs(rpcTxs)
	require.EqualValues(t, 10, cache.txCount())
	cache.addMemoryTxs(p2pTxs)
	require.EqualValues(t, 20, cache.txCount())
	cache.addMemoryTxs(internalTxs)
	require.EqualValues(t, 30, cache.txCount())
}

func TestMergeAndSplitTxsBySource(t *testing.T) {
	cache := newTxCache()
	rpcTxs, p2pTxs, internalTxs := generateTxsBySource(30, false)
	cache.addMemoryTxs(rpcTxs)
	cache.addMemoryTxs(rpcTxs)
	cache.addMemoryTxs(p2pTxs)
	cache.addMemoryTxs(internalTxs)

	tmpRpcTxs, tmpP2PTxs, tmpInternalTxs := cache.mergeAndSplitTxsBySource(nil)
	require.EqualValues(t, append(rpcTxs.txs, rpcTxs.txs...), tmpRpcTxs)
	require.EqualValues(t, p2pTxs.txs, tmpP2PTxs)
	require.EqualValues(t, internalTxs.txs, tmpInternalTxs)
}

func TestIsFlushByTxCount(t *testing.T) {
	cache := newTxCache()
	cache.flushThreshold = 20
	rpcTxs, _, _ := generateTxsBySource(10, false)
	cache.addMemoryTxs(rpcTxs)
	require.False(t, cache.isFlushByTxCount(nil))
	require.True(t, cache.isFlushByTxCount(rpcTxs))

	cache.addMemoryTxs(rpcTxs)
	require.True(t, cache.isFlushByTxCount(nil))
}

func TestIsFlushByTime(t *testing.T) {
	cache := newTxCache()
	cache.flushTimeOut = 200 * time.Microsecond
	require.True(t, cache.isFlushByTime())
	cache.reset()
	require.False(t, cache.isFlushByTime())
	time.Sleep(time.Millisecond * 200)
	require.True(t, cache.isFlushByTime())
}

func TestReset(t *testing.T) {
	cache := newTxCache()
	rpcTxs, _, _ := generateTxsBySource(10, false)
	cache.addMemoryTxs(rpcTxs)
	require.EqualValues(t, 10, cache.txCount())
	cache.reset()
	require.EqualValues(t, 0, cache.txCount())
	require.EqualValues(t, 0, len(cache.txs))
}
