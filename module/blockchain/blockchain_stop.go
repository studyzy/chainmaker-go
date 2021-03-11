/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
)

// Stop all the modules.
func (bc *Blockchain) Stop() {
	// stop all module

	// stop sequence：
	// 1、tx pool
	// 2、sync service
	// 3、core engine
	// 4、consensus module
	// 5、spv node
	// 6、net service

	var stopModules []map[string]func() error
	if localconf.ChainMakerConfig.NodeConfig.Type == "spv" {
		stopModules = []map[string]func() error{
			{"NetService": bc.stopNetService},
			{"Spv": bc.stopSpv},
		}
	} else {
		if bc.getConsensusType() == consensus.ConsensusType_SOLO {
			// solo
			stopModules = []map[string]func() error{
				{"Consensus": bc.stopConsensus},
				{"Core": bc.stopCoreEngine},
				{"txPool": bc.stopTxPool},
			}

		} else {
			// not solo
			stopModules = []map[string]func() error{
				{"NetService": bc.stopNetService},
				{"Consensus": bc.stopConsensus},
				{"Core": bc.stopCoreEngine},
				{"Sync": bc.stopSyncService},
				{"txPool": bc.stopTxPool},
			}
		}
	}

	total := len(stopModules)

	// stop with total order
	for idx := total - 1; idx >= 0; idx-- {
		stopModule := stopModules[idx]
		for name, stopFunc := range stopModule {
			if err := stopFunc(); err != nil {
				bc.log.Errorf("stop module[%s] failed, %s", name, err)
				continue
			}
			bc.log.Infof("START STEP (%d/%d) => stop module[%s] success :)", total-idx, total, name)
		}
	}
}

func (bc *Blockchain) stopNetService() error {
	// stop net service
	if err := bc.netService.Stop(); err != nil {
		bc.log.Errorf("stop net service failed, %s", err.Error())
		return err
	}
	return nil
}

func (bc *Blockchain) stopSpv() error {
	// stop spv node
	bc.spv.Stop()
	return nil
}

func (bc *Blockchain) stopConsensus() error {
	// stop consensus module
	if err := bc.consensus.Stop(); err != nil {
		bc.log.Errorf("stop consensus failed, %s", err.Error())
		return err
	}
	return nil
}

func (bc *Blockchain) stopCoreEngine() error {
	// stop core engine
	bc.coreEngine.Stop()
	return nil
}

func (bc *Blockchain) stopSyncService() error {
	// stop sync
	bc.syncServer.Stop()
	return nil
}

func (bc *Blockchain) stopTxPool() error {
	// stop tx pool
	err := bc.txPool.Stop()
	if err != nil {
		bc.log.Errorf("stop tx pool failed, %s", err)

		return err
	}
	return nil
}
