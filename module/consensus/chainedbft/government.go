/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"chainmaker.org/chainmaker-go/consensus/chainedbft/message"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/types"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/government"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
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
	governmentContract protocol.Government
}

// createNextEpochIfRequired If the conditions are met, create the next epoch
func (cbi *ConsensusChainedBftImpl) createNextEpochIfRequired(height uint64) error {
	cbi.logger.Debugf("begin createNextEpochIfRequired ...")
	if cbi.governmentContract.GetEpochId() == cbi.smr.getEpochId() {
		cbi.logger.Debugf("end createNextEpochIfRequired not create next epoch")
		return nil
	}
	curEpoch := cbi.createEpoch(height)
	cbi.Lock()
	cbi.nextEpoch = curEpoch
	cbi.Unlock()
	cbi.logger.Debugf("ChainConf change! height [%d]", height)
	return nil
}

// createEpoch create the epoch in the block height
func (cbi *ConsensusChainedBftImpl) createEpoch(height uint64) *epochManager {
	var (
		validators        []*types.Validator
		members           []*consensusPb.GovernmentMember
		validatorsMembers []*consensusPb.GovernmentMember
	)
	if validators := cbi.governmentContract.GetValidators(); validators != nil {
		validatorsMembers = validators.([]*consensusPb.GovernmentMember)
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
		governmentContract: cbi.governmentContract,

		epochId:           cbi.governmentContract.GetEpochId(),
		switchHeight:      cbi.governmentContract.GetSwitchHeight(),
		skipTimeoutCommit: cbi.governmentContract.GetSkipTimeoutCommit(),
		msgPool: message.NewMsgPool(cbi.governmentContract.GetCachedLen(), int(cbi.governmentContract.
			GetGovMembersValidatorCount()), int(cbi.governmentContract.GetGovMembersValidatorMinCount())),
		cacheNextHeight: message.NewMsgPool(cbi.governmentContract.GetCachedLen(), int(cbi.governmentContract.
			GetGovMembersValidatorCount()), int(cbi.governmentContract.GetGovMembersValidatorMinCount())),
	}
	if membersInterface := cbi.governmentContract.GetMembers(); membersInterface != nil {
		members = membersInterface.([]*consensusPb.GovernmentMember)
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

func (cbi *ConsensusChainedBftImpl) TryEpochSwitch() bool {
	if cbi.nextEpoch == nil {
		return false
	}
	return cbi.nextEpoch.force
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
	validatorsInterface := cbi.governmentContract.GetValidators()
	if validatorsInterface == nil {
		return 0
	}
	validators := validatorsInterface.([]*consensusPb.GovernmentMember)
	validator, _ := government.GetProposer(level, cbi.governmentContract.GetNodeProposeRound(), validators)
	if validator != nil {
		return uint64(validator.Index)
	}
	return utils.InvalidIndex
}
