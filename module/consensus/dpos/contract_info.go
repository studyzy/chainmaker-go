/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	dpospb "chainmaker.org/chainmaker/pb-go/v2/consensus/dpos"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/vm-native/v2/dposmgr"
	"github.com/gogo/protobuf/proto"
	goproto "github.com/gogo/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const DposOrgId = "dpos_org_id"
const ModuleName = "dpos_module"

// getEpochInfo get epoch info from ledger
func (impl *DPoSImpl) getEpochInfo() (*syscontract.Epoch, error) {
	epoch, err := GetLatestEpochInfo(impl.stateDB)
	if err != nil {
		impl.log.Errorf("get epoch failed, reason: %s", err)
		return nil, err
	}
	impl.log.Debugf("epoch info: %s", epoch.String())
	return epoch, nil
}

func (impl *DPoSImpl) getNodeIDsFromValidators(epoch *syscontract.Epoch) ([]string, error) {
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
	prefix := dposmgr.ToValidatorPrefix()
	iterRange := util.BytesPrefix(prefix)
	iter, err := impl.stateDB.SelectObject(
		syscontract.SystemContract_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		impl.log.Errorf("read contract: %s error: %s", syscontract.SystemContract_DPOS_STAKE.String(), err)
		return nil, err
	}
	defer iter.Release()
	minSelfDelegationBz, err := impl.stateDB.ReadObject(
		syscontract.SystemContract_DPOS_STAKE.String(), []byte(dposmgr.KeyMinSelfDelegation))
	if err != nil {
		impl.log.Errorf(
			"get selfMinDelegation from contract %s failed, reason: %s",
			syscontract.SystemContract_DPOS_STAKE.String(), err)
		return nil, err
	}
	minSelfDelegation, ok := big.NewInt(0).SetString(string(minSelfDelegationBz), 10)
	if !ok {
		err := fmt.Errorf("invalid minSelfDelegation in stake contract")
		impl.log.Errorf("%s", err)
		return nil, err
	}
	impl.log.Debugf("minSelfDelegation: %s", minSelfDelegation.String())
	vals := make([]*syscontract.Validator, 0, 10)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			impl.log.Errorf("iterator read error: %s", err)
			return nil, err
		}
		val := syscontract.Validator{}
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
		if !vals[i].Jailed &&
			vals[i].Status == syscontract.BondStatus_BONDED &&
			selfDelegation.Cmp(minSelfDelegation) >= 0 {
			candidates = append(candidates, &dpospb.CandidateInfo{
				PeerId: vals[i].ValidatorAddress,
				Weight: vals[i].Tokens,
			})
		}
	}
	return candidates, nil
}

func (impl *DPoSImpl) createEpochRwSet(epoch *syscontract.Epoch) (*commonpb.TxRWSet, error) {
	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, epoch.EpochId)
	bz, err := proto.Marshal(epoch)
	if err != nil {
		impl.log.Errorf("marshal epoch failed, reason: %s", err)
		return nil, err
	}

	rw := &commonpb.TxRWSet{
		TxId: "",
		TxWrites: []*commonpb.TxWrite{
			{
				ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
				Key:          []byte(dposmgr.KeyCurrentEpoch),
				Value:        bz,
			},
			{
				ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
				Key:          dposmgr.ToEpochKey(fmt.Sprintf("%d", epoch.EpochId)),
				Value:        bz,
			},
		},
	}
	return rw, nil
}

func (impl *DPoSImpl) createValidatorsRwSet(epoch *syscontract.Epoch) (*commonpb.TxRWSet, error) {

	nodeIDs, err := impl.getNodeIDsFromValidators(epoch)
	if err != nil || len(nodeIDs) == 0 {
		impl.log.Errorf("create validators rwSet failed, reason: %s", err)
		return nil, err
	}

	chainConfig, err := GetChainConfig(impl.stateDB)
	if err != nil {
		impl.log.Errorf("create validators rwSet failed, reason: %s", err)
		return nil, err
	}
	orgConfig := &configPb.OrgConfig{
		OrgId:  DposOrgId,
		NodeId: nodeIDs,
	}
	chainConfig.Consensus.Nodes[0] = orgConfig

	//chainConfig.Sequence = chainConfig.Sequence + 1
	pbccPayload, err := goproto.Marshal(chainConfig)
	if err != nil {
		err = fmt.Errorf("proto marshal pbcc failed, err: %s", err.Error())
		impl.log.Errorf("create validators rwSet failed, reason: %s", err)
		return nil, err
	}
	rw := &commonpb.TxRWSet{
		TxId: "",
		TxWrites: []*commonpb.TxWrite{
			{
				ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
				Key:          []byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
				Value:        pbccPayload,
			},
		},
	}
	return rw, nil
}

//func (impl *DPoSImpl) createRewardRwSet(rewardAmount big.Int) (*commonpb.TxRWSet, error) {
//	return nil, nil
//}

//func (impl *DPoSImpl) createSlashRwSet(slashAmount big.Int) (*commonpb.TxRWSet, error) {
//	return nil, nil
//}

func (impl *DPoSImpl) completeUnbounding(epoch *syscontract.Epoch,
	block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*commonpb.TxRWSet, error) {
	undelegations, err := impl.getUnboundingEntries(epoch)
	if err != nil {
		return nil, err
	}

	rwSet, err := impl.createUnboundingRwSet(undelegations, block, blockTxRwSet)
	return rwSet, err
}

func (impl *DPoSImpl) getUnboundingEntries(epoch *syscontract.Epoch) ([]*syscontract.UnbondingDelegation, error) {
	prefix := dposmgr.ToUnbondingDelegationPrefix(epoch.EpochId)
	iterRange := util.BytesPrefix(prefix)
	iter, err := impl.stateDB.SelectObject(
		syscontract.SystemContract_DPOS_STAKE.String(), iterRange.Start, iterRange.Limit)
	if err != nil {
		impl.log.Errorf("new select range failed, reason: %s", err)
		return nil, err
	}
	defer iter.Release()

	undelegations := make([]*syscontract.UnbondingDelegation, 0, 10)
	for iter.Next() {
		kv, err := iter.Value()
		if err != nil {
			impl.log.Errorf("get kv from iterator failed, reason: %s", err)
			return nil, err
		}
		undelegation := syscontract.UnbondingDelegation{}
		if err = proto.Unmarshal(kv.Value, &undelegation); err != nil {
			impl.log.Errorf("unmarshal value to UnbondingDelegation failed, reason: %s", err)
			return nil, err
		}
		undelegations = append(undelegations, &undelegation)
	}
	if len(undelegations) > 0 {
		impl.log.Debugf("get unDelegations: %v", undelegations)
	}
	return undelegations, nil
}

func (impl *DPoSImpl) createUnboundingRwSet(undelegations []*syscontract.UnbondingDelegation,
	block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*commonpb.TxRWSet, error) {

	rwSet := &commonpb.TxRWSet{
		TxId: ModuleName,
	}
	var (
		err               error
		balances          = make(map[string]*big.Int, len(undelegations))
		stakeContractAddr = dposmgr.StakeContractAddr()
	)
	for _, undelegation := range undelegations {
		for _, entry := range undelegation.Entries {
			balance, ok := balances[undelegation.DelegatorAddress]
			if !ok {
				balance, err = impl.balanceOf(undelegation.DelegatorAddress, block, blockTxRwSet)
				if err != nil {
					return nil, err
				}
			}
			wSet, afterBalance, err := impl.addBalanceRwSet(undelegation.DelegatorAddress, balance, entry.Amount)
			if err != nil {
				return nil, err
			}
			rwSet.TxWrites = append(rwSet.TxWrites, wSet)
			balances[undelegation.DelegatorAddress] = afterBalance

			if balance, ok = balances[stakeContractAddr]; !ok {
				if balance, err = impl.balanceOf(stakeContractAddr, block, blockTxRwSet); err != nil {
					return nil, err
				}
			}
			if wSet, afterBalance, err = impl.subBalanceRwSet(
				stakeContractAddr, balance, entry.Amount); err != nil {
				return nil, err
			}
			rwSet.TxWrites = append(rwSet.TxWrites, wSet)
			balances[stakeContractAddr] = afterBalance
		}
	}
	if len(rwSet.TxWrites) > 0 {
		impl.log.Debugf("unbounding rwSet: %s", rwSet.String())
	}
	return rwSet, nil
}

func (impl *DPoSImpl) addBalanceRwSet(
	addr string, balance *big.Int, addAmount string) (*commonpb.TxWrite, *big.Int, error) {
	add, ok := big.NewInt(0).SetString(addAmount, 10)
	if !ok {
		impl.log.Errorf("invalid amount: %s", addAmount)
		return nil, nil, fmt.Errorf("invalid amount: %s", addAmount)
	}
	after := balance.Add(add, balance)
	return &commonpb.TxWrite{
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		Key:          []byte(dposmgr.BalanceKey(addr)),
		Value:        []byte(after.String()),
	}, after, nil
}

func (impl *DPoSImpl) subBalanceRwSet(
	addr string, before *big.Int, amount string) (*commonpb.TxWrite, *big.Int, error) {
	sub, ok := big.NewInt(0).SetString(amount, 10)
	if !ok {
		impl.log.Errorf("invalid amount: %s", amount)
		return nil, nil, fmt.Errorf("invalid amount: %s", amount)
	}
	if before.Cmp(sub) < 0 {
		impl.log.Errorf("invalid sub amount, beforeAmount: %s, subAmount: %s", before.String(), sub.String())
		return nil, nil, fmt.Errorf("invalid sub amount, beforeAmount: %s, subAmount: %s", before.String(), sub.String())
	}
	after := before.Sub(before, sub)
	return &commonpb.TxWrite{
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		Key:          []byte(dposmgr.BalanceKey(addr)),
		Value:        []byte(after.String()),
	}, after, nil
}

func (impl *DPoSImpl) balanceOf(
	addr string, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) (*big.Int, error) {
	key := []byte(dposmgr.BalanceKey(addr))
	val, err := impl.getState(syscontract.SystemContract_DPOS_ERC20.String(), key, block, blockTxRwSet)
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
