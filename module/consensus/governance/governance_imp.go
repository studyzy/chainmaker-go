/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"sync/atomic"
	"unsafe"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/consensus"
	"chainmaker.org/chainmaker/protocol"
)

type GovernanceContractImp struct {
	log                *logger.CMLogger
	Height             uint64 //Cache height
	store              protocol.BlockchainStore
	ledger             protocol.LedgerCache
	governmentContract *consensusPb.GovernanceContract //Cache government data
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
func (gcr *GovernanceContractImp) GetGovernanceContract() (*consensusPb.GovernanceContract, error) {
	//1. if cached height is latest,use cache
	block := gcr.ledger.GetLastCommittedBlock()
	if block.Header.GetBlockHeight() == atomic.LoadUint64(&gcr.Height) {
		if addr := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&gcr.governmentContract))); addr != nil {
			return gcr.governmentContract, nil
		}
	}

	var (
		err                error
		governmentContract *consensusPb.GovernanceContract = nil
	)
	//2. get gov contract from db or chainConfig
	if block.Header.GetBlockHeight() > 0 {
		if governmentContract, err = getGovernanceContractFromChainStore(gcr.store); err != nil {
			gcr.log.Errorf("getGovernanceContractFromChainStore err: %s", err)
			return nil, err
		}
	} else {
		//if genesis block, create government from genesis config
		chainConfig, err := getChainConfigFromChainStore(gcr.store)
		if err != nil {
			gcr.log.Errorf("getChainConfigFromChainStore err: %s", err)
			return nil, err
		}
		governmentContract, err = getGovernanceContractFromConfig(chainConfig)
		if err != nil {
			gcr.log.Errorf("getGovernanceContractFromConfig err: %s", err)
			return nil, err
		}
	}

	gcr.log.Debugf("government contract configuration: %s", governmentContract.String())
	//save as cache
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&gcr.governmentContract)), unsafe.Pointer(governmentContract))
	atomic.StoreUint64(&gcr.Height, block.Header.BlockHeight)
	return governmentContract, nil
}

//CheckAndCreateGovernmentArgs execute after block propose,create government txRWSet,wait to add to block header
//when block commit,government txRWSet take effect
func CheckAndCreateGovernmentArgs(proposalCache protocol.ProposalCache,
	block *commonPb.Block, governanceContract *consensusPb.GovernanceContract) (*commonPb.TxRWSet, error) {
	log.Debugf("CheckAndCreateGovernmentArgs start")

	// 1. check if chain config change
	if !utils.IsConfBlock(block) {
		return nil, nil
	}
	var isConfigChg = false
	chainConfig, err := getChainConfigFromBlock(block, proposalCache)
	if err != nil {
		return nil, err
	}
	if chainConfig != nil {
		if isConfigChg, err = updateGovContractByConfig(chainConfig, governanceContract); err != nil {
			return nil, err
		}
	}

	// 2. if chain config change or switch to next epoch, change the GovernanceContract epochId
	if isConfigChg {
		governanceContract.EpochId++
		governanceContract.NextSwitchHeight = block.Header.BlockHeight
	}

	// 4. create TxRWSet for GovernanceContract
	txRWSet, err := getGovernanceContractTxRWSet(governanceContract)
	return txRWSet, err
}
