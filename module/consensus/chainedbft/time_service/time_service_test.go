/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package timeservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
)

func TestTimerService_AddEvent(t *testing.T) {
	timerService := NewTimerService()
	go timerService.Start()
	firedCh := timerService.GetFiredCh()

	// 1. add paceEvent and check no timeOut event
	paceEvent := TimerEvent{
		Duration: time.Millisecond,
		State:    chainedbftpb.ConsStateType_PaceMaker,
	}
	timerService.AddEvent(&paceEvent)
	checkNoTimeOutEvent(t, firedCh)

	// 2. sleep to fired timeout
	time.Sleep(paceEvent.Duration * 2)
	checkTimeOutEvent(t, firedCh)

	// 3. re-add paceEvent event
	timerService.AddEvent(&paceEvent)
	checkNoTimeOutEvent(t, firedCh)

	timerService.AddEvent(&paceEvent)
	checkNoTimeOutEvent(t, firedCh)
	time.Sleep(paceEvent.Duration)
	dropTimerC(timerService.pacemakerTimer, "pacemaker", timerService.logger)

}

func TestTimerService_AddEvent2(t *testing.T) {
	timerService := NewTimerService()
	go timerService.Start()
	firedCh := timerService.GetFiredCh()

	// 4. add other event and no timeOut
	event := TimerEvent{
		Duration: time.Millisecond * 1,
		State:    chainedbftpb.ConsStateType_NewLevel,
		Height:   10,
	}

	timerService.AddEvent(&event)
	checkNoTimeOutEvent(t, firedCh)

	// 5. sleep to wait timeOut
	time.Sleep(event.Duration * 2)
	checkTimeOutEvent(t, firedCh)

	// 6. add event failed, so no timeOut
	timerService.AddEvent(&event)
	time.Sleep(event.Duration * 2)
	checkNoTimeOutEvent(t, firedCh)
}

func checkNoTimeOutEvent(t *testing.T, ch <-chan *TimerEvent) {
	select {
	case event := <-ch:
		require.Fail(t, "should no timeOut event, ", event)
	default:
	}
}

func checkTimeOutEvent(t *testing.T, ch <-chan *TimerEvent) {
	select {
	case <-ch:
	default:
		require.Fail(t, "should have timeOut event")
	}
}
