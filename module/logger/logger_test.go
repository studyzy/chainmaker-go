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
			return helloDebug
		})
		logger.InfoDynamic(func() string {
			count++
			wg.Done()
			return helloInfo
		})
		wg.Wait()
		c <- "ok"
	}()
	select {
	case <-to.C:
		t.Fail()
	case <-c:
		t.Log("success!")
	}
}

const helloDebug = "hello dynamic debug"
const helloInfo = "hello dynamic info"

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
			return helloDebug
		})
		logger.InfoDynamic(func() string {
			count++
			wg.Done()
			return helloInfo
		})
		wg.Wait()
		c <- "ok"
	}()
	select {
	case <-to.C:
		t.Fail()
	case <-c:
		t.Log("success!")
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
		return helloDebug
	})
	logger.InfoDynamic(func() string {
		count++
		t.Fail()
		return helloInfo
	})
	if count != 0 {
		t.Fail()
	}
}

func TestRotateSize(t *testing.T) {

	logger := GetLogger(MODULE_CORE)

	for i := 0; i < 100; i++ {
		//time.Sleep(100*time.Microsecond)
		logger.Info("this is info msg ")
		logger.Debugf("hello %s", "chainmaker ")
	}
}
