/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package timeservice

import (
	"time"

	"chainmaker.org/chainmaker-go/logger"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
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
	RoundTimeoutMill            = "HOTSTUFF_round_timeout_milli"
	RoundTimeoutIntervalMill    = "HOTSTUFF_round_timeout_delta_milli"
	ProposerTimeoutMill         = "HOTSTUFF_proposer_timeout_milli"
	ProposerTimeoutIntervalMill = "HOTSTUFF_proposer_timeout_delta_milli"
)

var (
	RoundTimeout            = 6000 * time.Millisecond
	RoundTimeoutInterval    = 500 * time.Millisecond
	ProposerTimeout         = 2000 * time.Millisecond
	ProposerTimeoutInterval = 500 * time.Millisecond
)

//GetEventTimeout returns the time duration per event type and consensus roundIndex
func GetEventTimeout(evtType TimerEventType, roundIndex int32) time.Duration {
	switch evtType {
	case PROPOSAL_BLOCK_TIMEOUT:
		return time.Duration(ProposerTimeout.Nanoseconds()+
			ProposerTimeoutInterval.Nanoseconds()*int64(roundIndex)) * time.Nanosecond
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
	State      chainedbftpb.ConsStateType // Monitored events
}

//TimerService provides timer service
type TimerService struct {
	timer          *time.Timer // timer for event(not include ConsStateType_PaceMaker event)
	lastEvent      *TimerEvent // The last timeout event added
	pacemakerEvent *TimerEvent // The last pacemaker event added
	pacemakerTimer *time.Timer // timer for pacemaker event

	eventCh chan *TimerEvent // For scheduling event timeouts
	firedCh chan *TimerEvent // For notifying event timeouts
	quitCh  chan struct{}    // Quit timer service

	logger *logger.CMLogger // log
}

//NewTimerService initializes an instance of timer service
func NewTimerService() *TimerService {
	ts := &TimerService{
		timer:          time.NewTimer(0),
		pacemakerTimer: time.NewTimer(RoundTimeout),

		eventCh: make(chan *TimerEvent, 10),
		firedCh: make(chan *TimerEvent, 10),
		quitCh:  make(chan struct{}, 0),
		logger:  logger.GetLogger(logger.MODULE_CONSENSUS),
	}
	dropTimerC(ts.timer, "event", ts.logger)
	dropTimerC(ts.pacemakerTimer, "PaceMaker", ts.logger)
	return ts
}

//Start starts timer service
func (ts *TimerService) Start() {
	ts.loop()
}

//Stop stops timer service
func (ts *TimerService) Stop() {
	close(ts.quitCh)
	dropTimerC(ts.timer, "event", ts.logger)
	dropTimerC(ts.pacemakerTimer, "PaceMaker", ts.logger)
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
				continue
			}
			ts.processEvent(newEvent)
		case <-ts.timer.C:
			if ts.lastEvent != nil {
				go ts.fireEvent(ts.lastEvent, "state")
			}
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
	ts.logger.Debug("received a timer event ", newEvent, " last timer event ", ts.lastEvent)
	if newEvent.State == chainedbftpb.ConsStateType_PaceMaker {
		ts.pacemakerEvent = newEvent
		dropTimerC(ts.pacemakerTimer, "Pacemaker", ts.logger)
		ts.pacemakerTimer.Reset(ts.pacemakerEvent.Duration)
		return
	}

	if !validNewEvent(ts.lastEvent, newEvent) {
		return
	}
	dropTimerC(ts.timer, "event", ts.logger)
	ts.lastEvent = newEvent
	ts.timer.Reset(ts.lastEvent.Duration)
	ts.logger.Debugf("time service scheduled state timeout: duration [%v] height [%v] levelIndex [%v] level [%v] "+
		"state [%v] for service index [%v]", newEvent.Duration, newEvent.Height, newEvent.LevelIndex,
		newEvent.Level, newEvent.State, newEvent.Index)
}

func (ts *TimerService) fireEvent(firedEvent *TimerEvent, detailType string) {
	ts.logger.Debugf("fired %s event %v", detailType, firedEvent)
	ts.firedCh <- firedEvent
}

//GetFiredCh returns a channel to receive events
func (ts *TimerService) GetFiredCh() <-chan *TimerEvent {
	return ts.firedCh
}

func validNewEvent(lastEvent, newEvent *TimerEvent) bool {
	if lastEvent == nil {
		return true
	}
	if newEvent.Height < lastEvent.Height && newEvent.EpochId == lastEvent.EpochId {
		return false
	}
	if newEvent.Height == lastEvent.Height && newEvent.EpochId == lastEvent.EpochId {
		if newEvent.LevelIndex < lastEvent.LevelIndex || newEvent.Level < lastEvent.Level {
			return false
		} else if newEvent.Level == lastEvent.Level && newEvent.State > 0 && newEvent.State <= lastEvent.State {
			return false
		}
	}
	return true
}
