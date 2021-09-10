/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"sync/atomic"

	"chainmaker.org/chainmaker/logger/v2"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

const (
	MaxSyncBlockNum = 10
)

//blockSyncReq defines a block sync request
type blockSyncReq struct {
	height     uint64 // Height of QC received from other nodes
	blockID    []byte // BlockID of QC received from other nodes
	targetPeer uint64 // The identity of the requested node
}

// syncManager Synchronize block data from other peers
type syncManager struct {
	currReqID uint64 // The ID of the current request

	quitC         chan struct{}
	blockSyncReqC chan *blockSyncReq // receive req from local node

	logger *logger.CMLogger
	server *ConsensusChainedBftImpl
}

func newSyncManager(server *ConsensusChainedBftImpl) *syncManager {
	return &syncManager{
		currReqID:     1,
		quitC:         make(chan struct{}),
		blockSyncReqC: make(chan *blockSyncReq),

		server: server,
		logger: server.logger,
	}
}

func (sm *syncManager) start() {
	go sm.reqLoop()
}

func (sm *syncManager) stop() {
	close(sm.quitC)
	close(sm.blockSyncReqC)
}

func (sm *syncManager) reqLoop() {
	for {
		select {
		case req, ok := <-sm.blockSyncReqC:
			if !ok {
				continue
			}
			if sm.startSyncReq(req) {
				sm.logger.Debugf("receive all response that was met the condition from peer: %d", req.targetPeer)
				continue
			}
			sm.logger.Errorf("No response was received that met the condition from peer:%d", req.targetPeer)
		case <-sm.quitC:
			return
		}
	}
}

func (sm *syncManager) startSyncReq(req *blockSyncReq) bool {
	defer func() {
		atomic.AddUint64(&sm.currReqID, 1)
	}()
	msg := sm.constructReqMsg(req)
	sm.server.signAndSendToPeer(msg, req.targetPeer)
	return true
}

func (sm *syncManager) constructReqMsg(req *blockSyncReq) *chainedbftpb.ConsensusPayload {
	sm.logger.Debugf("server selfIndexInEpoch [%d], got sync req.height:%d:%x to [%v]",
		sm.server.selfIndexInEpoch, req.height, req.blockID, req.targetPeer)
	startHeight := sm.server.chainStore.getCurrentQC().Height
	commitBlock := sm.server.ledgerCache.GetLastCommittedBlock()
	var lockedBlockHash []byte
	if lockedBlock := sm.server.smr.safetyRules.GetLockedBlock(); lockedBlock != nil {
		lockedBlockHash = lockedBlock.Header.BlockHash
	}
	msg := sm.server.constructBlockFetchMsg(sm.currReqID, req.blockID, req.height,
		req.height-startHeight, commitBlock.Header.BlockHash, lockedBlockHash)
	return msg
}
