/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"chainmaker.org/chainmaker-go/consensus/chainedbft/message"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker/protocol"
)

//epochManager manages the components that shared across epoch
type epochManager struct {
	index        uint64 //Local assigned index in next committee group
	epochId      uint64
	createHeight uint64 //Next height across epoch
	switchHeight uint64 //real switch epoch

	msgPool            *message.MsgPool //The msg pool associated to next epoch
	governanceContract protocol.Government
}

// createNextEpochIfRequired If the conditions are met, create the next epoch
func (cbi *ConsensusChainedBftImpl) createNextEpochIfRequired(height uint64) (*epochManager, error) {
	governContract, err := cbi.government.GetGovernanceContract()
	if err != nil {
		return nil, err
	}

	cbi.logger.Debugf("begin createNextEpochIfRequired, "+
		"contractEpoch:%d, nodeEpoch:%d", governContract.EpochId, cbi.smr.getEpochId())
	if governContract.EpochId == cbi.smr.getEpochId() {
		return nil, nil
	}
	epoch, err := cbi.createEpoch(height)
	cbi.logger.Debugf("end createNextEpochIfRequired")
	return epoch, err
}

// createEpoch create the epoch in the block height
func (cbi *ConsensusChainedBftImpl) createEpoch(height uint64) (*epochManager, error) {
	govContract, err := cbi.government.GetGovernanceContract()
	if err != nil {
		return nil, err
	}

	epoch := &epochManager{
		index:        utils.InvalidIndex,
		createHeight: height,

		epochId:            govContract.EpochId,
		switchHeight:       govContract.NextSwitchHeight,
		governanceContract: cbi.government,
		msgPool: message.NewMsgPool(govContract.GetCachedLen(),
			int(govContract.GetValidatorNum()), int(govContract.MinQuorumForQc)),
	}
	for _, v := range govContract.Validators {
		if v.NodeId == cbi.id {
			epoch.index = uint64(v.Index)
			break
		}
	}
	return epoch, nil
}

//isValidProposer checks whether given index is valid at level
func (cbi *ConsensusChainedBftImpl) isValidProposer(height, level uint64, index uint64) bool {
	proposerIndex := cbi.getProposer(height, level)
	if proposerIndex == index {
		return true
	}
	return false
}

func (cbi *ConsensusChainedBftImpl) getProposer(height, level uint64) uint64 {
	contractInfo := cbi.smr.info
	validators := cbi.smr.getPeers(height)
	if len(validators) == 0 {
		return utils.InvalidIndex
	}
	index := (level / contractInfo.GetNodeProposeRound()) % uint64(len(validators))
	return uint64(validators[index].index)
}
