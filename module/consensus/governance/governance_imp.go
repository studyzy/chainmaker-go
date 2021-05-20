/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"sync"

	"chainmaker.org/chainmaker-go/logger"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
)

type GovernanceContractImp struct {
	log                *logger.CMLogger
	Height             int64 //Cache height
	store              protocol.BlockchainStore
	ledger             protocol.LedgerCache
	governmentContract *consensusPb.GovernanceContract //Cache government data
	sync.RWMutex
}

func NewGovernanceContract(store protocol.BlockchainStore, ledger protocol.LedgerCache) protocol.Government {
	governmentContract := &GovernanceContractImp{
		log:                logger.GetLogger(logger.MODULE_CONSENSUS),
		Height:             0,
		store:              store,
		ledger:             ledger,
		governmentContract: nil,
	}
	return governmentContract
}

//Get Government data from cache,ChainStore,chainConfig
func (gcr *GovernanceContractImp) GetGovernmentContract() (*consensusPb.GovernanceContract, error) {
	//if cached height is latest,use cache
	block := gcr.ledger.GetLastCommittedBlock()
	if gcr.governmentContract != nil && block.Header.GetBlockHeight() == gcr.Height {
		return gcr.governmentContract, nil
	}
	var (
		err                error
		governmentContract *consensusPb.GovernanceContract = nil
	)
	//get from chainStore
	if block.Header.GetBlockHeight() > 0 {
		if governmentContract, err = getGovernanceContractFromChainStore(gcr.store); err != nil {
			gcr.log.Errorw("getGovernanceContractFromChainStore err,", "err", err)
			return nil, err
		}
	} else {
		//if genesis block,create government from genesis config
		chainConfig, err := getChainConfigFromChainStore(gcr.store)
		if err != nil {
			gcr.log.Errorw("getChainConfigFromChainStore err,", "err", err)
			return nil, err
		}
		governmentContract, err = getGovernanceContractFromConfig(chainConfig)
		if err != nil {
			gcr.log.Errorw("getGovernanceContractFromConfig err,", "err", err)
			return nil, err
		}
	}
	log.Debugf("government contract configuration: %v", governmentContract.String())
	//save as cache
	gcr.Lock()
	gcr.governmentContract = governmentContract
	gcr.Height = block.Header.GetBlockHeight()
	gcr.Unlock()
	return governmentContract, nil
}

//get actual consensus node num
func (gcr *GovernanceContractImp) GetGovMembersValidatorCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.N
}

// actual consensus node num at least
func (gcr *GovernanceContractImp) GetGovMembersValidatorMinCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.MinQuorumForQc
}

func (gcr *GovernanceContractImp) GetLastGovMembersValidatorMinCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.LastMinQuorumForQc
}

func (gcr *GovernanceContractImp) GetCachedLen() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.CachedLen
}

//get consensus node list
func (gcr *GovernanceContractImp) GetMembers() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range governmentContract.Members {
		members = append(members, &consensusPb.GovernanceMember{
			Index: member.Index, NodeID: member.NodeID,
		})
	}
	return members
}

//get cur actual consensus node
func (gcr *GovernanceContractImp) GetValidators() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range governmentContract.Validators {
		members = append(members, &consensusPb.GovernanceMember{
			Index: member.Index, NodeID: member.NodeID,
		})
	}
	return members
}

//get next epoch consensus node
func (gcr *GovernanceContractImp) GetNextValidators() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range governmentContract.NextValidators {
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
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.NextSwitchHeight
}

func (gcr *GovernanceContractImp) GetSkipTimeoutCommit() bool {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return false
	}
	return governmentContract.SkipTimeoutCommit
}

func (gcr *GovernanceContractImp) GetNodeProposeRound() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		//log.Errorw("getGovernmentContract err,", "err", err)
		return 0
	}
	return governmentContract.NodeProposeRound
}

func (gcr *GovernanceContractImp) GetEpochId() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		return 0
	}
	return governmentContract.EpochId
}

//use by chainConf, check chain config before chain_config_contract run
func (gcr *GovernanceContractImp) Verify(consensusType consensusPb.ConsensusType, chainConfig *configPb.ChainConfig) error {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Warnw("GetGovernmentContract err,", "err", err)
		return err
	}
	if _, err = checkChainConfig(chainConfig, governmentContract); err != nil {
		gcr.log.Warnw("checkChainConfig err,", "err", err)
	}
	return err
}
