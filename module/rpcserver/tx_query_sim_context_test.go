/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTxQuerySimContext(t *testing.T) {

	tx := &commonPb.Transaction{
		Header: &commonPb.Payload{
			ChainId:        "",
			Sender:         nil,
			TxType:         0,
			TxId:           "1",
			Timestamp:      0,
			ExpirationTime: 0,
		},
		RequestPayload:   nil,
		RequestSignature: nil,
		Result:           nil,
	}
	var txSimContextImpl protocol.TxSimContext
	txSimContextImpl = &txQuerySimContextImpl{
		tx:            tx,
		txReadKeyMap:  make(map[string]*commonPb.TxRead, 8),
		txWriteKeyMap: make(map[string]*commonPb.TxWrite, 8),
		sqlRowCache:   make(map[int32]protocol.SqlRows, 0),
		kvRowCache:    make(map[int32]protocol.StateIterator, 0),
		txWriteKeySql: make([]*commonPb.TxWrite, 0),
	}

	contractName := "contract1"
	txSimContextImpl.Put(contractName, []byte("K1"), []byte("V1"))
	txSimContextImpl.Put(contractName, []byte("K2"), []byte("V2"))
	txSimContextImpl.Put(contractName, []byte("K2"), []byte("V3"))
	txSimContextImpl.Put(contractName, []byte("K3"), []byte("V1"))

	v1, _ := txSimContextImpl.Get(contractName, []byte("K1"))
	v2, _ := txSimContextImpl.Get(contractName, []byte("K2"))
	v3, _ := txSimContextImpl.Get(contractName, []byte("K3"))
	txSimContextImpl.Get(contractName, []byte("K3"))
	require.Equal(t, string(v1), "V1")
	require.Equal(t, string(v2), "V3")
	require.Equal(t, string(v3), "V1")

	set := txSimContextImpl.GetTxRWSet(true)
	require.Equal(t, len(set.TxWrites), 3)
	require.Equal(t, len(set.TxReads), 3)

}
