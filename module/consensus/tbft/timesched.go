/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"fmt"
	"time"

	"chainmaker.org/chainmaker/logger/v2"

	tbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/tbft"
)

var (
	defaultTimeSchedulerBufferSize = 10
)

// timeoutInfo is used for consensus state transition because of
// timeout
type timeoutInfo struct {
	time.Duration
	Height uint64
	Round  int32
	Step   tbftpb.Step
}

func (ti timeoutInfo) String() string {
	return fmt.Sprintf("timeoutInfo(%v-%d/%d/%s)", ti.Duration, ti.Height, ti.Round, ti.Step)
}

func (ti timeoutInfo) ToProto() *tbftpb.TimeoutInfo {
	return &tbftpb.TimeoutInfo{
		Duration: ti.Duration.Microseconds(),
		Height:   ti.Height,
		Round:    ti.Round,
		Step:     ti.Step,
	}
}

func newTimeoutInfoFromProto(ti *tbftpb.TimeoutInfo) timeoutInfo {
	return timeoutInfo{
		Duration: time.Duration(ti.Duration),
		Height:   ti.Height,
		Round:    ti.Round,
		Step:     ti.Step,
	}
}

// timeScheduler is used by consensus for shecdule timeout events.
// Outdated timeouts will be ignored in processing.
type timeScheduler struct {
	logger   *logger.CMLogger
	id       string
	timer    *time.Timer
	bufferC  chan timeoutInfo
	timeoutC chan timeoutInfo
	stopC    chan struct{}
}

// NewTimeSheduler returns a new timeScheduler
func newTimeSheduler(logger *logger.CMLogger, id string) *timeScheduler {
	ts := &timeScheduler{
		logger:   logger,
		id:       id,
		timer:    time.NewTimer(0),
		bufferC:  make(chan timeoutInfo, defaultTimeSchedulerBufferSize),
		timeoutC: make(chan timeoutInfo, defaultTimeSchedulerBufferSize),
		stopC:    make(chan struct{}),
	}
	if !ts.timer.Stop() {
		<-ts.timer.C
	}

	return ts
}

// Start starts the timeScheduler
func (ts *timeScheduler) Start() {
	go ts.handle()
}

// Stop stops the timeScheduler
func (ts *timeScheduler) Stop() {
	close(ts.stopC)
}

// stopTimer stop timer of timeScheduler
func (ts *timeScheduler) stopTimer() {
	ts.timer.Stop()
}

// AddTimeoutInfo add a timeoutInfo event to timeScheduler
func (ts *timeScheduler) AddTimeoutInfo(ti timeoutInfo) {
	ts.logger.Infof("len(ts.bufferC): %d", len(ts.bufferC))
	ts.bufferC <- ti
}

// GetTimeoutC returns timeoutC for consuming
func (ts *timeScheduler) GetTimeoutC() <-chan timeoutInfo {
	return ts.timeoutC
}

func (ts *timeScheduler) handle() {
	ts.logger.Debugf("[%s] start handle timeout", ts.id)
	defer ts.logger.Debugf("[%s] stop handle timeout", ts.id)
	var ti timeoutInfo // lastest timeout event had been seen
	for {
		select {
		case t := <-ts.bufferC:
			ts.logger.Debugf("[%s] %s receive timeoutInfo: %s, ts.bufferC len: %d", ts.id, ti, t, len(ts.bufferC))

			// ignore outdated timeouts
			if t.Height < ti.Height {
				continue
			} else if t.Height == ti.Height {
				if t.Round < ti.Round {
					continue
				} else if t.Round == ti.Round && t.Step <= ti.Step {
					continue
				}
			}

			// stop timer first
			ts.stopTimer()

			// update with new timeout
			ti = t
			ts.timer.Reset(ti.Duration)
			ts.logger.Debugf("[%s] schedule %s", ts.id, ti)

		case <-ts.timer.C: // timeout
			ts.logger.Debugf("[%s] %s timeout", ts.id, ti)
			ts.timeoutC <- ti
		case <-ts.stopC:
			return
		}
	}
}
