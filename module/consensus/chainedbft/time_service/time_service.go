/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package timeservice

import (
	"fmt"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

//TimerEventType defines the consensus event type
type TimerEventType int

//Events: proposal block, vote block, get transaction,
//empty block, commit block, heartbeat
const (
	PROPOSAL_BLOCK_TIMEOUT TimerEventType = iota
	VOTE_BLOCK_TIMEOUT
	ROUND_TIMEOUT
)

const (
	DefaultRoundTimeout         = 6000
	DefaultRoundTimeoutInterval = 500
)

var (
	RoundTimeout         time.Duration
	RoundTimeoutInterval time.Duration
)

//GetEventTimeout returns the time duration per event type and consensus roundIndex
func GetEventTimeout(evtType TimerEventType, roundIndex int32) time.Duration {
	switch evtType {
	case ROUND_TIMEOUT:
		return time.Duration(RoundTimeout.Nanoseconds()+
			RoundTimeoutInterval.Nanoseconds()*int64(roundIndex)) * time.Nanosecond
	default:
		return 0
	}
}

//TimerEvent defines a timer event
type TimerEvent struct {
	Index      uint64                     // Index of the local node in the validator collection of the current epoch
	Level      uint64                     // level in the consensus
	Height     uint64                     // Height in the consensus
	EpochId    uint64                     // EpochId in the consensus
	LevelIndex uint64                     // diff in the committed level and current level
	Duration   time.Duration              // timeout
	PreBlkHash []byte                     // only used in proposed event
	State      chainedbftpb.ConsStateType // Monitored events
}

func (t *TimerEvent) String() string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("height: %d, level: %d, epochID: %d,"+
		" duration: %s, state: %s", t.Height, t.Level, t.EpochId, t.Duration, t.State)
}

//TimerService provides timer service
type TimerService struct {
	pacemakerEvent *TimerEvent // The last pacemaker event added
	pacemakerTimer *time.Timer // timer for pacemaker event

	eventCh chan *TimerEvent // For scheduling event timeouts
	firedCh chan *TimerEvent // For notifying event timeouts
	quitCh  chan struct{}    // Quit timer service

	logger *logger.CMLogger // log
}

//NewTimerService initializes an instance of timer service
func NewTimerService(log *logger.CMLogger) *TimerService {
	ts := &TimerService{
		pacemakerTimer: time.NewTimer(RoundTimeout),
		eventCh:        make(chan *TimerEvent, 10),
		firedCh:        make(chan *TimerEvent, 10),
		quitCh:         make(chan struct{}),
		logger:         log,
	}
	dropTimerC(ts.pacemakerTimer, "start timeService", ts.logger)
	return ts
}

//Start starts timer service
func (ts *TimerService) Start() {
	ts.loop()
}

//Stop stops timer service
func (ts *TimerService) Stop() {
	close(ts.quitCh)
	dropTimerC(ts.pacemakerTimer, "stop timeService", ts.logger)
}

func dropTimerC(t *time.Timer, detail string, log *logger.CMLogger) {
	if t != nil && !t.Stop() {
		select {
		case <-t.C:
			log.Debugf("stop timer: %s", detail)
		default:
			log.Debugf("timer:( %s ) not fired", detail)
		}
	}
}

//AddEvent adds an timer event to timer channel
func (ts *TimerService) AddEvent(event *TimerEvent) {
	ts.eventCh <- event
}

//loop listens/notifies timer events
func (ts *TimerService) loop() {
	ts.logger.Debug("starting timeout loop...")
	for {
		select {
		case newEvent, ok := <-ts.eventCh:
			if !ok {
				ts.logger.Warnf("add timeout msg failed")
				continue
			}
			ts.processEvent(newEvent)
		case <-ts.pacemakerTimer.C:
			if ts.pacemakerEvent != nil {
				go ts.fireEvent(ts.pacemakerEvent, "pacemaker")
			}
		case <-ts.quitCh:
			return
		}
	}
}

func (ts *TimerService) processEvent(newEvent *TimerEvent) {
	ts.logger.Debugf("received a timer event: %s, last timer event: %s", newEvent, ts.pacemakerEvent)
	if newEvent.State == chainedbftpb.ConsStateType_PACEMAKER {
		ts.pacemakerEvent = newEvent
		dropTimerC(ts.pacemakerTimer, "Pacemaker", ts.logger)
		ts.pacemakerTimer.Reset(ts.pacemakerEvent.Duration)
		return
	}
}

func (ts *TimerService) fireEvent(firedEvent *TimerEvent, detailType string) {
	ts.logger.Debugf("fired %s event: %v", detailType, firedEvent)
	ts.firedCh <- firedEvent
}

//GetFiredCh returns a channel to receive events
func (ts *TimerService) GetFiredCh() <-chan *TimerEvent {
	return ts.firedCh
}
