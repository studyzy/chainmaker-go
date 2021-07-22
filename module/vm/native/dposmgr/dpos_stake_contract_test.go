/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dposmgr

import (
	"fmt"
	"regexp"
	"sort"
	"testing"

	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker/protocol/mock"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

var (
	DelegateAddress  = "GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"
	ValidatorAddress = "4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	NodeID           = "NodeTestc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"

	address1     = "1"
	address2     = "2"
	address3     = "3"
	address4     = "4"
	amount       = "100000000"
	biggerAmount = "1000000000"
)

func TestFormatKey(t *testing.T) {
	// test validator key
	validatorKey := "V/4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	ValidatorAddressKey := ToValidatorKey(ValidatorAddress)
	require.Equal(t, validatorKey, string(ValidatorAddressKey))

	// test validator prefix
	vPrefix := "V/"
	ValidatorPrefix := ToValidatorPrefix()
	require.Equal(t, vPrefix, string(ValidatorPrefix))

	// test delegate key
	delegationKey := "D/GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB/4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	DelegationAddressKey := ToDelegationKey(DelegateAddress, ValidatorAddress)
	require.Equal(t, delegationKey, string(DelegationAddressKey))

	// test delegate prefix
	dPrefix := "D/GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"
	delegationPrefix := ToDelegationPrefix(DelegateAddress)
	require.Equal(t, dPrefix, string(delegationPrefix))

	// test epoch key
	EpochKey := ToEpochKey("99")
	epochKey := "E/99"
	require.Equal(t, epochKey, string(EpochKey))

	// test node id
	NodeIDKey := ToNodeIDKey(NodeID)
	nodeKey := "N/NodeTestc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	require.Equal(t, nodeKey, string(NodeIDKey))

	// test unbonding delegation key
	unbondingDelegationKey := "U/\u0000\u0000\u0000\u0000\u0000\u0000\u0000c/GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB/4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	UnbondingDelegationKey := ToUnbondingDelegationKey(99, DelegateAddress, ValidatorAddress)
	require.Equal(t, unbondingDelegationKey, string(UnbondingDelegationKey))

	// test unbonding delegation prefix
	udPrefix := "U/\u0000\u0000\u0000\u0000\u0000\u0000\u0000c"
	UnbondingDelegationPrefix := ToUnbondingDelegationPrefix(99)
	require.Equal(t, udPrefix, string(UnbondingDelegationPrefix))

	// test ToReverseNodeIDKey
	ReverseNodeIDKey := "NR/nodeID"
	reverseNodeIDKey := ToReverseNodeIDKey("nodeID")
	require.Equal(t, ReverseNodeIDKey, string(reverseNodeIDKey))
}

func TestStakeContractAddr(t *testing.T) {
	// test StakeContractAddr
	StakeContractAddress := "FmGvrEHewSTDUHR4sDTCoxVq1JRRJU9QYUVsEy13zf2X"
	stakeContractAddress := StakeContractAddr()
	require.EqualValues(t, StakeContractAddress, stakeContractAddress)
}

func TestDPosStakeRuntime_NodeID(t *testing.T) {
	// init
	rt, ctx, fn := setUp(t)
	defer fn()
	// set NodeID
	params := make(map[string][]byte, 32)
	params[paramNodeID] = []byte(paramNodeID)
	bz, err := rt.SetNodeID(ctx, params)
	require.Equal(t, string(bz), paramNodeID)
	require.Nil(t, err)
	// get NodeID
	addr, err := loadSenderAddress(ctx)
	require.Nil(t, err)
	params = make(map[string][]byte, 32)
	params[paramAddress] = []byte(addr)
	bz, err = rt.GetNodeID(ctx, params)
	// asert equal
	require.Equal(t, string(bz), paramNodeID)
}

func TestDPosStakeRuntime_GetAllCandidates(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	bz, err := rt.GetAllCandidates(ctx, nil)
	require.Nil(t, err)
	vc := &syscontract.ValidatorVector{}
	err = proto.Unmarshal(bz, vc)
	require.Nil(t, err)
	require.Equal(t, len(vc.Vector), 4)
}

func TestDPosStakeRuntime_GetValidatorByAddress(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramAddress] = []byte("1")
	bz, err := rt.GetValidatorByAddress(ctx, params)
	require.Nil(t, err)
	require.Equal(t, bz, initValidator(t, "1"))
}

func TestDPosStakeRuntime_Delegate(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramTo] = []byte(address1)
	params[paramAmount] = []byte(amount)
	bz, err := rt.Delegate(ctx, params)
	require.Nil(t, err)
	d := &syscontract.Delegation{}
	err = proto.Unmarshal(bz, d)
	require.Nil(t, err)
	require.Equal(t, d, newDelegation("GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB", address1, amount))

	// test over range
	params[paramTo] = []byte(address1)
	params[paramAmount] = []byte(biggerAmount)
	bz, err = rt.Delegate(ctx, params)
	require.Equal(t, err, fmt.Errorf("address balance is not enough, contract[DPOS_ERC20] from address[GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB] balance[0] < value[1000000000]"))
	require.Equal(t, string(bz), "")
}

func TestDPosStakeRuntime_GetDelegationsByAddress(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramAddress] = []byte("1")
	bz, err := rt.GetDelegationsByAddress(ctx, params)
	d := &syscontract.DelegationInfo{}
	d.Infos = append(d.Infos, newDelegation(address1, address1, amount))
	bzExpect, err := proto.Marshal(d)
	require.Nil(t, err)
	require.Equal(t, bz, bzExpect)
}

func TestDPosStakeRuntime_GetUserDelegationByValidator(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramAddress] = []byte("1")
	bz, err := rt.GetDelegationsByAddress(ctx, params)
	d := &syscontract.DelegationInfo{}
	d.Infos = append(d.Infos, newDelegation(address1, address1, amount))
	bzExpect, err := proto.Marshal(d)
	require.Nil(t, err)
	require.Equal(t, bz, bzExpect)
}

func TestDPosStakeRuntime_UnDelegate(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	// prepare
	params := make(map[string][]byte, 32)
	params[paramTo] = []byte(address1)
	params[paramAmount] = []byte(amount)
	_, err := rt.Delegate(ctx, params)
	require.Nil(t, err)
	// test logic
	params = make(map[string][]byte, 32)
	params[paramFrom] = []byte(address1)
	params[paramAmount] = []byte(amount)
	bz, err := rt.UnDelegate(ctx, params)
	require.Nil(t, err)
	d := &syscontract.UnbondingDelegation{}
	err = proto.Unmarshal(bz, d)
	require.Nil(t, err)
	require.Equal(t, d, initUnbondingDelegation())

	// test over range
	params[paramTo] = []byte(address1)
	params[paramAmount] = []byte(biggerAmount)
	bz, err = rt.Delegate(ctx, params)
	require.Equal(t, err, fmt.Errorf("address balance is not enough, contract[DPOS_ERC20] from address[GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB] balance[0] < value[1000000000]"))
	require.Equal(t, string(bz), "")

	// test get delegation after all share undelegated
	params = make(map[string][]byte, 32)
	params[paramAddress] = []byte("1")
	bz, err = rt.GetDelegationsByAddress(ctx, params)
	di := &syscontract.DelegationInfo{}
	err = proto.Unmarshal(bz, di)
	require.Nil(t, err)
	require.Equal(t, di, initDelegationInfo(address1, address1, amount)) // validator self delegation

	params = make(map[string][]byte, 32)
	params[paramDelegatorAddress] = []byte(DelegateAddress)
	params[paramValidatorAddress] = []byte(address1)
	bz, err = rt.GetUserDelegationByValidator(ctx, params)
	require.Nil(t, bz)
	require.Equal(t, err, fmt.Errorf("no delegation as delegator: GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB, validdator: 1"))
}

func TestDPosStakeRuntime_ReadLatestEpoch(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	bz, err := rt.ReadLatestEpoch(ctx, nil)
	bzExpect, err := proto.Marshal(latestEpoch())
	require.Nil(t, err)
	require.Equal(t, bz, bzExpect)
}

func TestDPosStakeRuntime_ReadMinSelfDelegation(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	bz, err := rt.ReadMinSelfDelegation(ctx, nil)
	require.Nil(t, err)
	require.Equal(t, string(bz), amount)
}

func TestDPosStakeRuntime_ReadEpochValidatorNumber(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	bz, err := rt.ReadEpochValidatorNumber(ctx, nil)
	require.Nil(t, err)
	require.Equal(t, string(bz), "4")
}

func TestDPosStakeRuntime_UpdateEpochValidatorNumber(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramEpochValidatorNumber] = []byte("4")
	bz, err := rt.UpdateEpochValidatorNumber(ctx, params)
	require.Nil(t, err)
	require.Equal(t, string(bz), "4")

	// test over range case
	params = make(map[string][]byte, 32)
	params[paramEpochValidatorNumber] = []byte("5")
	bz, err = rt.UpdateEpochValidatorNumber(ctx, params)
	require.Equal(t, err, fmt.Errorf("new validator amount is over range, current all candidates number is: [4]"))
	require.Equal(t, string(bz), "")
}

func TestDPosStakeRuntime_ReadEpochBlockNumber(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	bz, err := rt.ReadEpochBlockNumber(ctx, nil)
	require.Nil(t, err)
	require.Equal(t, string(bz), "1")
}

func TestDPosStakeRuntime_UpdateMinSelfDelegation(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramMinSelfDelegation] = []byte(amount)
	bz, err := rt.UpdateMinSelfDelegation(ctx, params)
	require.Nil(t, err)
	require.Equal(t, string(bz), amount)

	// test over range
	params = make(map[string][]byte, 32)
	params[paramMinSelfDelegation] = []byte(biggerAmount)
	bz, err = rt.UpdateMinSelfDelegation(ctx, params)
	require.Equal(t, err, fmt.Errorf("min self delegation change over range, biggest self delegation is: [100000000]"))
	require.Equal(t, string(bz), amount)
}

func TestDPosStakeRuntime_ReadCompleteUnBoundingEpochNumber(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	bz, err := rt.ReadCompleteUnBoundingEpochNumber(ctx, nil)
	require.Nil(t, err)
	require.Equal(t, string(bz), "1")
}

func TestDPosStakeRuntime_UpdateEpochBlockNumber(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string][]byte, 32)
	params[paramEpochBlockNumber] = []byte("1")
	bz, err := rt.UpdateEpochBlockNumber(ctx, params)
	require.Nil(t, err)
	require.Equal(t, string(bz), "1")

	// test over range case
	params[paramEpochBlockNumber] = []byte("0")
	bz, err = rt.UpdateEpochBlockNumber(ctx, params)
	require.Equal(t, err, fmt.Errorf("epochBlockNumber less than or equal to 0"))
	require.Nil(t, bz)
}

func TestSortCollections(t *testing.T) {
	c := Collections{"1", "2", "3", "400000000000000", "50000000000"}
	sort.Sort(c)
	require.Equal(t, c, Collections{"400000000000000", "50000000000", "3", "2", "1"})
}

func setUp(t *testing.T) (*DPoSStakeRuntime, protocol.TxSimContext, func()) {
	dPoSStakeRuntime := NewDPoSStakeRuntime(NewLogger())
	ctrl := gomock.NewController(t)
	txSimContext := mock.NewMockTxSimContext(ctrl)

	cache := NewCacheMock()
	txSimContext.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte, value []byte) error {
			cache.Put(name, string(key), value)
			return nil
		},
	).AnyTimes()
	txSimContext.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte) ([]byte, error) {
			return cache.Get(name, string(key)), nil
		},
	).AnyTimes()
	txSimContext.EXPECT().Del(gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key []byte) error {
			return cache.Del(name, string(key))
		},
	).AnyTimes()
	txSimContext.EXPECT().GetSender().DoAndReturn(
		func() *acPb.Member {
			return &acPb.Member{
				OrgId:      "wx-org1.chainmaker.org",
				MemberInfo: ownerCert(),
				//IsFullCert: true,
			}
		},
	).AnyTimes()
	txSimContext.EXPECT().Select(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
			ks := cache.Keys()
			iter := new(kvIterator)
			for _, k := range ks {
				ok, err := regexp.MatchString("^"+syscontract.SystemContract_DPOS_STAKE.String()+"/"+string(startKey), k)
				require.Nil(t, err)
				if ok {
					bz := cache.GetByKey(k)
					kv := new(store.KV)
					kv.Key = []byte(k)
					kv.Value = bz
					iter.append(kv)
				}
			}
			return iter, nil
		},
	).AnyTimes()
	// init stake contract
	// init 4 validator
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToValidatorKey(address1)), initValidator(t, address1))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToValidatorKey(address2)), initValidator(t, address2))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToValidatorKey(address3)), initValidator(t, address3))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToValidatorKey(address4)), initValidator(t, address4))

	// init 4 delegation
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToDelegationKey(address1, address1)), initDelegation(t, address1))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToDelegationKey(address2, address2)), initDelegation(t, address2))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToDelegationKey(address3, address3)), initDelegation(t, address3))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), string(ToDelegationKey(address4, address4)), initDelegation(t, address4))

	// init basic params
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), KeyMinSelfDelegation, []byte(amount))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), KeyCompletionUnbondingEpochNumber, encodeUint64ToBigEndian(1))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), KeyEpochValidatorNumber, encodeUint64ToBigEndian(4))
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), KeyEpochBlockNumber, encodeUint64ToBigEndian(1))

	// init epoch
	cache.Put(syscontract.SystemContract_DPOS_STAKE.String(), KeyCurrentEpoch, initEpoch(t))

	// init erc20
	// init owner
	cache.Put(syscontract.SystemContract_DPOS_ERC20.String(), KeyOwner, []byte("GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"))

	// init balance
	cache.Put(syscontract.SystemContract_DPOS_ERC20.String(), BalanceKey("GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"), []byte(amount))

	//err = dPoSRuntime.setDecimals(txSimContext, Decimals)
	//require.Nil(t, err)
	// 增发指定数量的token
	//params := make(map[string][]byte, 32)
	//params[paramNameTo] = Owner
	//params[paramNameValue] = TotalSupply
	//result, err := dPoSRuntime.Mint(txSimContext, params)
	//require.Nil(t, err)
	//require.Equal(t, string(result), TotalSupply)
	return dPoSStakeRuntime, txSimContext, ctrl.Finish
}

func initValidator(t *testing.T, addr string) []byte {
	v := newValidator(addr)
	v.Status = syscontract.BondStatus_BONDED
	v.Tokens = amount
	v.DelegatorShares = amount
	v.SelfDelegation = amount
	bz, err := proto.Marshal(v)
	require.Nil(t, err)
	return bz
}

func initDelegationInfo(addr1, addr2 string, amount string) *syscontract.DelegationInfo {
	d := newDelegation(addr1, addr2, amount)
	di := &syscontract.DelegationInfo{}
	di.Infos = append(di.Infos, d)
	return di
}

func initDelegation(t *testing.T, addr string) []byte {
	d := newDelegation(addr, addr, amount)
	bz, err := proto.Marshal(d)
	require.Nil(t, err)
	return bz
}

func initUnbondingDelegation() *syscontract.UnbondingDelegation {
	ud := newUnbondingDelegation(2, DelegateAddress, address1)
	ude := newUnbondingDelegationEntry(1, 2, amount)
	ud.Entries = append(ud.Entries, ude)
	return ud
}

func initEpoch(t *testing.T) []byte {
	e := &syscontract.Epoch{
		EpochId:               1,
		ProposerVector:        []string{address1, address2, address3, address4},
		NextEpochCreateHeight: 1,
	}
	bz, err := proto.Marshal(e)
	require.Nil(t, err)
	return bz
}

func latestEpoch() *syscontract.Epoch {
	return &syscontract.Epoch{
		EpochId:               1,
		ProposerVector:        []string{address1, address2, address3, address4},
		NextEpochCreateHeight: 1,
	}
}

// iter implement
type kvIterator struct {
	kvs   []*store.KV
	idx   int
	count int
}

func (kvi *kvIterator) append(kv *store.KV) {
	kvi.kvs = append(kvi.kvs, kv)
	kvi.count++
}
func (kvi *kvIterator) Next() bool {
	kvi.idx++
	return kvi.idx <= kvi.count
}

func (kvi *kvIterator) Value() (*store.KV, error) {
	return kvi.kvs[kvi.idx-1], nil
}

func (kvi *kvIterator) Release() {
	kvi.idx = 0
	kvi.count = 0
	kvi.kvs = make([]*store.KV, 0)
}
