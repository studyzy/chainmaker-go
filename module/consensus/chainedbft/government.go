/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"chainmaker.org/chainmaker-go/consensus/chainedbft/message"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/types"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/governance"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	chainUtils "chainmaker.org/chainmaker-go/utils"
)

//epochManager manages the components that shared across epoch
type epochManager struct {
	force             bool   //use epoch immediately
	index             uint64 //Local assigned index in next committee group
	epochId           uint64
	createHeight      uint64 //Next height across epoch
	switchHeight      uint64 //real switch epoch
	skipTimeoutCommit bool

	msgPool            *message.MsgPool   //The msg pool associated to next epoch
	cacheNextHeight    *message.MsgPool   //The msg pool associated to next height
	useValidators      []*types.Validator //The peer pool associated to next epoch
	governanceContract protocol.Government
}

// createNextEpochIfRequired If the conditions are met, create the next epoch
func (cbi *ConsensusChainedBftImpl) createNextEpochIfRequired(height uint64) error {
	cbi.logger.Debugf("begin createNextEpochIfRequired ...")
	if cbi.governanceContract.GetEpochId() == cbi.smr.getEpochId() {
		cbi.logger.Debugf("end createNextEpochIfRequired not create next epoch")
		return nil
	}
	curEpoch := cbi.createEpoch(height)
	cbi.mtx.Lock()
	cbi.nextEpoch = curEpoch
	cbi.mtx.Unlock()
	cbi.logger.Debugf("ChainConf change! height [%d]", height)
	return nil
}

// createEpoch create the epoch in the block height
func (cbi *ConsensusChainedBftImpl) createEpoch(height uint64) *epochManager {
	var (
		validators        []*types.Validator
		members           []*consensusPb.GovernanceMember
		validatorsMembers []*consensusPb.GovernanceMember
	)
	if validators := cbi.governanceContract.GetValidators(); validators != nil {
		validatorsMembers = validators.([]*consensusPb.GovernanceMember)
	}
	for _, v := range validatorsMembers {
		validators = append(validators, &types.Validator{
			Index: uint64(v.Index), NodeID: v.NodeID,
		})
	}

	epoch := &epochManager{
		force:              false,
		index:              utils.InvalidIndex,
		createHeight:       height,
		useValidators:      validators,
		governanceContract: cbi.governanceContract,

		epochId:           cbi.governanceContract.GetEpochId(),
		switchHeight:      cbi.governanceContract.GetSwitchHeight(),
		skipTimeoutCommit: cbi.governanceContract.GetSkipTimeoutCommit(),
		msgPool: message.NewMsgPool(cbi.governanceContract.GetCachedLen(), int(cbi.governanceContract.
			GetGovMembersValidatorCount()), int(cbi.governanceContract.GetGovMembersValidatorMinCount())),
		cacheNextHeight: message.NewMsgPool(cbi.governanceContract.GetCachedLen(), int(cbi.governanceContract.
			GetGovMembersValidatorCount()), int(cbi.governanceContract.GetGovMembersValidatorMinCount())),
	}
	if membersInterface := cbi.governanceContract.GetMembers(); membersInterface != nil {
		members = membersInterface.([]*consensusPb.GovernanceMember)
	}
	for _, v := range members {
		if v.NodeID == cbi.id {
			epoch.index = uint64(v.Index)
			break
		}
	}
	cbi.logger.Debugf("createEpoch useValidators len [%d]",
		len(epoch.useValidators))
	return epoch
}

//isValidProposer checks whether given index is valid at level
func (cbi *ConsensusChainedBftImpl) isValidProposer(level uint64, index uint64) bool {
	proposerIndex := cbi.getProposer(level)
	if proposerIndex == index {
		return true
	}
	return false
}

func (cbi *ConsensusChainedBftImpl) getProposer(level uint64) uint64 {
	beginGetContract := chainUtils.CurrentTimeMillisSeconds()
	validatorsInterface := cbi.governanceContract.GetValidators()
	if validatorsInterface == nil {
		return 0
	}
	endGetContract := chainUtils.CurrentTimeMillisSeconds()
	validators := validatorsInterface.([]*consensusPb.GovernanceMember)
	validator, _ := governance.GetProposer(level, cbi.governanceContract.GetNodeProposeRound(), validators)
	endCalValidators := chainUtils.CurrentTimeMillisSeconds()
	cbi.logger.Debugf("time cost in getProposer, getContractTime: %d, calValidatorTime: %d",
		endGetContract-beginGetContract, endCalValidators-endGetContract)
	if validator != nil {
		return uint64(validator.Index)
	}
	return utils.InvalidIndex
}
