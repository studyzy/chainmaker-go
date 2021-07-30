/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"sync"
	"testing"
	"time"
)

func TestLogger(_ *testing.T) {

	logger := GetLogger(MODULE_CORE)
	logger.Infof("core log ......")

	logger = GetLogger(MODULE_CONSENSUS)
	logger.Infof("consensus log .....")

	logger = GetLogger(MODULE_EVENT)
	logger.Infof("event log .....")

	logger = GetLogger(MODULE_BRIEF)
	logger.Infof("brief log .....")
}

func TestDebugDynamicLog(t *testing.T) {
	logConfig = DefaultLogConfig()
	logConfig.SystemLog.LogInConsole = true
	logConfig.SystemLog.LogLevelDefault = "DEBUG"
	logger := GetLogger("DebugTest")
	count := 0
	to := time.NewTicker(time.Second)
	logger.Debug("start debug log")
	logger.Error("error log include trace")
	wg := sync.WaitGroup{}
	wg.Add(2)
	c := make(chan string)
	go func() {
		logger.DebugDynamic(func() string {
			count++
			wg.Done()
			return "hello dynamic debug"
		})
		logger.InfoDynamic(func() string {
			count++
			wg.Done()
			return "hello dynamic info"
		})
		wg.Wait()
		c <- "ok"
	}()
	select {
	case <-to.C:
		t.Fail()
	case <-c:
		t.Log("succes!")
	}
}

func TestInfoDynamicLog(t *testing.T) {
	logConfig = DefaultLogConfig()
	logConfig.SystemLog.LogInConsole = true
	logConfig.SystemLog.LogLevelDefault = "INFO"
	logger := GetLogger("InfoTest")
	count := 0
	to := time.NewTicker(time.Second)
	wg := sync.WaitGroup{}
	wg.Add(1)
	c := make(chan string)
	go func() {
		logger.DebugDynamic(func() string {
			count++
			t.Fail()
			return "hello dynamic debug"
		})
		logger.InfoDynamic(func() string {
			count++
			wg.Done()
			return "hello dynamic info"
		})
		wg.Wait()
		c <- "ok"
	}()
	select {
	case <-to.C:
		t.Fail()
	case <-c:
		t.Log("succes!")
	}

}

func TestDynamicLogWhenWarnLevel(t *testing.T) {
	logConfig = DefaultLogConfig()
	logConfig.SystemLog.LogInConsole = true
	logConfig.SystemLog.LogLevelDefault = "WARN"
	logger := GetLogger("WarnTest")
	count := 0
	logger.DebugDynamic(func() string {
		count++
		t.Fail()
		return "hello dynamic debug"
	})
	logger.InfoDynamic(func() string {
		count++
		t.Fail()
		return "hello dynamic info"
	})
	if count != 0 {
		t.Fail()
	}
}

func TestRotateSize(t *testing.T) {

	logger := GetLogger(MODULE_CORE)
	logger2 := GetLogger(MODULE_CONSENSUS)
	loggerBrief := GetLogger(MODULE_BRIEF)
	loggerEvent := GetLogger(MODULE_EVENT)
	loggerAccess := GetLogger(MODULE_ACCESS)
	loggerBC := GetLogger(MODULE_BLOCKCHAIN)

	loggerCli := GetLogger(MODULE_CLI)
	loggerDPOS := GetLogger(MODULE_DPOS)
	loggerLedger := GetLogger(MODULE_LEDGER)
	loggerMonitor := GetLogger(MODULE_MONITOR)
	loggerNet := GetLogger(MODULE_NET)
	loggerRpc := GetLogger(MODULE_RPC)

	loggerSNAPSHOT := GetLogger(MODULE_SNAPSHOT)
	loggerSPV := GetLogger(MODULE_SPV)
	loggerStorage := GetLogger(MODULE_STORAGE)
	loggerSync := GetLogger(MODULE_SYNC)
	loggerTxpool := GetLogger(MODULE_TXPOOL)
	loggerVm := GetLogger(MODULE_VM)

	go printLog(logger,1)
	go printLog(logger2,2)
	go printLog(loggerBrief,3)
	go printLog(loggerEvent,4)
	go printLog(loggerAccess,5)
	go printLog(loggerBC,6)

	//---------------------------------------
	go printLog(loggerCli,1)
	go printLog(loggerDPOS,2)
	go printLog(loggerLedger,3)
	go printLog(loggerMonitor,4)
	go printLog(loggerNet,5)
	go printLog(loggerRpc,6)

	//---------------------------------------
	go printLog(loggerSNAPSHOT,1)
	go printLog(loggerSPV,2)
	go printLog(loggerStorage,3)
	go printLog(loggerSync,4)
	go printLog(loggerTxpool,5)
	go printLog(loggerVm,6)
	select {

	}
}

func printLog(logger *CMLogger,index int){
	for {
		//time.Sleep(100*time.Microsecond)
		logger.Info("this is info msg ",index)
		logger.Debugf("hello %s", "chainmaker %d",index)
	}
}