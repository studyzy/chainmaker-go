/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package liveness

import (
	"sync"
	"time"

	timeservice "chainmaker.org/chainmaker-go/consensus/chainedbft/time_service"
	"chainmaker.org/chainmaker/logger/v2"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
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
	if tc.Level > p.timeoutCertificate.Level {
		p.timeoutCertificate = tc
	}
}

// ProcessCertificates Push status of consensus to the next block height or level, and set
// a local timeout `ConsStateType_PACE_MAKER` when a new level is reached.
// height The height of the received QC
// hqcLevel The highest QC in local node
// htcLevel The tc level in incoming msg(proposal or vote),
// hcLevel The latest committed level in local node
// When the consensus enters the next level, return true, otherwise return false.
//func (p *Pacemaker) ProcessCertificates(height, hqcLevel, htcLevel, hcLevel uint64) bool {
func (p *Pacemaker) ProcessCertificates(qc *chainedbftpb.QuorumCert, tc *chainedbftpb.QuorumCert, hcLevel uint64) bool {
	p.rwMtx.Lock()
	defer p.rwMtx.Unlock()
	p.logger.Debugf("process certificates begin (smrHeight:%d,"+
		"smrCurrentLevel:%d, smrHtcLevel:%d, smrHCLevel:%d, smrHQCLevel: %d",
		p.height, p.currentLevel, p.highestTCLevel, p.highestCommittedLevel, p.highestQCLevel)
	defer func() {
		if qc != nil {
			p.logger.Debugf("process certificates end (hqcHeight:%d, hqcLevel:%d, smrHeight:%d, smrHQCLevel:%d,"+
				" smrCurrentLevel:%d, smrHtcLevel:%d, smrHCLevel:%d,  hcLevel:%d,", qc.Height, qc.Level,
				p.height, p.highestQCLevel, p.currentLevel, p.highestTCLevel, p.highestCommittedLevel, hcLevel)
		} else {
			p.logger.Debugf("process certificates end (tcHeight:%d, tcLevel:%d, smrHeight:%d, smrHQCLevel: %d "+
				"smrCurrentLevel:%d, smrHtcLevel:%d, smrHCLevel:%d, hcLevel:%d, ", tc.Height, tc.Level,
				p.height, p.highestQCLevel, p.currentLevel, p.highestTCLevel, p.highestCommittedLevel, hcLevel)
		}
	}()
	if hcLevel > p.highestCommittedLevel {
		p.highestCommittedLevel = hcLevel
	}
	if tc != nil && tc.Level > p.highestTCLevel {
		p.highestTCLevel = tc.Level
	}
	// qc != nil: 表示接收到了新的QC信息，
	// 副本：如果 qc.Height > p.height，表示n-f节点已对更高的区块 qc.Height达成一致，此时节点应用该QC后，应该进入qc.Height+1阶段
	if qc != nil && qc.Height >= p.height {
		p.height = qc.Height + 1
	}
	if qc != nil && qc.Level > p.highestQCLevel {
		p.highestQCLevel = qc.Level
	}
	maxLevel := p.highestTCLevel
	if p.highestQCLevel > p.highestTCLevel {
		maxLevel = p.highestQCLevel
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
		State:      chainedbftpb.ConsStateType_PACEMAKER,
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
