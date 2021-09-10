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
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
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
	useValidators      []*types.Validator //The peer pool associated to next epoch
	governanceContract *governance.GovernanceContractImp
}

// createNextEpochIfRequired If the conditions are met, create the next epoch
func (cbi *ConsensusChainedBftImpl) createNextEpochIfRequired(height uint64) {
	cbi.logger.Debugf("begin createNextEpochIfRequired, "+
		"contractEpoch:%d, nodeEpoch:%d", cbi.governanceContract.GetEpochId(), cbi.smr.getEpochId())

	if cbi.governanceContract.GetEpochId() == cbi.smr.getEpochId() {
		return
	}
	cbi.createEpoch(height)
	cbi.logger.Debugf("end createNextEpochIfRequired")
}

// createEpoch create the epoch in the block height
func (cbi *ConsensusChainedBftImpl) createEpoch(height uint64) {
	var (
		validators        []*types.Validator
		members           []*consensusPb.GovernanceMember
		validatorsMembers []*consensusPb.GovernanceMember
		ok                bool
	)
	if validators := cbi.governanceContract.GetValidators(); validators != nil {
		if validatorsMembers, ok = validators.([]*consensusPb.GovernanceMember); !ok {
			cbi.logger.Errorf("create epoch failed:validator invalid")
			return
		}
	}
	for _, v := range validatorsMembers {
		validators = append(validators, &types.Validator{
			Index: uint64(v.Index), NodeID: v.NodeId,
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
	}
	if membersInterface := cbi.governanceContract.GetMembers(); membersInterface != nil {
		if members, ok = membersInterface.([]*consensusPb.GovernanceMember); !ok {
			cbi.logger.Errorf("create epoch failed: governace member invalid")
			return
		}
	}
	for _, v := range members {
		if v.NodeId == cbi.id {
			epoch.index = uint64(v.Index)
			break
		}
	}
	cbi.nextEpoch = epoch
	cbi.logger.Debugf("createEpoch useValidators len [%d]", len(epoch.useValidators))
}

//isValidProposer checks whether given index is valid at level
func (cbi *ConsensusChainedBftImpl) isValidProposer(level uint64, index uint64) bool {
	proposerIndex := cbi.getProposer(level)
	return proposerIndex == index
}

func (cbi *ConsensusChainedBftImpl) getProposer(level uint64) uint64 {
	validatorsInterface := cbi.governanceContract.GetValidators()
	if validatorsInterface == nil {
		return 0
	}
	if validators, ok := validatorsInterface.([]*consensusPb.GovernanceMember); ok {
		validator, _ := governance.GetProposer(level, cbi.governanceContract.GetNodeProposeRound(), validators)
		if validator != nil {
			return uint64(validator.Index)
		}
	}
	return utils.InvalidIndex
}
