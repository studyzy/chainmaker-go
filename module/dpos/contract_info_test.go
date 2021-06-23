/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"fmt"
	"math/big"
	"testing"

	"chainmaker.org/chainmaker-go/pb/protogo/common"

	"github.com/stretchr/testify/require"

	"chainmaker.org/chainmaker-go/vm/native"
	"github.com/golang/mock/gomock"
)

func initTestImpl(t *testing.T) (*DPoSImpl, func()) {
	ctrl := gomock.NewController(t)
	mockStore := newMockBlockChainStore(ctrl)
	mockConf := newMockChainConf(ctrl)
	impl := NewDPoSImpl(mockConf, mockStore)
	return impl, func() { ctrl.Finish() }
}

func TestGetStakeAddr(t *testing.T) {
	fmt.Println(native.StakeContractAddr())
}

func TestDPoSImpl_addBalanceRwSet(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	// 1. addr1 not have balance
	rwSet, err := impl.addBalanceRwSet("addr1", "1000", &common.Block{}, nil)
	require.NoError(t, err)
	amount, ok := big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, amount.Uint64(), 1000)

	// 2. testAddr have balance in the blockChain
	rwSet, err = impl.addBalanceRwSet(testAddr, "10000", &common.Block{}, nil)
	require.NoError(t, err)
	amount, ok = big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, 10000+testAddrBalance, int(amount.Uint64()))

	// 3. testAddr have balance in the blockChain and block
	blockRwSet := make(map[string]*common.TxRWSet)
	blockRwSet["tx1"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: common.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(), Key: []byte(native.BalanceKey(testAddr)), Value: []byte("2000")},
	}}
	rwSet, err = impl.addBalanceRwSet(testAddr, "10000",
		&common.Block{Txs: []*common.Transaction{{Header: &common.TxHeader{TxId: "tx1"}}}}, blockRwSet)
	require.NoError(t, err)
	amount, ok = big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, 10000+2000, int(amount.Uint64()))
}

func TestDPoSImpl_SubBalanceRwSet(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	// 1. invalid sub amount
	rwSet, err := impl.subBalanceRwSet("addr1", "1000", &common.Block{}, nil)
	require.Error(t, err)

	// 2. sub 1000 from blockchain
	rwSet, err = impl.subBalanceRwSet(testAddr, "1000", &common.Block{}, nil)
	require.NoError(t, err)
	amount, ok := big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, testAddrBalance-1000, int(amount.Uint64()))

	// 3. sub 1000 from block
	blockRwSet := make(map[string]*common.TxRWSet)
	blockRwSet["tx1"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: common.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(), Key: []byte(native.BalanceKey(testAddr)), Value: []byte("2000")},
	}}
	rwSet, err = impl.subBalanceRwSet(testAddr, "1000",
		&common.Block{Txs: []*common.Transaction{{Header: &common.TxHeader{TxId: "tx1"}}}}, blockRwSet)
	require.NoError(t, err)
	amount, ok = big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, 2000-1000, int(amount.Uint64()))
}

func TestDPoSImpl_CompleteUnbounding(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	rwSet, err := impl.completeUnbounding(&common.Epoch{}, &common.Block{}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, len(rwSet.TxWrites))
}

func TestDPoSImpl_GetAllCandidateInfo(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	candidates, err := impl.getAllCandidateInfo()
	require.NoError(t, err)
	require.EqualValues(t, 0, len(candidates))
}
