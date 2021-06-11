package dpos

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"chainmaker.org/chainmaker-go/pb/protogo/common"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	dpospb "chainmaker.org/chainmaker-go/pb/protogo/dpos"
	"chainmaker.org/chainmaker-go/vm/native"

	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const ModuleName = "dpos_module"

// getEpochInfo get epoch info from ledger
func (impl *DPoSImpl) getEpochInfo() (*commonpb.Epoch, error) {
	epoch, err := GetLatestEpochInfo(impl.stateDB)
	if err != nil {
		impl.log.Errorf("get epoch failed, reason: %s", err)
		return nil, err
	}
	impl.log.Debugf("epoch info: %s", epoch.String())
	return epoch, nil
}

func (impl *DPoSImpl) getNodeIDsFromValidators(epoch *common.Epoch) ([]string, error) {
	nodeIDs, err := GetNodeIDsFromValidators(impl.stateDB, epoch.ProposerVector)
	if err != nil {
		impl.log.Errorf("get nodeids from ledger failed, reason: %s", err)
		return nil, err
	}
	impl.log.Debugf("curr validators nodeID: %v", nodeIDs)
	return nodeIDs, nil
}

// getAllCandidateInfo get all candidates from ledger
func (impl *DPoSImpl) getAllCandidateInfo() ([]*dpospb.CandidateInfo, error) {
	prefix := native.ToValidatorPrefix()
	iterRange := util.BytesPrefix(prefix)
	iter, err := impl.stateDB.SelectObject(commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		impl.log.Errorf("read contract: %s error: %s", commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), err)
		return nil, err
	}
	defer iter.Release()
	minSelfDelegationBz, err := impl.stateDB.ReadObject(commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), []byte(native.KeyMinSelfDelegation))
	if err != nil {
		impl.log.Errorf("get selfMinDelegation from contract %s failed, reason: %s", commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), err)
		return nil, err
	}
	minSelfDelegation, ok := big.NewInt(0).SetString(string(minSelfDelegationBz), 10)
	if !ok {
		err := fmt.Errorf("invalid minSelfDelegation in stake contract")
		impl.log.Errorf("%s", err)
		return nil, err
	}
	impl.log.Debugf("minSelfDelegation: %s", minSelfDelegation.String())
	vals := make([]*commonpb.Validator, 0, 10)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			impl.log.Errorf("iterator read error: %s", err)
			return nil, err
		}
		val := commonpb.Validator{}
		if err = proto.Unmarshal(kv.Value, &val); err != nil {
			impl.log.Errorf("unmarshal validator failed, reason: %s", err)
			return nil, err
		}
		vals = append(vals, &val)
	}
	if len(vals) == 0 {
		impl.log.Warnf("not find candidate .")
		return nil, nil
	}
	candidates := make([]*dpospb.CandidateInfo, 0, len(vals))
	for i := 0; i < len(vals); i++ {
		selfDelegation, ok := big.NewInt(0).SetString(vals[i].SelfDelegation, 10)
		if !ok {
			impl.log.Errorf("validator selfDelegation not parse to big.Int, actual: %s ", vals[i].SelfDelegation)
			return nil, fmt.Errorf("validator selfDelegation not parse to big.Int, actual: %s ", vals[i].SelfDelegation)
		}
		impl.log.Debugf("mixture candidatesInfo: %s", vals[i].String())
		if !vals[i].Jailed && vals[i].Status == commonpb.BondStatus_Bonded && selfDelegation.Cmp(minSelfDelegation) >= 0 {
			candidates = append(candidates, &dpospb.CandidateInfo{
				PeerID: vals[i].ValidatorAddress,
				Weight: vals[i].Tokens,
			})
		}
	}
	return candidates, nil
}

func (impl *DPoSImpl) createEpochRwSet(epoch *commonpb.Epoch) (*commonpb.TxRWSet, error) {
	id := make([]byte, 8)
	currPreFix := []byte(native.KeyCurrentEpoch)
	binary.BigEndian.PutUint64(id, epoch.EpochID)
	bz, err := proto.Marshal(epoch)
	if err != nil {
		impl.log.Errorf("marshal epoch failed, reason: %s", err)
		return nil, err
	}

	rw := &commonpb.TxRWSet{
		TxId: "",
		TxWrites: []*commonpb.TxWrite{
			{
				ContractName: commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
				Key:          currPreFix,
				Value:        bz,
			},
			{
				ContractName: commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(),
				Key:          native.ToEpochKey(fmt.Sprintf("%d", epoch.EpochID)),
				Value:        bz,
			},
		},
	}
	return rw, nil
}

func (impl *DPoSImpl) createRewardRwSet(rewardAmount big.Int) (*commonpb.TxRWSet, error) {
	return nil, nil
}

func (impl *DPoSImpl) createSlashRwSet(slashAmount big.Int) (*commonpb.TxRWSet, error) {
	return nil, nil
}

func (impl *DPoSImpl) completeUnbonding(epoch *commonpb.Epoch, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*commonpb.TxRWSet, error) {
	prefix := native.ToUnbondingDelegationPrefix(epoch.EpochID)
	iterRange := util.BytesPrefix(prefix)
	iter, err := impl.stateDB.SelectObject(commonpb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		impl.log.Errorf("new select range failed, reason: %s", err)
		return nil, err
	}
	defer iter.Release()

	undelegations := make([]*commonpb.UnbondingDelegation, 0, 10)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			impl.log.Errorf("get kv from iterator failed, reason: %s", err)
			return nil, err
		}
		undelegation := commonpb.UnbondingDelegation{}
		if err = proto.Unmarshal(kv.Value, &undelegation); err != nil {
			impl.log.Errorf("unmarshal value to UnbondingDelegation failed, reason: %s", err)
			return nil, err
		}
		undelegations = append(undelegations, &undelegation)
	}
	rwSet := &commonpb.TxRWSet{
		TxId: ModuleName,
	}
	for _, undelegation := range undelegations {
		for _, entry := range undelegation.Entries {
			wSet, err := impl.addBalanceRwSet(undelegation.DelegatorAddress, entry.Amount, block, blockTxRwSet)
			if err != nil {
				return nil, err
			}
			rwSet.TxWrites = append(rwSet.TxWrites, wSet)

			stakeContractAddr := native.StakeContractAddr()
			if wSet, err = impl.subBalanceRwSet(stakeContractAddr, entry.Amount, block, blockTxRwSet); err != nil {
				return nil, err
			}
			rwSet.TxWrites = append(rwSet.TxWrites, wSet)
		}
	}
	return rwSet, nil
}

func (impl *DPoSImpl) addBalanceRwSet(addr string, amount string, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*commonpb.TxWrite, error) {
	before, err := impl.balanceOf(addr, block, blockTxRwSet)
	if err != nil {
		return nil, err
	}
	add, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		impl.log.Errorf("invalid amount: %s", amount)
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}
	after := before.Add(add, before)
	return &commonpb.TxWrite{
		ContractName: commonpb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		Key:          []byte(native.BalanceKey(addr)),
		Value:        []byte(after.String()),
	}, nil
}

func (impl *DPoSImpl) subBalanceRwSet(addr string, amount string, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*commonpb.TxWrite, error) {
	before, err := impl.balanceOf(addr, block, blockTxRwSet)
	if err != nil {
		return nil, err
	}
	sub, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		impl.log.Errorf("invalid amount: %s", amount)
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}
	if before.Cmp(sub) < 0 {
		impl.log.Errorf("invalid sub amount, beforeAmount: %s, subAmount: %s", before.String(), sub.String())
		return nil, fmt.Errorf("invalid sub amount, beforeAmount: %s, subAmount: %s", before.String(), sub.String())
	}
	after := before.Sub(before, sub)
	return &commonpb.TxWrite{
		ContractName: commonpb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String(),
		Key:          []byte(native.BalanceKey(addr)),
		Value:        []byte(after.String()),
	}, nil
}

func (impl *DPoSImpl) balanceOf(addr string, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*big.Int, error) {
	key := []byte(native.BalanceKey(addr))
	val, err := impl.getState(key, block, blockTxRwSet)
	if err != nil {
		return nil, err
	}
	balance := big.NewInt(0)
	if len(val) == 0 {
		return balance, nil
	}
	balance, ok := balance.SetString(string(val), 10)
	if !ok {
		return balance, fmt.Errorf("invalid amount: %s", val)
	}
	return balance, nil
}
