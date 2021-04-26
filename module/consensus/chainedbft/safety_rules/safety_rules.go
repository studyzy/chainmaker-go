/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package safetyrules

import (
	"encoding/hex"
	"sync"

	blockpool "chainmaker.org/chainmaker-go/consensus/chainedbft/block_pool"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
)

//SafetyRules implementation to validate incoming qc and block, include commit rules(3-chain) and vote rules
type SafetyRules struct {
	sync.RWMutex

	lastCommittedLevel uint64        // the latest committed level in local node
	lastCommittedBlock *common.Block // the latest committed block in local node

	lastVoteMsg   *chainedbftpb.ConsensusPayload //
	lastVoteLevel uint64                         //

	lockedLevel uint64        // the latest locked level in local node
	lockedBlock *common.Block // the latest locked block in local node

	logger    *logger.CMLogger
	blockPool *blockpool.BlockPool // store blocks and qc in memory
}

//NewSafetyRules init a SafetyRules
func NewSafetyRules(logger *logger.CMLogger, blockPool *blockpool.BlockPool) *SafetyRules {
	return &SafetyRules{
		logger:    logger,
		blockPool: blockPool,
	}
}

//GetLastVoteLevel get last vote's level
func (sr *SafetyRules) GetLastVoteLevel() uint64 {
	sr.RLock()
	defer sr.RUnlock()
	return sr.lastVoteLevel
}

//GetLastVoteMsg get last vote msg
func (sr *SafetyRules) GetLastVoteMsg() *chainedbftpb.ConsensusPayload {
	sr.RLock()
	defer sr.RUnlock()
	return sr.lastVoteMsg
}

//GetLastCommittedBlock get last commiteed block
func (sr *SafetyRules) GetLastCommittedBlock() *common.Block {
	sr.RLock()
	defer sr.RUnlock()
	return sr.lastCommittedBlock
}

//GetLastCommittedLevel get last committeed level
func (sr *SafetyRules) GetLastCommittedLevel() uint64 {
	sr.RLock()
	defer sr.RUnlock()
	return sr.lastCommittedLevel
}

//GetLockedLevel get locked level
func (sr *SafetyRules) GetLockedLevel() uint64 {
	sr.RLock()
	defer sr.RUnlock()
	return sr.lockedLevel
}

//GetLockedBlock get locked block
func (sr *SafetyRules) GetLockedBlock() *common.Block {
	sr.RLock()
	defer sr.RUnlock()
	return sr.lockedBlock
}

//SetLastVote set last vote
func (sr *SafetyRules) SetLastVote(vote *chainedbftpb.ConsensusPayload, level uint64) {
	sr.Lock()
	defer sr.Unlock()
	if level <= sr.lastVoteLevel {
		return
	}
	sr.lastVoteMsg = vote
	sr.lastVoteLevel = level
}

//SetLastCommittedBlock set last committed blcok
func (sr *SafetyRules) SetLastCommittedBlock(block *common.Block, level uint64) {
	sr.Lock()
	defer sr.Unlock()
	if level <= sr.lastCommittedLevel ||
		block.Header.BlockHeight <= sr.lastCommittedBlock.Header.BlockHeight {
		return
	}
	sr.lastCommittedBlock = block
	sr.lastCommittedLevel = level
}

//VoteRules validate incoming block and qc to vote
func (sr *SafetyRules) VoteRules(level uint64, qc *chainedbftpb.QuorumCert) bool {
	sr.RLock()
	defer sr.RUnlock()

	if level <= sr.lastVoteLevel {
		sr.logger.Debugf("vote rules failed,"+
			" level <= lastVote.level, level [%v], lastVote level [%v]", level, sr.lastVoteLevel)
		return false
	}
	var (
		err     error
		qcLevel uint64
		qcBlock *common.Block
	)
	if qcBlock = sr.blockPool.GetBlockByID(string(qc.BlockID)); qcBlock == nil {
		sr.logger.Debugf("vote rules failed, preblock not exist, pre block hash [%v]", hex.EncodeToString(qc.BlockID))
		return false
	}
	if qcLevel, err = utils.GetLevelFromBlock(qcBlock); err != nil {
		sr.logger.Debugf("vote rules failed, get parent block's level error, block hash [%v], err %v",
			qcBlock.Header.BlockHash, err)
		return false
	}
	if qcLevel != qc.Level {
		sr.logger.Debugf("vote rules failed, parent block's level is not equal qc'level, block level [%v], qc level [%v]",
			qcLevel, qc.Level)
		return false
	}
	if qcLevel < sr.lockedLevel {
		sr.logger.Debugf("vote rules failed, preLevel <= locked Level, preLevel [%v], locked level [%v]",
			qcLevel, sr.lockedLevel)
		return false
	}
	return true
}

//CommitRules validate incoming qc to commit by three-chain
func (sr *SafetyRules) CommitRules(qc *chainedbftpb.QuorumCert) (bool, *common.Block, uint64) {
	if qc == nil {
		sr.logger.Debugf("commit rules failed, qc is nil")
		return false, nil, 0
	}
	if qc.NewView {
		sr.logger.Debugf("commit rules failed, qc is new view tc")
		return false, nil, 0
	}

	sr.Lock()
	defer sr.Unlock()
	var (
		curQC       = qc
		qcBlock     *common.Block
		parentBlock *common.Block
		grandBlock  *common.Block
	)
	if qcBlock = sr.blockPool.GetBlockByID(string(qc.BlockID)); qcBlock == nil {
		sr.logger.Debugf("commit rules failed, qc's block[%x] is nil", qc.BlockID)
		return false, nil, 0
	}
	if parentBlock = sr.blockPool.GetBlockByID(string(qcBlock.Header.PreBlockHash)); parentBlock == nil {
		sr.logger.Debugf("commit rules failed, qc's parent[%x] block is nil", qc.BlockID)
		return false, nil, 0
	}
	if grandBlock = sr.blockPool.GetBlockByID(string(parentBlock.Header.PreBlockHash)); grandBlock == nil {
		sr.logger.Debugf("commit rules failed, qc's grandBlock is nil")
		return false, nil, 0
	}

	var (
		parentQC = sr.blockPool.GetQCByID(string(parentBlock.Header.BlockHash))
		grandQC  = sr.blockPool.GetQCByID(string(grandBlock.Header.BlockHash))
	)
	if parentQC == nil || grandQC == nil {
		sr.logger.Debugf("commit rules failed, qc's parent qc or parent parent qc is nil")
		return false, nil, 0
	}
	if curQC.Height == parentQC.Height+1 && parentQC.Height == grandQC.Height+1 {
		sr.logger.Debugf("commit rules success, qc satisfy three-chain, qc level [%v], "+
			"parent level [%v], grand level [%v]", curQC.Height, parentQC.Height, grandQC.Height)
		return true, grandBlock, grandQC.Level
	}
	sr.logger.Debugf("commit rules failed, qc not satisfy three-chain, qc level [%v], "+
		"parent level [%v], grand level [%v]", curQC.Level, parentQC.Level, grandQC.Level)
	return false, nil, 0
}

//UpdateLockedQC process incoming qc, update locked state by two-chain
func (sr *SafetyRules) UpdateLockedQC(qc *chainedbftpb.QuorumCert) {
	if qc.NewView || qc.BlockID == nil {
		sr.logger.Debugf("incoming qc failed, received new view or nil block id")
		return
	}

	sr.Lock()
	defer sr.Unlock()
	var (
		block     *common.Block
		prevBlock *common.Block
		prevQC    *chainedbftpb.QuorumCert
	)
	if block = sr.blockPool.GetBlockByID(string(qc.BlockID)); block == nil {
		sr.logger.Debugf("incoming qc failed, nil block for qc certified id [%v] on [%v]:[%v]",
			hex.EncodeToString(qc.BlockID), qc.Height, qc.Level)
		return
	}
	if prevBlock = sr.blockPool.GetBlockByID(string(block.Header.PreBlockHash)); prevBlock == nil {
		return
	}
	if prevQC = sr.blockPool.GetQCByID(string(block.Header.PreBlockHash)); prevQC == nil {
		return
	}
	rootHash := string(sr.blockPool.GetRootBlock().Header.BlockHash)
	if rootQC := sr.blockPool.GetQCByID(rootHash); prevQC.Level <= rootQC.Level {
		return
	}
	if prevQC.Level <= sr.lockedLevel {
		return
	}
	sr.logger.Debugf("incoming qc success, update locked level from old [%v] to new [%v] ",
		sr.lockedLevel, prevQC.Level)
	sr.lockedLevel = prevQC.Level
	sr.lockedBlock = prevBlock
}
