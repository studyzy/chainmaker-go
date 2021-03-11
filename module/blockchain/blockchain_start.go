/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
)

// Start all the modules.
func (bc *Blockchain) Start() error {
	// start all module

	// start sequence：
	// 1、net service
	// 2、spv node
	// 3、core engine
	// 4、consensus module
	// 5、sync service
	// 6、tx pool

	var startModules []map[string]func() error
	if bc.getConsensusType() == consensus.ConsensusType_SOLO {
		// solo
		startModules = []map[string]func() error{
			{"Core": bc.startCoreEngine},
			{"Consensus": bc.startConsensus},
			{"txPool": bc.startTxPool},
		}
	} else {
		// not solo
		startModules = []map[string]func() error{
			{"NetService": bc.startNetService},
			{"Core": bc.startCoreEngine},
			{"Consensus": bc.startConsensus},
			{"txPool": bc.startTxPool},
			{"Sync": bc.startSyncService},
		}
	}

	total := len(startModules)

	for idx, startModule := range startModules {
		for name, startFunc := range startModule {
			if err := startFunc(); err != nil {
				bc.log.Errorf("start module[%s] failed, %s", name, err)
				return err
			}
			bc.log.Infof("START STEP (%d/%d) => start module[%s] success :)", idx+1, total, name)
		}
	}

	return nil
}

func (bc *Blockchain) startNetService() error {
	// start net service
	if err := bc.netService.Start(); err != nil {
		bc.log.Errorf("start net service failed, %s", err.Error())
		return err
	}
	return nil
}

func (bc *Blockchain) startConsensus() error {
	// start consensus module
	if err := bc.consensus.Start(); err != nil {
		bc.log.Errorf("start consensus failed, %s", err.Error())
		return err
	}
	return nil
}

func (bc *Blockchain) startCoreEngine() error {
	// start core engine
	bc.coreEngine.Start()
	return nil
}

func (bc *Blockchain) startSyncService() error {
	// start sync
	if err := bc.syncServer.Start(); err != nil {
		bc.log.Errorf("start sync server failed, %s", err.Error())
		return err
	}
	return nil
}

func (bc *Blockchain) startTxPool() error {
	// start tx pool
	err := bc.txPool.Start()
	if err != nil {
		bc.log.Errorf("start tx pool failed, %s", err)

		return err
	}
	return nil
}
