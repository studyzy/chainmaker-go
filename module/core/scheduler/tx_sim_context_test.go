/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"crypto/sha256"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSha256(t *testing.T) {
	fmt.Println(sha256.Sum256([]byte("aaa")))
	fmt.Println(sha256.Sum256([]byte("bbb")))
	fmt.Println(sha256.Sum256([]byte("bbb")))
}

func TestTxSimContext(t *testing.T) {

	tx := &commonpb.Transaction{
		Header: &commonpb.TxHeader{
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
	var txSimContext protocol.TxSimContext
	txSimContext = &txSimContextImpl{
		txExecSeq:     0,
		tx:            tx,
		txReadKeyMap:  make(map[string]*commonpb.TxRead, 8),
		txWriteKeyMap: make(map[string]*commonpb.TxWrite, 8),
		sqlRowCache:   make(map[int32]protocol.SqlRows, 0),
		txWriteKeySql: make([]*commonpb.TxWrite, 0),
	}

	contractName := "contract1"
	txSimContext.Put(contractName, []byte("K1"), []byte("V1"))
	txSimContext.Put(contractName, []byte("K2"), []byte("V2"))
	txSimContext.Put(contractName, []byte("K2"), []byte("V3"))
	txSimContext.Put(contractName, []byte("K3"), []byte("V1"))

	v1, _ := txSimContext.Get(contractName, []byte("K1"))
	v2, _ := txSimContext.Get(contractName, []byte("K2"))
	v3, _ := txSimContext.Get(contractName, []byte("K3"))
	txSimContext.Get(contractName, []byte("K3"))
	require.Equal(t, string(v1), "V1")
	require.Equal(t, string(v2), "V3")
	require.Equal(t, string(v3), "V1")

	set := txSimContext.GetTxRWSet()
	require.Equal(t, len(set.TxWrites), 3)
	require.Equal(t, len(set.TxReads), 3)
}
