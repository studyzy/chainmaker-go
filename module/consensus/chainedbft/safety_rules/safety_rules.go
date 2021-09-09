/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package safetyrules

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	blockpool "chainmaker.org/chainmaker-go/consensus/chainedbft/block_pool"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"chainmaker.org/chainmaker/protocol/v2"
)

//SafetyRules implementation to validate incoming qc and block, include commit rules(3-chain) and vote rules
type SafetyRules struct {
	sync.RWMutex
	chainStore protocol.BlockchainStore

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
func NewSafetyRules(logger *logger.CMLogger, blkPool *blockpool.BlockPool,
	chainStore protocol.BlockchainStore) *SafetyRules {
	sf := &SafetyRules{
		logger:      logger,
		blockPool:   blkPool,
		chainStore:  chainStore,
		lockedBlock: blkPool.GetRootBlock(),
		lockedLevel: blkPool.GetHighestQC().Level,
	}
	return sf
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

//GetLastCommittedBlock get last committed block
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
	if level <= sr.lastCommittedLevel || (sr.lastCommittedBlock != nil &&
		block.Header.BlockHeight <= sr.lastCommittedBlock.Header.BlockHeight) {
		return
	}
	sr.lastCommittedBlock = block
	sr.lastCommittedLevel = level
}

func (sr *SafetyRules) getBlockByHash(blkHash string) *common.Block {
	if blk := sr.blockPool.GetBlockByID(blkHash); blk != nil {
		return blk
	}
	if blk, err := sr.chainStore.GetBlockByHash([]byte(blkHash)); err == nil && blk != nil {
		return blk
	}
	return nil
}

//SafeNode validate incoming block and qc to vote
func (sr *SafetyRules) SafeNode(proposal *chainedbftpb.ProposalData) error {
	sr.RLock()
	defer sr.RUnlock()

	var (
		justQc = proposal.JustifyQc
	)

	// 1. 活性规则：The liveness rule is the replica will accept m
	// if m.justify has a higher view than the current locked QC
	if justQc.Level > sr.lockedLevel {
		sr.logger.Infof("safeNode success: proposal: %x satisfy liveness rules", proposal.Block.Header.BlockHash)
		return nil
	}

	// 2. 安全规则：The safety rule to accept a proposal is the branch of m.node
	// extends from the currently locked node locked QC.node
	currBlock := proposal.Block
	currHeight := proposal.Height
	for currBlock != nil && currHeight > uint64(sr.lockedBlock.Header.BlockHeight) {
		currBlock = sr.getBlockByHash(string(currBlock.Header.PreBlockHash))
		if currBlock != nil {
			currHeight = uint64(currBlock.Header.BlockHeight)
		}
	}
	if currBlock == nil {
		return fmt.Errorf("not found block: %d", currHeight-1)
	}
	if !bytes.Equal(currBlock.Header.BlockHash, sr.lockedBlock.Header.BlockHash) {
		return fmt.Errorf("safety rules failed, not extend block from lockedBlock, proposal: %x extend "+
			"from: %x, lockedBlock: %x", proposal.Block.Header.BlockHash, currBlock.Header.BlockHash,
			sr.lockedBlock.Header.BlockHash)
	}
	sr.logger.Infof("safeNode success: proposal: %x satisfy safety rules", proposal.Block.Header.BlockHash)
	return nil
}

//CommitRules validate incoming qc to commit by three-chain
func (sr *SafetyRules) CommitRules(qc *chainedbftpb.QuorumCert) (commit bool, commitBlock *common.Block,
	commitLevel uint64) {
	if qc == nil {
		sr.logger.Debugf("commit rules, qc is nil")
		return false, nil, 0
	}
	if qc.NewView {
		sr.logger.Debugf("commit rules, qc is new view tc")
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
	if qcBlock = sr.blockPool.GetBlockByID(string(qc.BlockId)); qcBlock == nil {
		sr.logger.Debugf("commit rules, qc's block[%x] is nil", qc.BlockId)
		return false, nil, 0
	}
	if parentBlock = sr.blockPool.GetBlockByID(string(qcBlock.Header.PreBlockHash)); parentBlock == nil {
		sr.logger.Debugf("commit rules, qc's parent[%x] block is nil", qc.BlockId)
		return false, nil, 0
	}
	if grandBlock = sr.blockPool.GetBlockByID(string(parentBlock.Header.PreBlockHash)); grandBlock == nil {
		sr.logger.Debugf("commit rules, qc's grandBlock is nil")
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
	if qc == nil || qc.NewView || qc.BlockId == nil {
		sr.logger.Debugf("received new view or nil block id qc, info: %s", qc.String())
		return
	}

	sr.Lock()
	defer sr.Unlock()
	var (
		block     *common.Block
		prevBlock *common.Block
		prevQC    *chainedbftpb.QuorumCert
	)
	if block = sr.blockPool.GetBlockByID(string(qc.BlockId)); block == nil {
		sr.logger.Debugf("incoming qc failed, nil block for qc certified id [%v] on [%v]:[%v]",
			hex.EncodeToString(qc.BlockId), qc.Height, qc.Level)
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
