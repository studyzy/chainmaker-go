/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockchain

// Start all the modules.
func (bc *Blockchain) Start() error {
	// start all module

	// start sequence：
	// 1、net service
	// 2、spv node
	// 3、core engine
	// 4、consensus module
	// 5、tx pool
	// 6、sync service

	var startModules = make([]map[string]func() error, 0)
	if bc.isModuleInit(moduleNameNetService) {
		startModules = append(startModules, map[string]func() error{moduleNameNetService: bc.startNetService})
	}
	//if bc.isModuleInit(moduleNameSpv) {
	//	startModules = append(startModules, map[string]func() error{moduleNameSpv: bc.startSpv})
	//}
	if bc.isModuleInit(moduleNameCore) {
		startModules = append(startModules, map[string]func() error{moduleNameCore: bc.startCoreEngine})
	}
	if bc.isModuleInit(moduleNameConsensus) {
		startModules = append(startModules, map[string]func() error{moduleNameConsensus: bc.startConsensus})
	}
	if bc.isModuleInit(moduleNameTxPool) {
		startModules = append(startModules, map[string]func() error{moduleNameTxPool: bc.startTxPool})
	}
	if bc.isModuleInit(moduleNameSync) {
		startModules = append(startModules, map[string]func() error{moduleNameSync: bc.startSyncService})
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
	bc.startModules[moduleNameNetService] = struct{}{}
	return nil
}

func (bc *Blockchain) startConsensus() error {
	// start consensus module
	if bc.consensus == nil {
		return nil
	}
	if err := bc.consensus.Start(); err != nil {
		bc.log.Errorf("start consensus failed, %s", err.Error())
		return err
	}
	bc.startModules[moduleNameConsensus] = struct{}{}
	return nil
}

func (bc *Blockchain) startCoreEngine() error {
	// start core engine
	bc.coreEngine.Start()
	bc.startModules[moduleNameCore] = struct{}{}
	return nil
}

func (bc *Blockchain) startSyncService() error {
	// start sync
	if err := bc.syncServer.Start(); err != nil {
		bc.log.Errorf("start sync server failed, %s", err.Error())
		return err
	}
	bc.startModules[moduleNameSync] = struct{}{}
	return nil
}

func (bc *Blockchain) startTxPool() error {
	// start tx pool
	err := bc.txPool.Start()
	if err != nil {
		bc.log.Errorf("start tx pool failed, %s", err)
		return err
	}
	bc.startModules[moduleNameTxPool] = struct{}{}
	return nil
}

func (bc *Blockchain) isModuleStartUp(moduleName string) bool {
	_, res := bc.startModules[moduleName]
	return res
}
