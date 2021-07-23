/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package timeservice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//func TestTimerService_AddEvent(t *testing.T) {
//	log := logger.GetLogger("test")
//	timerService := NewTimerService(log)
//	go timerService.Start()
//	firedCh := timerService.GetFiredCh()
//
//	// 1. add paceEvent and check no timeOut event
//	paceEvent := TimerEvent{
//		Duration: time.Millisecond,
//		State:    chainedbftpb.ConsStateType_PACEMAKER,
//	}
//	timerService.AddEvent(&paceEvent)
//	checkNoTimeOutEvent(t, firedCh)
//
//	// 2. sleep to fired timeout
//	time.Sleep(paceEvent.Duration * 10)
//	checkTimeOutEvent(t, firedCh)
//
//	// 3. re-add paceEvent event
//	timerService.AddEvent(&paceEvent)
//	checkNoTimeOutEvent(t, firedCh)
//
//	timerService.AddEvent(&paceEvent)
//	checkNoTimeOutEvent(t, firedCh)
//	time.Sleep(paceEvent.Duration)
//	dropTimerC(timerService.pacemakerTimer, "pacemaker", timerService.logger)
//
//}

func checkNoTimeOutEvent(t *testing.T, ch <-chan *TimerEvent) {
	select {
	case event := <-ch:
		require.Fail(t, "should no timeOut event, ", event)
	default:
	}
}

func checkTimeOutEvent(t *testing.T, firedCh <-chan *TimerEvent) {
	select {
	case <-firedCh:
	default:
		require.Fail(t, "should have timeOut event")
	}
}
