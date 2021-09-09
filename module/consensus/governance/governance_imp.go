/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"sync"

	"chainmaker.org/chainmaker/logger/v2"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
)

type GovernanceContractImp struct {
	log                *logger.CMLogger
	Height             uint64 //Cache height
	store              protocol.BlockchainStore
	ledger             protocol.LedgerCache
	governmentContract *consensusPb.GovernanceContract //Cache government data
	sync.RWMutex
}

func NewGovernanceContract(store protocol.BlockchainStore, ledger protocol.LedgerCache) *GovernanceContractImp {
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
		governmentContract *consensusPb.GovernanceContract
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
		gcr.log.Errorf("GetGovMembersValidatorCount error, failed reason: %s", err)
		return 0
	}
	return governmentContract.N
}

// actual consensus node num at least
func (gcr *GovernanceContractImp) GetGovMembersValidatorMinCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetGovMembersValidatorMinCount error, failed reason: %s", err)
		return 0
	}
	return governmentContract.MinQuorumForQc
}

func (gcr *GovernanceContractImp) GetLastGovMembersValidatorMinCount() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetLastGovMembersValidatorMinCount error, failed reason: %s", err)
		return 0
	}
	return governmentContract.LastMinQuorumForQc
}

func (gcr *GovernanceContractImp) GetCachedLen() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetCachedLen error, failed reason: %s", err)
		return 0
	}
	return governmentContract.CachedLen
}

//get consensus node list
func (gcr *GovernanceContractImp) GetMembers() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetMembers error, failed reason: %s", err)
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range governmentContract.Members {
		members = append(members, &consensusPb.GovernanceMember{
			Index: member.Index, NodeId: member.NodeId,
		})
	}
	return members
}

//get cur actual consensus node
func (gcr *GovernanceContractImp) GetValidators() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetValidators error, failed reason: %s", err)
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range governmentContract.Validators {
		members = append(members, &consensusPb.GovernanceMember{
			Index: member.Index, NodeId: member.NodeId,
		})
	}
	return members
}

//get next epoch consensus node
func (gcr *GovernanceContractImp) GetNextValidators() interface{} {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetNextValidators error, failed reason: %s", err)
		return nil
	}
	var members []*consensusPb.GovernanceMember
	for _, member := range governmentContract.NextValidators {
		newMember := &consensusPb.GovernanceMember{
			Index:  member.Index,
			NodeId: member.NodeId,
		}
		members = append(members, newMember)
	}
	return members
}

//get next epoch switch heigh
func (gcr *GovernanceContractImp) GetSwitchHeight() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetSwitchHeight error, failed reason: %s", err)
		return 0
	}
	return governmentContract.NextSwitchHeight
}

func (gcr *GovernanceContractImp) GetSkipTimeoutCommit() bool {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetSkipTimeoutCommit error, failed reason: %s", err)
		return false
	}
	return governmentContract.SkipTimeoutCommit
}

func (gcr *GovernanceContractImp) GetNodeProposeRound() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetNodeProposeRound error, failed reason: %s", err)
		return 0
	}
	return governmentContract.NodeProposeRound
}

func (gcr *GovernanceContractImp) GetEpochId() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetEpochId error, failed reason: %s", err)
		return 0
	}
	return governmentContract.EpochId
}

//use by chainConf, check chain config before chain_config_contract run
func (gcr *GovernanceContractImp) Verify(consensusType consensusPb.ConsensusType,
	chainConfig *configPb.ChainConfig) error {
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

func (gcr *GovernanceContractImp) GetRoundTimeoutMill() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetRoundTimeoutMill error, failed reason: %s", err)
		return 0
	}
	return governmentContract.HotstuffRoundTimeoutMill
}

func (gcr *GovernanceContractImp) GetRoundTimeoutIntervalMill() uint64 {
	governmentContract, err := gcr.GetGovernmentContract()
	if err != nil {
		gcr.log.Errorf("GetRoundTimeoutIntervalMill error, failed reason: %s", err)
		return 0
	}
	return governmentContract.HotstuffRoundTimeoutIntervalMill
}
