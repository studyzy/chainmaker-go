/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"testing"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker-go/pb/protogo/common"

	"chainmaker.org/chainmaker-go/vm/native"
)

func TestDPoSImpl_GetState(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	contractName := "test_contract"
	key1 := []byte(native.BalanceKey("addr1"))
	key2 := []byte(native.BalanceKey("addr2"))
	key3 := []byte(native.BalanceKey("addr3"))

	blk := &common.Block{
		Txs: []*common.Transaction{
			{Header: &common.TxHeader{TxId: "tx1"}},
			{Header: &common.TxHeader{TxId: "tx2"}},
			{Header: &common.TxHeader{TxId: "tx3"}},
		},
	}
	blkRwSets := make(map[string]*common.TxRWSet, 3)
	blkRwSets["tx1"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: contractName, Key: key1, Value: []byte("val1-1")},
		{ContractName: contractName, Key: key2, Value: []byte("val2-1")},
		{ContractName: contractName, Key: key3, Value: []byte("val3-1")},
	}}
	blkRwSets["tx2"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: contractName, Key: key1, Value: []byte("val1-2")},
		{ContractName: contractName, Key: key3, Value: []byte("val3-2")},
	}}
	blkRwSets["tx3"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: contractName, Key: key2, Value: []byte("val2-3")},
		{ContractName: contractName, Key: key3, Value: []byte("val3-3")},
	}}

	val, err := impl.getState(contractName, key1, blk, blkRwSets)
	require.NoError(t, err)
	require.EqualValues(t, val, []byte("val1-2"))

	val, err = impl.getState(contractName, key3, blk, blkRwSets)
	require.NoError(t, err)
	require.EqualValues(t, val, []byte("val3-3"))

	val, err = impl.getState(contractName, []byte("key4"), blk, blkRwSets)
	require.NoError(t, err)
	require.EqualValues(t, len(val), 0)
}
