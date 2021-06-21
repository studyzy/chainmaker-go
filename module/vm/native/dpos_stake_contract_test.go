package native

import (
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker/protocol/mock"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"regexp"
	"sort"
	"testing"
)

const (
	DelegateAddress = "GMx5CwXvH9FyGwD5CbHsCXfM6XmAyzjb9iVRDiYBTxdB"
	ValidatorAddress = "4yp3FUSrc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"
	NodeID = "NodeTestc1jyCgHMXswPeSE9N4Dnys1Hsg3NtBbzu2F4"

	address1 = "1"
	address2 = "2"
	address3 = "3"
	address4 = "4"
	amount = "100000000"
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
	StakeContractAddress := "BPoXnWti3XvkXY2TjFBhozqWasDP5BRvcUYNA86YgosQ"
	stakeContractAddress := StakeContractAddr()
	require.Equal(t, StakeContractAddress, stakeContractAddress)
}

func TestDPosStakeRuntime_NodeID(t *testing.T) {
	// init
	rt, ctx, fn := setUp(t)
	defer fn()
	// set NodeID
	params := make(map[string]string, 32)
	params[paramNodeID] = paramNodeID
	bz, err := rt.SetNodeID(ctx, params)
	require.Equal(t, string(bz), paramNodeID)
	require.Nil(t, err)
	// get NodeID
	addr, err := loadSenderAddress(ctx)
	require.Nil(t, err)
	params = make(map[string]string, 32)
	params[paramAddress] = addr
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
	vc := &commonPb.ValidatorVector{}
	err = proto.Unmarshal(bz, vc)
	require.Nil(t, err)
	require.Equal(t, len(vc.Vector), 4)
}

func TestDPosStakeRuntime_GetValidatorByAddress(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string]string, 32)
	params[paramAddress] = "1"
	bz, err := rt.GetValidatorByAddress(ctx, params)
	require.Nil(t, err)
	require.Equal(t, bz, initValidator(t, "1"))
}

func TestDPosStakeRuntime_Delegate(t *testing.T) {
	//// init erc20
	//ert, ctx1, fn1 := initEnv(t)
	//defer fn1()
	//// init stake
	//srt, ctx2, fn2 := setUp(t)
	//defer fn2()
}

func TestDPosStakeRuntime_GetDelegationsByAddress(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string]string, 32)
	params[paramAddress] = "1"
	bz, err := rt.GetDelegationsByAddress(ctx, params)
	d := &commonPb.DelegationInfo{}
	d.Infos = append(d.Infos, newDelegation("1", "1", "100000000"))
	bzExpect, err := proto.Marshal(d)
	require.Nil(t, err)
	require.Equal(t, bz, bzExpect)
}

func TestDPosStakeRuntime_GetUserDelegationByValidator(t *testing.T) {
	rt, ctx, fn := setUp(t)
	defer fn()
	// call api
	params := make(map[string]string, 32)
	params[paramAddress] = "1"
	bz, err := rt.GetDelegationsByAddress(ctx, params)
	d := &commonPb.DelegationInfo{}
	d.Infos = append(d.Infos, newDelegation("1", "1", "100000000"))
	bzExpect, err := proto.Marshal(d)
	require.Nil(t, err)
	require.Equal(t, bz, bzExpect)
}

func TestDPosStakeRuntime_UnDelegate(t *testing.T) {

}

func TestDPosStakeRuntime_ReadLatestEpoch(t *testing.T) {

}

func TestDPosStakeRuntime_ReadMinSelfDelegation(t *testing.T) {

}

func TestDPosStakeRuntime_ReadEpochValidatorNumber(t *testing.T) {

}
func TestDPosStakeRuntime_UpdateEpochValidatorNumber(t *testing.T) {

}
func TestDPosStakeRuntime_ReadEpochBlockNumber(t *testing.T) {

}
func TestDPosStakeRuntime_UpdateMinSelfDelegation(t *testing.T) {

}
func TestDPosStakeRuntime_ReadCompleteUnBoundingEpochNumber(t *testing.T) {

}
func TestDPosStakeRuntime_UpdateEpochBlockNumber(t *testing.T) {

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
	txSimContext.EXPECT().GetSender().DoAndReturn(
		func() *acPb.SerializedMember {
			return &acPb.SerializedMember{
				OrgId:      "wx-org1.chainmaker.org",
				MemberInfo: ownerCert(),
				IsFullCert: true,
			}
		},
	).AnyTimes()
	txSimContext.EXPECT().Select(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
			ks := cache.Keys()
			iter := new(kvIterator)
			for _, k := range ks {
				ok, err := regexp.MatchString("^" + commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String() + "/" + string(startKey), k)
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

	// init 4 validator
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToValidatorKey(address1)), initValidator(t, address1))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToValidatorKey(address2)), initValidator(t, address2))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToValidatorKey(address3)), initValidator(t, address3))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToValidatorKey(address4)), initValidator(t, address4))

	// init 4 delegation
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToDelegationKey("1", "1")), initDelegation(t, "1"))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToDelegationKey("2", "2")), initDelegation(t, "2"))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToDelegationKey("3", "3")), initDelegation(t, "3"))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), string(ToDelegationKey("4", "4")), initDelegation(t, "4"))


	// init basic params
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), KeyMinSelfDelegation, []byte("100000000"))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), KeyCompletionUnbondingEpochNumber, encodeUint64ToBigEndian(1))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), KeyEpochValidatorNumber, encodeUint64ToBigEndian(4))
	cache.Put(commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), KeyEpochBlockNumber, encodeUint64ToBigEndian(1))

	//err := dPoSStakeRuntime.setOwner(txSimContext, Owner)
	//require.Nil(t, err)
	//err = dPoSRuntime.setDecimals(txSimContext, Decimals)
	//require.Nil(t, err)
	// 增发指定数量的token
	//params := make(map[string]string, 32)
	//params[paramNameTo] = Owner
	//params[paramNameValue] = TotalSupply
	//result, err := dPoSRuntime.Mint(txSimContext, params)
	//require.Nil(t, err)
	//require.Equal(t, string(result), TotalSupply)
	return dPoSStakeRuntime, txSimContext, ctrl.Finish
}

func initValidator(t *testing.T, addr string) []byte {
	v := newValidator(addr)
	v.Status = commonPb.BondStatus_Bonded
	v.Tokens = "100000000"
	v.DelegatorShares = "100000000"
	v.SelfDelegation = "100000000"
	bz, err := proto.Marshal(v)
	require.Nil(t, err)
	return bz
}

func initDelegation(t *testing.T, addr string) []byte {
	d := newDelegation(addr, addr, "100000000")
	bz, err := proto.Marshal(d)
	require.Nil(t, err)
	return bz
}

// iter implement
type kvIterator struct {
	kvs []*store.KV
	idx       int
	count     int
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
