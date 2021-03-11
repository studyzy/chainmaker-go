/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"sync"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
)

type GovernmentContractImp struct {
	log                *logger.CMLogger
	store              protocol.BlockchainStore
	governmentContract *consensusPb.GovernmentContract //Cache governance data
	Height             int64                           //Cache height
	sync.RWMutex
}

func NewGovernmentContract(store protocol.BlockchainStore) protocol.Government {
	governmentContract := &GovernmentContractImp{
		log:                logger.GetLogger(logger.MODULE_CONSENSUS),
		store:              store,
		governmentContract: nil,
		Height:             0,
	}
	return governmentContract
}

//Get Government data from cache,ChainStore,chainConfig
func (gcr *GovernmentContractImp) GetGovernmentContract() (*consensusPb.GovernmentContract, error) {
	//if cached height is latest,use cache
	block, err := gcr.store.GetLastBlock()
	if err != nil {
		gcr.log.Errorw("GetLastBlock err,", "err", err)
		return nil, err
	}
	if gcr.governmentContract != nil && block.Header.GetBlockHeight() == gcr.Height {
		return gcr.governmentContract, nil
	}
	var governmentContract *consensusPb.GovernmentContract = nil
	//get from chainStore
	if block.Header.GetBlockHeight() != 0 {
		governmentContract, err = getGovernmentContractFromChainStore(gcr.store)
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
		governmentContract, err = getGovernmentContractFromConfig(chainConfig)
		if err != nil {
			gcr.log.Errorw("getGovernmentContractFromConfig err,", "err", err)
			return nil, err
		}
	}
	//save as cache
	gcr.Lock()
	gcr.governmentContract = governmentContract
	gcr.Height = block.Header.GetBlockHeight()
	gcr.Unlock()
	return governmentContract, nil
}

//get actual consensus node num
func (gcr *GovernmentContractImp) GetGovMembersValidatorCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.N
}

// actual consensus node num at least
func (gcr *GovernmentContractImp) GetGovMembersValidatorMinCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.MinQuorumForQc
}

func (gcr *GovernmentContractImp) GetCachedLen() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.CachedLen
}

//get consensus node list
func (gcr *GovernmentContractImp) GetMembers() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return nil
	}
	var members []*consensusPb.GovernmentMember
	for _, member := range governmentContract.Members {
		newMember := &consensusPb.GovernmentMember{
			Index:  member.Index,
			NodeID: member.NodeID,
		}
		members = append(members, newMember)
	}

	return members
}

//get cur actual consensus node
func (gcr *GovernmentContractImp) GetValidators() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return nil
	}
	//gcr.log.Errorw("GetValidators len,", "len", len(governmentContract.Validators))
	var members []*consensusPb.GovernmentMember
	for _, member := range governmentContract.Validators {
		newMember := &consensusPb.GovernmentMember{
			Index:  member.Index,
			NodeID: member.NodeID,
		}
		members = append(members, newMember)
	}
	return members
}

//get next epoch consensus node
func (gcr *GovernmentContractImp) GetNextValidators() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return nil
	}
	var members []*consensusPb.GovernmentMember
	for _, member := range governmentContract.NextValidators {
		newMember := &consensusPb.GovernmentMember{
			Index:  member.Index,
			NodeID: member.NodeID,
		}
		members = append(members, newMember)
	}
	return members
}

//get next epoch switch heigh
func (gcr *GovernmentContractImp) GetSwitchHeight() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.NextSwitchHeight
}

func (gcr *GovernmentContractImp) GetSkipTimeoutCommit() bool {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return false
	}
	return governmentContract.SkipTimeoutCommit
}

func (gcr *GovernmentContractImp) GetNodeProposeRound() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.NodeProposeRound
}

func (gcr *GovernmentContractImp) GetEpochId() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.EpochId
}

//use by chainconf,check chain config before chain_config_contract run
func (gcr *GovernmentContractImp) Verify(consensusType consensusPb.ConsensusType, chainConfig *configPb.ChainConfig) error {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Warnw("GetGovernmentContract err,", "err", err)
	}
	_, err = checkChainConfig(chainConfig, governmentContract)
	if err != nil {
		gcr.log.Warnw("checkChainConfig err,", "err", err)
	}
	return err
}
