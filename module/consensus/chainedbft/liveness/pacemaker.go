/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package liveness

import (
	"sync"
	"time"

	timeservice "chainmaker.org/chainmaker-go/consensus/chainedbft/time_service"
	"chainmaker.org/chainmaker-go/logger"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
)

//Pacemaker govern the advancement of levels and height in the local node.
//The Pacemaker keeps track of qc and tc; which qc is proposal type, tc is newView type.
type Pacemaker struct {
	rwMtx                 sync.RWMutex
	height                uint64                   // The height of the latest received QC
	epochId               uint64                   // The epochID in now
	currentLevel          uint64                   // The current level of the local node
	highestQCLevel        uint64                   // The highest QC in local node
	highestTCLevel        uint64                   // The tc level in incoming msg(proposal or vote)
	highestCommittedLevel uint64                   // The latest committed level in local node
	timeoutCertificate    *chainedbftpb.QuorumCert // The latest timeout QC info

	ts               *timeservice.TimerService // Timer service
	logger           *logger.CMLogger
	selfIndexInEpoch uint64 // The index of the local node in the validator set of this epoch
}

//NewPacemaker init a pacemaker
func NewPacemaker(logger *logger.CMLogger, index uint64,
	height uint64, epochId uint64, ts *timeservice.TimerService) *Pacemaker {
	return &Pacemaker{
		ts:               ts,
		rwMtx:            sync.RWMutex{},
		logger:           logger,
		height:           height,
		epochId:          epochId,
		selfIndexInEpoch: index,
	}
}

//GetHeight get height
func (p *Pacemaker) GetHeight() uint64 {
	p.rwMtx.RLock()
	defer p.rwMtx.RUnlock()
	return p.height
}

//GetCurrentLevel get current level
func (p *Pacemaker) GetCurrentLevel() uint64 {
	p.rwMtx.RLock()
	defer p.rwMtx.RUnlock()
	return p.currentLevel
}

func (p *Pacemaker) GetEpochId() uint64 {
	p.rwMtx.RLock()
	defer p.rwMtx.RUnlock()
	return p.epochId
}

//GetHighestTCLevel get highest timeout qc' level
func (p *Pacemaker) GetHighestTCLevel() uint64 {
	p.rwMtx.RLock()
	defer p.rwMtx.RUnlock()
	return p.highestTCLevel
}

//ProcessLocalTimeout process local timeout, setup pacemaker ticker
func (p *Pacemaker) ProcessLocalTimeout(level uint64) bool {
	p.rwMtx.RLock()
	defer p.rwMtx.RUnlock()
	if level != p.currentLevel {
		return false
	}
	p.setupTimeout()
	return true
}

//GetTC get timeout qc
func (p *Pacemaker) GetTC() *chainedbftpb.QuorumCert {
	p.rwMtx.RLock()
	defer p.rwMtx.RUnlock()
	return p.timeoutCertificate
}

//UpdateTC update incoming tc, update internal state
func (p *Pacemaker) UpdateTC(tc *chainedbftpb.QuorumCert) {
	p.rwMtx.Lock()
	defer p.rwMtx.Unlock()
	if p.timeoutCertificate == nil {
		p.timeoutCertificate = tc
		return
	}
	if tc.Height <= p.timeoutCertificate.Height ||
		tc.Level <= p.timeoutCertificate.Level {
		return
	}
	p.timeoutCertificate = tc
}

// ProcessCertificates Push status of consensus to the next block height or level, and set
// a local timeout `ConsStateType_PaceMaker` when a new level is reached.
// height The height of the received QC
// hqcLevel The highest QC in local node
// htcLevel The tc level in incoming msg(proposal or vote),
// hcLevel The latest committed level in local node
// When the consensus enters the next level, return true, otherwise return false.
func (p *Pacemaker) ProcessCertificates(height, hqcLevel, htcLevel, hcLevel uint64) bool {
	p.rwMtx.Lock()
	defer func() {
		p.logger.Debugf("process certificates (height:%d, smrHeight:%d, hqcLevel:%d, "+
			"smrCurrentLevel:%d, htcLevel:%d, smrHtcLevel:%d, hcLevel:%d, smrHCLevel:%d", height,
			p.height, hqcLevel, p.currentLevel, htcLevel, p.highestTCLevel, hcLevel, p.highestCommittedLevel)
		p.rwMtx.Unlock()
	}()
	if hcLevel > p.highestCommittedLevel {
		p.highestCommittedLevel = hcLevel
	}
	if hqcLevel > p.highestQCLevel {
		p.highestQCLevel = hqcLevel
	}
	if htcLevel > p.highestTCLevel {
		p.highestTCLevel = htcLevel
	}
	if height > p.height {
		p.height = height
	}

	maxLevel := p.highestTCLevel
	// When htcLevel> hqcLevel, it means consensus has timeout, otherwise, it means height +1
	if hqcLevel >= p.highestTCLevel {
		maxLevel = hqcLevel
		p.height = height + 1
	}
	newLevel := maxLevel + 1
	if newLevel > p.currentLevel {
		p.currentLevel = newLevel
		p.setupTimeout()
		return true
	}
	return false
}

func (p *Pacemaker) setupTimeout() {
	diff, duration := p.getTimeDuration(timeservice.ROUND_TIMEOUT)
	newLevelEvent := &timeservice.TimerEvent{
		State:      chainedbftpb.ConsStateType_PaceMaker,
		Index:      p.selfIndexInEpoch,
		Level:      p.currentLevel,
		Height:     p.height,
		EpochId:    p.epochId,
		Duration:   duration,
		LevelIndex: diff,
	}
	p.ts.AddEvent(newLevelEvent)
}

func (p *Pacemaker) getTimeDuration(eventType timeservice.TimerEventType) (diff uint64, duration time.Duration) {
	index := p.currentLevel - 1
	if p.highestCommittedLevel != 0 {
		if p.currentLevel-p.highestCommittedLevel < 3 {
			p.logger.Debugf("setup timeout (service index [%v] currentLevel [%v] highestCommittedLevel [%v])\n",
				p.selfIndexInEpoch, p.currentLevel, p.highestCommittedLevel)
			index = 0
		} else {
			index = p.currentLevel - p.highestCommittedLevel - 3
		}
	}
	return index, timeservice.GetEventTimeout(eventType, int32(index))
}
