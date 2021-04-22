/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"sync"

	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
)

type GovernanceContractImp struct {
	log                *logger.CMLogger
	store              protocol.BlockchainStore
	GovernanceContract *consensusPb.GovernanceContract //Cache governance data
	Height             int64                           //Cache height
	sync.RWMutex
}

func NewGovernanceContract(store protocol.BlockchainStore) protocol.Government {
	GovernanceContract := &GovernanceContractImp{
		log:                logger.GetLogger(logger.MODULE_CONSENSUS),
		store:              store,
		GovernanceContract: nil,
		Height:             0,
	}
	return GovernanceContract
}

//Get Government data from cache,ChainStore,chainConfig
func (gcr *GovernanceContractImp) GetGovernanceContract() (*consensusPb.GovernanceContract, error) {
	//if cached height is latest,use cache
	block, err := gcr.store.GetLastBlock()
	if err != nil {
		gcr.log.Errorw("GetLastBlock err,", "err", err)
		return nil, err
	}
	if gcr.GovernanceContract != nil && block.Header.GetBlockHeight() == gcr.Height {
		return gcr.GovernanceContract, nil
	}
	var GovernanceContract *consensusPb.GovernanceContract = nil
	//get from chainStore
	if block.Header.GetBlockHeight() != 0 {
		GovernanceContract, err = getGovernanceContractFromChainStore(gcr.store)
		if err != nil {
			gcr.log.Errorw("GetLastBlock err,", "err", err)
			return nil, err
		}
	} else {
		//if genesis block,create governance from gensis config
		chainConfig, err := getChainConfigFromChainStore(gcr.store)
		if err != nil {
			gcr.log.Errorw("getChainConfigFromChainStore err,", "err", err)
			return nil, err
		}
		GovernanceContract, err = getGovernanceContractFromConfig(chainConfig)
		if err != nil {
			gcr.log.Errorw("getGovernanceContractFromConfig err,", "err", err)
			return nil, err
		}
	}
	//save as cache
	gcr.Lock()
	gcr.GovernanceContract = GovernanceContract
	gcr.Height = block.Header.GetBlockHeight()
	gcr.Unlock()
	return GovernanceContract, nil
}

//get actual consensus node num
func (gcr *GovernanceContractImp) GetGovMembersValidatorCount() uint64 {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return 0
	}
	return GovernanceContract.N
}

// actual consensus node num at least
func (gcr *GovernanceContractImp) GetGovMembersValidatorMinCount() uint64 {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return 0
	}
	return GovernanceContract.MinQuorumForQc
}

func (gcr *GovernanceContractImp) GetCachedLen() uint64 {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return 0
	}
	return GovernanceContract.CachedLen
}

//get consensus node list
func (gcr *GovernanceContractImp) GetMembers() interface{} {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range GovernanceContract.Members {
		newMember := &consensusPb.GovernanceMember{
			Index:  member.Index,
			NodeID: member.NodeID,
		}
		members = append(members, newMember)
	}

	return members
}

//get cur actual consensus node
func (gcr *GovernanceContractImp) GetValidators() interface{} {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return nil
	}
	//gcr.log.Errorw("GetValidators len,", "len", len(GovernanceContract.Validators))
	var members []*consensusPb.GovernanceMember
	for _, member := range GovernanceContract.Validators {
		newMember := &consensusPb.GovernanceMember{
			Index:  member.Index,
			NodeID: member.NodeID,
		}
		members = append(members, newMember)
	}
	return members
}

//get next epoch consensus node
func (gcr *GovernanceContractImp) GetNextValidators() interface{} {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range GovernanceContract.NextValidators {
		newMember := &consensusPb.GovernanceMember{
			Index:  member.Index,
			NodeID: member.NodeID,
		}
		members = append(members, newMember)
	}
	return members
}

//get next epoch switch heigh
func (gcr *GovernanceContractImp) GetSwitchHeight() uint64 {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return 0
	}
	return GovernanceContract.NextSwitchHeight
}

func (gcr *GovernanceContractImp) GetSkipTimeoutCommit() bool {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return false
	}
	return GovernanceContract.SkipTimeoutCommit
}

func (gcr *GovernanceContractImp) GetNodeProposeRound() uint64 {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return 0
	}
	return GovernanceContract.NodeProposeRound
}

func (gcr *GovernanceContractImp) GetEpochId() uint64 {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		//log.Errorw("getGovernanceContract err,", "err", err)
		return 0
	}
	return GovernanceContract.EpochId
}

//use by chainconf,check chain config before chain_config_contract run
func (gcr *GovernanceContractImp) Verify(consensusType consensusPb.ConsensusType, chainConfig *configPb.ChainConfig) error {
	GovernanceContract, err := gcr.GetGovernanceContract()
	if err != nil {
		gcr.log.Warnw("GetGovernanceContract err,", "err", err)
	}
	_, err = checkChainConfig(chainConfig, GovernanceContract)
	if err != nil {
		gcr.log.Warnw("checkChainConfig err,", "err", err)
	}
	return err
}
