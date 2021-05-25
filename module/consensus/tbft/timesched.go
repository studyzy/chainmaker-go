/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"fmt"
	"time"

	"chainmaker.org/chainmaker-go/logger"

	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
	"github.com/gogo/protobuf/types"
)

var (
	defaultTimeSchedulerBufferSize = 10
)

// timeoutInfo is used for consensus state transition because of
// timeout
type timeoutInfo struct {
	time.Duration
	Height int64
	Round  int32
	Step   tbftpb.Step
}

func (ti timeoutInfo) String() string {
	return fmt.Sprintf("timeoutInfo(%v-%d/%d/%s)", ti.Duration, ti.Height, ti.Round, ti.Step)
}

func (ti timeoutInfo) ToProto() *tbftpb.TimeoutInfo {
	return &tbftpb.TimeoutInfo{
		Duration: types.DurationProto(ti.Duration),
		Height:   ti.Height,
		Round:    ti.Round,
		Step:     ti.Step,
	}
}

func newTimeoutInfoFromProto(ti *tbftpb.TimeoutInfo) timeoutInfo {
	duration, err := types.DurationFromProto(ti.Duration)
	if err != nil {
		panic(err)
	}
	return timeoutInfo{
		Duration: duration,
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
func NewTimeSheduler(logger *logger.CMLogger, id string) *timeScheduler {
	ts := &timeScheduler{
		logger:   logger,
		id:       id,
		timer:    time.NewTimer(0),
		bufferC:  make(chan timeoutInfo, defaultTimeSchedulerBufferSize),
		timeoutC: make(chan timeoutInfo, defaultTimeSchedulerBufferSize),
		stopC:    make(chan struct{}),
	}
	ts.stopTimer()

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
			ts.logger.Debugf("[%s] %s receive timeoutInfo: %s", ts.id, ti, t)

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
