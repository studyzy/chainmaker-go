/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/monitor"
	"chainmaker.org/chainmaker-go/protocol"
)

type TxSchedulerFactory struct {
}

// NewTxScheduler building a transaction scheduler
func (sf TxSchedulerFactory) NewTxScheduler(vmMgr protocol.VmManager, chainConf protocol.ChainConf) protocol.TxScheduler {
	if chainConf.ChainConfig().Scheduler != nil && chainConf.ChainConfig().Scheduler.EnableEvidence {
		return newTxSchedulerEvidence(vmMgr, chainConf)
	} else {
		return newTxScheduler(vmMgr, chainConf)
	}
}

// newTxScheduler building a regular transaction scheduler
func newTxScheduler(vmMgr protocol.VmManager, chainConf protocol.ChainConf) *TxScheduler {
	log := logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId)
	log.Debugf("use the common TxScheduler.")
	var TxScheduler = &TxScheduler{
		lock:            sync.Mutex{},
		VmManager:       vmMgr,
		scheduleFinishC: make(chan bool),
		log:             log,
		chainConf:       chainConf,
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		TxScheduler.metricVMRunTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_PROPOSER_SCHEDULER, "metric_vm_run_time",
			"VM run time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return TxScheduler
}

// newTxSchedulerEvidence building a evidence transaction scheduler
func newTxSchedulerEvidence(vmMgr protocol.VmManager, chainConf protocol.ChainConf) *TxSchedulerEvidence {
	log := logger.GetLoggerByChain(logger.MODULE_CORE, chainConf.ChainConfig().ChainId)
	log.Debugf("use the evidence TxScheduler.")
	TxSchedulerEvidence := &TxSchedulerEvidence{
		delegate: &TxScheduler{
			lock:            sync.Mutex{},
			VmManager:       vmMgr,
			scheduleFinishC: make(chan bool),
			log:             log,
			chainConf:       chainConf,
		},
	}

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		TxSchedulerEvidence.delegate.metricVMRunTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_PROPOSER_SCHEDULER, "metric_vm_run_time",
			"VM run time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return TxSchedulerEvidence
}
