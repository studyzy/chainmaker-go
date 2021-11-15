/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

// Stop all the modules.
func (bc *Blockchain) Stop() {
	// stop all module

	// stop sequence：
	// 1、sync service
	// 2、core engine
	// 3、consensus module
	// 4、net service
	// 5、tx pool
	// 6、vm

	var stopModules = make([]map[string]func() error, 0)
	if bc.isModuleStartUp(moduleNameNetService) {
		stopModules = append(stopModules, map[string]func() error{moduleNameNetService: bc.stopNetService})
	}
	if bc.isModuleStartUp(moduleNameConsensus) {
		stopModules = append(stopModules, map[string]func() error{moduleNameConsensus: bc.stopConsensus})
	}
	if bc.isModuleStartUp(moduleNameCore) {
		stopModules = append(stopModules, map[string]func() error{moduleNameCore: bc.stopCoreEngine})
	}
	if bc.isModuleStartUp(moduleNameSync) {
		stopModules = append(stopModules, map[string]func() error{moduleNameSync: bc.stopSyncService})
	}
	if bc.isModuleStartUp(moduleNameTxPool) {
		stopModules = append(stopModules, map[string]func() error{moduleNameTxPool: bc.stopTxPool})
	}
	if bc.isModuleStartUp(moduleNameVM) {
		stopModules = append(stopModules, map[string]func() error{moduleNameVM: bc.stopVM})
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
			bc.log.Infof("STOP STEP (%d/%d) => stop module[%s] success :)", total-idx, total, name)
		}
	}
}

// StopOnRequirements close the module instance which is required to shut down when chain configuration updating.
func (bc *Blockchain) StopOnRequirements() {
	stopMethodMap := map[string]func() error{
		moduleNameNetService: bc.stopNetService,
		moduleNameSync:       bc.stopSyncService,
		moduleNameCore:       bc.stopCoreEngine,
		moduleNameConsensus:  bc.stopConsensus,
		moduleNameTxPool:     bc.stopTxPool,
		moduleNameVM:         bc.stopVM,
	}

	// stop sequence：
	// 1、sync service
	// 2、core engine
	// 3、consensus module
	// 4、net service
	// 5、tx pool

	sequence := map[string]int{
		moduleNameSync:       0,
		moduleNameCore:       1,
		moduleNameConsensus:  2,
		moduleNameNetService: 3,
		moduleNameTxPool:     4,
		moduleNameVM:         5,
	}
	closeFlagArray := [6]string{}
	for moduleName := range bc.startModules {
		_, ok := bc.initModules[moduleName]
		if ok {
			continue
		}
		seq, canStop := sequence[moduleName]
		if canStop {
			closeFlagArray[seq] = moduleName
		}
	}
	// stop modules
	for i := range closeFlagArray {
		moduleName := closeFlagArray[i]
		if moduleName == "" {
			continue
		}
		stopFunc := stopMethodMap[moduleName]
		err := stopFunc()
		if err != nil {
			bc.log.Errorf("stop module[%s] failed, %s", moduleName, err)
			continue
		}
		bc.log.Infof("stop module[%s] success :)", moduleName)
	}
}

func (bc *Blockchain) stopNetService() error {
	// stop net service
	if err := bc.netService.Stop(); err != nil {
		bc.log.Errorf("stop net service failed, %s", err.Error())
		return err
	}
	delete(bc.startModules, moduleNameNetService)
	return nil
}

func (bc *Blockchain) stopConsensus() error {
	// stop consensus module
	if err := bc.consensus.Stop(); err != nil {
		bc.log.Errorf("stop consensus failed, %s", err.Error())
		return err
	}
	delete(bc.startModules, moduleNameConsensus)
	return nil
}

func (bc *Blockchain) stopCoreEngine() error {
	// stop core engine
	bc.coreEngine.Stop()
	delete(bc.startModules, moduleNameCore)
	return nil
}

func (bc *Blockchain) stopSyncService() error {
	// stop sync
	bc.syncServer.Stop()
	delete(bc.startModules, moduleNameSync)
	return nil
}

func (bc *Blockchain) stopTxPool() error {
	// stop tx pool
	err := bc.txPool.Stop()
	if err != nil {
		bc.log.Errorf("stop tx pool failed, %s", err)

		return err
	}
	delete(bc.startModules, moduleNameTxPool)
	return nil
}

func (bc *Blockchain) stopVM() error {
	// stop vm
	err := bc.vmMgr.Stop()
	if err != nil {
		bc.log.Errorf("stop vm failed, %s", err)
		return err
	}
	delete(bc.startModules, moduleNameVM)
	return nil
}
