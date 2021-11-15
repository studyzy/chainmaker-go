/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"fmt"
	"math/big"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/vm-native/v2/dposmgr"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func initTestImpl(t *testing.T) (*DPoSImpl, func()) {
	ctrl := gomock.NewController(t)
	mockStore := newMockBlockChainStore(ctrl)
	mockConf := newMockChainConf(ctrl)
	impl := NewDPoSImpl(mockConf, mockStore)
	return impl, func() { ctrl.Finish() }
}

func TestGetStakeAddr(t *testing.T) {
	fmt.Println(dposmgr.StakeContractAddr())
}

func TestDPoSImpl_addBalanceRwSet(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	// 1. addr1 not have balance
	rwSet, _, err := impl.addBalanceRwSet("addr1", big.NewInt(0), "1000")
	require.NoError(t, err)
	amount, ok := big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, amount.Uint64(), 1000)

	// 2. testAddr have balance in the blockChain
	rwSet, _, err = impl.addBalanceRwSet(testAddr, big.NewInt(int64(testAddrBalance)), "10000")
	require.NoError(t, err)
	amount, ok = big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, 10000+testAddrBalance, int(amount.Uint64()))

	// 3. testAddr have balance in the blockChain and block
	blockRwSet := make(map[string]*common.TxRWSet)
	blockRwSet["tx1"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: syscontract.SystemContract_DPOS_ERC20.String(), Key: []byte(dposmgr.BalanceKey(testAddr)), Value: []byte("2000")},
	}}
	balance, err := impl.balanceOf(testAddr, &common.Block{Txs: []*common.Transaction{{Payload: &common.Payload{TxId: "tx1"}}}}, blockRwSet)
	require.NoError(t, err)
	rwSet, _, err = impl.addBalanceRwSet(testAddr, balance, "10000")
	require.NoError(t, err)
	amount, ok = big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, 10000+2000, int(amount.Uint64()))
}

func TestDPoSImpl_SubBalanceRwSet(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	// 1. invalid sub amount
	rwSet, _, err := impl.subBalanceRwSet("addr1", big.NewInt(0), "1000")
	require.Error(t, err)

	// 2. sub 1000 from blockchain
	rwSet, _, err = impl.subBalanceRwSet(testAddr, big.NewInt(9999), "1000")
	require.NoError(t, err)
	amount, ok := big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, testAddrBalance-1000, int(amount.Uint64()))

	// 3. sub 1000 from block
	blockRwSet := make(map[string]*common.TxRWSet)
	blockRwSet["tx1"] = &common.TxRWSet{TxWrites: []*common.TxWrite{
		{ContractName: syscontract.SystemContract_DPOS_ERC20.String(), Key: []byte(dposmgr.BalanceKey(testAddr)), Value: []byte("2000")},
	}}
	balance, err := impl.balanceOf(testAddr, &common.Block{Txs: []*common.Transaction{{Payload: &common.Payload{TxId: "tx1"}}}}, blockRwSet)
	require.NoError(t, err)
	rwSet, _, err = impl.subBalanceRwSet(testAddr, balance, "1000")
	require.NoError(t, err)
	amount, ok = big.NewInt(0).SetString(string(rwSet.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, 2000-1000, int(amount.Uint64()))
}

func TestDPoSImpl_CompleteUnbounding(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	rwSet, err := impl.completeUnbounding(&syscontract.Epoch{}, &common.Block{}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, len(rwSet.TxWrites))
}

//TODO: please use mock store to replace storeFactory.NewStore
//func TestDPoSImpl_GetUnboundingEntries(t *testing.T) {
//	impl, fn := initDPoSWithStore(t)
//	defer fn()
//
//	entries, err := impl.getUnboundingEntries(&syscontract.Epoch{EpochId: 10})
//	require.NoError(t, err)
//	require.EqualValues(t, 0, len(entries))
//
//	blk, blkRwSet := generateUnboundingBlock(t, 4, 10, 1, 10)
//	require.NoError(t, impl.stateDB.PutBlock(blk, blkRwSet))
//	entries, err = impl.getUnboundingEntries(&syscontract.Epoch{EpochId: 10})
//	require.NoError(t, err)
//	require.EqualValues(t, 4, len(entries))
//
//	blk, blkRwSet = generateUnboundingBlock(t, 4, 10, 2, 20)
//	require.NoError(t, impl.stateDB.PutBlock(blk, blkRwSet))
//	entries, err = impl.getUnboundingEntries(&syscontract.Epoch{EpochId: 20})
//	require.NoError(t, err)
//	require.EqualValues(t, 4, len(entries))
//
//	entries, err = impl.getUnboundingEntries(&syscontract.Epoch{EpochId: 30})
//	require.NoError(t, err)
//	require.EqualValues(t, 0, len(entries))
//}

func TestDPoSImpl_CreateUnboundingRwSet(t *testing.T) {
	impl, fn := initTestImpl(t)
	defer fn()

	// 1. no entries
	rwSet, err := impl.createUnboundingRwSet(nil, &common.Block{}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, len(rwSet.TxWrites))

	// 2. create entries and create rwSet
	entries := createUndelegationEntries()
	rwSet, err = impl.createUnboundingRwSet(entries, &common.Block{}, nil)
	require.NoError(t, err)
	for _, v := range rwSet.TxWrites {
		fmt.Println(v)
	}
	require.EqualValues(t, 8, len(rwSet.TxWrites))
	last := rwSet.TxWrites[len(rwSet.TxWrites)-1]
	stakeAddrBalance, ok := big.NewInt(0).SetString(string(last.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, int(stakeAddrBalance.Uint64()), 6000)
	lastAddr1 := rwSet.TxWrites[len(rwSet.TxWrites)-2]
	lastAddrBalance1, ok := big.NewInt(0).SetString(string(lastAddr1.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, int(lastAddrBalance1.Uint64()), 2000)
	lastAddr2 := rwSet.TxWrites[len(rwSet.TxWrites)-4]
	lastAddrBalance2, ok := big.NewInt(0).SetString(string(lastAddr2.Value), 10)
	require.True(t, ok)
	require.EqualValues(t, int(lastAddrBalance2.Uint64()), 2000)
}

func createUndelegationEntries() []*syscontract.UnbondingDelegation {
	delAddr1 := fmt.Sprintf("delegatorAddr-%d", 1)
	valAddr1 := fmt.Sprintf("validatorAddr-%d", 1)
	delAddr2 := fmt.Sprintf("delegatorAddr-%d", 2)
	valAddr2 := fmt.Sprintf("validatorAddr-%d", 2)

	entries := make([]*syscontract.UnbondingDelegation, 0, 4)
	for i := 0; i < 4; i++ {
		delAddr := delAddr1
		valAddr := valAddr1
		if i%2 == 0 {
			delAddr = delAddr2
			valAddr = valAddr2
		}
		entry := &syscontract.UnbondingDelegation{
			EpochId: "8", DelegatorAddress: delAddr, ValidatorAddress: valAddr,
			Entries: []*syscontract.UnbondingDelegationEntry{
				{CreationEpochId: 1, CompletionEpochId: 8, Amount: "1000"},
			},
		}
		entries = append(entries, entry)
	}
	return entries
}

//TODO: please use mock store to replace storeFactory.NewStore
//func TestDPoSImpl_GetAllCandidateInfo(t *testing.T) {
//	impl, fn := initDPoSWithStore(t)
//	defer fn()
//
//	// 0. no candidates
//	candidates, err := impl.getAllCandidateInfo()
//	require.NoError(t, err)
//	require.EqualValues(t, 0, len(candidates))
//
//	// 1. init 6 candidates
//	blk, blkRwSet := generateCandidateBlockAndRwSet(t, 6, 10, 1)
//	require.NoError(t, impl.stateDB.PutBlock(blk, blkRwSet))
//	candidates, err = impl.getAllCandidateInfo()
//	require.NoError(t, err)
//	require.EqualValues(t, 6, len(candidates))
//
//	// 2. add other 10 candidates
//	blk, blkRwSet = generateCandidateBlockAndRwSet(t, 10, 20, 2)
//	require.NoError(t, impl.stateDB.PutBlock(blk, blkRwSet))
//	candidates, err = impl.getAllCandidateInfo()
//	require.NoError(t, err)
//	require.EqualValues(t, 16, len(candidates))
//}

//TODO: please use mock store to replace storeFactory.NewStore
//func initDPoSWithStore(t *testing.T) (*DPoSImpl, func()) {
//	ctrl := gomock.NewController(t)
//	mockConf := newMockChainConf(ctrl)
//
//	var storeFactory store.Factory
//	storeLogger := logger.GetLoggerByChain(logger.MODULE_STORAGE, "test-chain")
//	testStore, err := storeFactory.NewStore("test-chain", &conf.StorageConfig{
//		StorePath:              "test/data",
//		DisableContractEventDB: true,
//		StateDbConfig: &conf.DbConfig{
//			Provider:      "leveldb",
//			LevelDbConfig: &localconf.LevelDbConfig{StorePath: "test/state"},
//		},
//		BlockDbConfig: &conf.DbConfig{
//			Provider:      "leveldb",
//			LevelDbConfig: &localconf.LevelDbConfig{StorePath: "test/block"},
//		},
//		ResultDbConfig: &conf.DbConfig{
//			Provider:      "leveldb",
//			LevelDbConfig: &localconf.LevelDbConfig{StorePath: "test/result"},
//		},
//	}, storeLogger)
//	defer require.NoError(t, err)
//	blk, blkRwSet := generateBlockWithStakeConfig()
//	require.NoError(t, testStore.PutBlock(blk, blkRwSet))
//	impl := NewDPoSImpl(mockConf, testStore)
//	return impl, func() {
//		ctrl.Finish()
//		require.NoError(t, os.RemoveAll("test"))
//	}
//}

var testSelfMinDelegation = "10000000"

func generateBlockWithStakeConfig() (*common.Block, []*common.TxRWSet) {
	var (
		blk = &common.Block{
			Header: &common.BlockHeader{ChainId: "test-chain", BlockHeight: 0},
			Txs: []*common.Transaction{
				{Payload: &common.Payload{TxId: "config-tx"}},
			},
		}
		rwSet = make([]*common.TxRWSet, 0, 1)
	)
	rwSet = append(rwSet, &common.TxRWSet{
		TxWrites: []*common.TxWrite{
			{
				ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
				Key:          []byte(dposmgr.KeyMinSelfDelegation), Value: []byte(testSelfMinDelegation),
			},
		},
	})
	return blk, rwSet
}

func generateCandidateBlockAndRwSet(t *testing.T, txNum, base int, blockHeight uint64) (*common.Block, []*common.TxRWSet) {
	var (
		blk = &common.Block{
			Header: &common.BlockHeader{ChainId: "test-chain", BlockHeight: blockHeight},
		}
		rwSet = make([]*common.TxRWSet, 0, txNum)
	)

	for i := 0; i < txNum; i++ {
		txId := fmt.Sprintf("txId-%d", i+1)
		blk.Txs = append(blk.Txs, &common.Transaction{
			Payload: &common.Payload{TxId: txId},
		})

		valAddr := fmt.Sprintf("validatorAddr-%d-%d", base, i+1)
		validator := &syscontract.Validator{
			ValidatorAddress: valAddr,
			Jailed:           false,
			Status:           syscontract.BondStatus_BONDED,
			SelfDelegation:   testSelfMinDelegation,
		}
		bz, err := validator.Marshal()
		require.NoError(t, err)
		rwSet = append(rwSet, &common.TxRWSet{
			TxWrites: []*common.TxWrite{
				{
					ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
					Key:          dposmgr.ToValidatorKey(valAddr), Value: bz,
				},
			},
		})
	}
	return blk, rwSet
}

func generateUnboundingBlock(t *testing.T, txNum, base int, blockHeight uint64, completeEpoch uint64) (*common.Block, []*common.TxRWSet) {
	var (
		blk = &common.Block{
			Header: &common.BlockHeader{ChainId: "test-chain", BlockHeight: blockHeight},
		}
		rwSet = make([]*common.TxRWSet, 0, txNum)
	)

	for i := 0; i < txNum; i++ {
		txId := fmt.Sprintf("txId-%d", i+1)
		blk.Txs = append(blk.Txs, &common.Transaction{
			Payload: &common.Payload{TxId: txId},
		})

		delAddr := fmt.Sprintf("delegatorAddr-%d-%d", base, i+1)
		valAddr := fmt.Sprintf("validatorAddr-%d-%d", base, i+1)
		entry := &syscontract.UnbondingDelegation{
			EpochId:          fmt.Sprintf("%d", completeEpoch),
			DelegatorAddress: delAddr,
			ValidatorAddress: valAddr,
			Entries: []*syscontract.UnbondingDelegationEntry{
				{CreationEpochId: completeEpoch - 1, CompletionEpochId: completeEpoch, Amount: "1000"},
			},
		}
		bz, err := entry.Marshal()
		require.NoError(t, err)
		rwSet = append(rwSet, &common.TxRWSet{
			TxWrites: []*common.TxWrite{
				{
					ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
					Key:          dposmgr.ToUnbondingDelegationKey(completeEpoch, delAddr, valAddr), Value: bz,
				},
			},
		})
	}
	return blk, rwSet
}
