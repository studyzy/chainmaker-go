/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"sync"

	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/protocol/v2"
)

type TxSchedulerFactory struct {
}

// NewTxScheduler building a transaction scheduler
func (sf TxSchedulerFactory) NewTxScheduler(vmMgr protocol.VmManager, chainConf protocol.ChainConf,
	storeHelper conf.StoreHelper) protocol.TxScheduler {
	if chainConf.ChainConfig().Scheduler != nil && chainConf.ChainConfig().Scheduler.EnableEvidence {
		return newTxSchedulerEvidence(vmMgr, chainConf, storeHelper)
	}
	return newTxScheduler(vmMgr, chainConf, storeHelper)
}

// newTxScheduler building a regular transaction scheduler
func newTxScheduler(vmMgr protocol.VmManager, chainConf protocol.ChainConf, storeHelper conf.StoreHelper) *TxScheduler {
	log := logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId)
	log.Debugf("use the common TxScheduler.")
	var txScheduler = &TxScheduler{
		lock:            sync.Mutex{},
		VmManager:       vmMgr,
		scheduleFinishC: make(chan bool),
		log:             log,
		chainConf:       chainConf,
		StoreHelper:     storeHelper,
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		txScheduler.metricVMRunTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_PROPOSER_SCHEDULER, "metric_vm_run_time",
			"VM run time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return txScheduler
}

// newTxSchedulerEvidence building a evidence transaction scheduler
func newTxSchedulerEvidence(vmMgr protocol.VmManager, chainConf protocol.ChainConf,
	storeHelper conf.StoreHelper) *TxSchedulerEvidence {
	log := logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId)
	log.Debugf("use the evidence TxScheduler.")
	txSchedulerEvidence := &TxSchedulerEvidence{
		delegate: &TxScheduler{
			lock:            sync.Mutex{},
			VmManager:       vmMgr,
			scheduleFinishC: make(chan bool),
			log:             log,
			chainConf:       chainConf,
			StoreHelper:     storeHelper,
		},
	}

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		txSchedulerEvidence.delegate.metricVMRunTime = monitor.NewHistogramVec(
			monitor.SUBSYSTEM_CORE_PROPOSER_SCHEDULER,
			"metric_vm_run_time",
			"VM run time metric",
			[]float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10},
			"chainId",
		)
	}
	return txSchedulerEvidence
}
