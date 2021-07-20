/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package governance

import (
	"sync/atomic"
	"unsafe"

	"chainmaker.org/chainmaker-go/logger"
	configPb "chainmaker.org/chainmaker/pb-go/config"
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
func (gcr *GovernanceContractImp) GetGovernanceContract() (*consensusPb.GovernanceContract, error) {
	//if cached height is latest,use cache
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
	//get from chainStore
	if block.Header.GetBlockHeight() > 0 {
		if governmentContract, err = getGovernanceContractFromChainStore(gcr.store); err != nil {
			gcr.log.Errorf("getGovernanceContractFromChainStore err: %s", err)
			return nil, err
		}
	} else {
		//if genesis block,create government from genesis config
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

//use by chainConf, check chain config before chain_config_contract run
func (gcr *GovernanceContractImp) Verify(consensusType consensusPb.ConsensusType, chainConfig *configPb.ChainConfig) error {
	governmentContract, err := gcr.GetGovernanceContract()
	if err != nil {
		gcr.log.Warnw("GetGovernmentContract err,", "err", err)
		return err
	}
	if _, err = checkChainConfig(chainConfig, governmentContract); err != nil {
		gcr.log.Warnw("checkChainConfig err,", "err", err)
	}
	return err
}
