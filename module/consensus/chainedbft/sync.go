/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"bytes"
	"sort"
	"sync/atomic"
	"time"

	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	timeservice "chainmaker.org/chainmaker-go/consensus/chainedbft/time_service"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/governance"
	"chainmaker.org/chainmaker-go/logger"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/golang/protobuf/proto"
)

const (
	MaxSyncBlockNum = 10
)

type orderBlocks []*chainedbftpb.BlockPair

//Len returns the size of orderBlocks
func (ob orderBlocks) Len() int { return len(ob) }

//Swap swaps the ith object with jth object in orderBlocks
func (ob orderBlocks) Swap(i, j int) { ob[i], ob[j] = ob[j], ob[i] }

//Less checks the ith object's level < the jth object's level
func (ob orderBlocks) Less(i, j int) bool { return ob[i].QC.Level < ob[j].QC.Level }

//blockSyncReq defines a block sync request
type blockSyncReq struct {
	height      uint64 // Height of QC received from other nodes
	blockID     []byte // BlockID of QC received from other nodes
	targetPeer  uint64 // The identity of the requested node
	startLevel  uint64 // Data for the next level required by the local node
	targetLevel uint64 // Level of QC received from other nodes
}

//syncMsg defines a sync msg from remote peer
type syncMsg struct {
	fromPeer uint64
	msg      *chainedbftpb.ConsensusPayload
}

// syncManager Synchronize block data from other peers
type syncManager struct {
	currReqID     uint64 // The ID of the current request
	nextReqHeight int64  // lack of start level now
	targetHeight  uint64 // need to fetch level

	quitC         chan struct{}
	reqDone       chan bool          // responsible for synchronization with the consensus main process;
	syncDone      chan struct{}      // responsible for synchronization of request and reply processes
	syncMsgC      chan *syncMsg      // receive resp from server
	blockSyncReqC chan *blockSyncReq // receive req from local node

	logger *logger.CMLogger
	server *ConsensusChainedBftImpl
}

func newSyncManager(server *ConsensusChainedBftImpl) *syncManager {
	return &syncManager{
		currReqID:     1,
		nextReqHeight: 1,
		quitC:         make(chan struct{}),
		reqDone:       make(chan bool),
		syncDone:      make(chan struct{}),
		syncMsgC:      make(chan *syncMsg, 256),
		blockSyncReqC: make(chan *blockSyncReq),

		server: server,
		logger: server.logger,
	}
}

func (sm *syncManager) start() {
	go sm.reqLoop()
	go sm.respLoop()
}

func (sm *syncManager) stop() {
	close(sm.quitC)
	close(sm.blockSyncReqC)
	close(sm.syncMsgC)
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
	t := time.NewTimer(timeservice.RoundTimeout * 2)
	defer func() {
		t.Stop()
		atomic.AddUint64(&sm.currReqID, 1)
	}()

	msg := sm.constructReqMsg(req)
	sm.server.signAndSendToPeer(msg, req.targetPeer)
	select {
	case <-t.C:
		sm.reqDone <- false
		return false
	case <-sm.syncDone:
		sm.reqDone <- true
		return true
	case <-sm.quitC:
		return true
	}
}

func (sm *syncManager) respLoop() {
	for {
		select {
		case <-sm.quitC:
			return
		case syncMsg, ok := <-sm.syncMsgC:
			if !ok {
				continue
			}
			respID := syncMsg.msg.GetBlockFetchRespMsg().RespID
			lastRecvHeight := sm.processBlocks(syncMsg)
			if atomic.LoadUint64(&sm.currReqID) == respID && atomic.LoadUint64(&sm.targetHeight) == lastRecvHeight {
				sm.syncDone <- struct{}{}
			}
		}
	}
}

//new peerSyncer and start a go coroutine
func (sm *syncManager) constructReqMsg(req *blockSyncReq) *chainedbftpb.ConsensusPayload {
	sm.logger.Debugf("server selfIndexInEpoch [%d], got sync req.height:%d:%x to [%v]",
		sm.server.selfIndexInEpoch, req.height, req.blockID, req.targetPeer)
	atomic.StoreUint64(&sm.targetHeight, req.height)
	startHeight := sm.server.chainStore.getCurrentQC().Height
	msg := sm.server.constructBlockFetchMsg(sm.currReqID, req.blockID, req.height, req.height-startHeight)
	return msg
}

func (sm *syncManager) processBlocks(msg *syncMsg) uint64 {
	blockFetchMsg := msg.msg.GetBlockFetchRespMsg()
	sm.logger.Infof("server selfIndexInEpoch [%d] processBlocks, status: %s, count: %d, authorIdx: %d",
		sm.server.selfIndexInEpoch, blockFetchMsg.Status, len(blockFetchMsg.Blocks), blockFetchMsg.AuthorIdx)
	if blockFetchMsg.Status != chainedbftpb.BlockFetchStatus_Succeeded {
		return 0
	}

	var (
		blockPairs      = blockFetchMsg.Blocks
		blocks          = make(map[string]bool, len(blockPairs))
		lastBlockHeight uint64
	)
	sort.Sort(orderBlocks(blockPairs))
	for _, blockPair := range blockPairs {
		sm.logger.Debugf("server selfIndexInEpoch [%v] process block [%v:%v]", sm.server.selfIndexInEpoch,
			blockPair.QC.Height, blockPair.QC.Level)
		if !sm.validateBlockPair(msg.fromPeer, blockPair, blocks) {
			return lastBlockHeight
		}
		if !sm.insertBlockAndQC(msg.fromPeer, blockPair) {
			return lastBlockHeight
		}
		sm.nextReqHeight = blockPair.Block.Header.BlockHeight + 1
		lastBlockHeight = uint64(blockPair.Block.Header.BlockHeight)
	}
	sm.logger.Debugf("process block height: %d", lastBlockHeight)
	return lastBlockHeight
}

func (sm *syncManager) validateBlockPair(fromPeer uint64, blockPair *chainedbftpb.BlockPair, seeBlocks map[string]bool) bool {
	qc := blockPair.QC
	header := blockPair.Block.GetHeader()
	if qc.EpochId != sm.server.smr.getEpochId() {
		sm.logger.Errorf("server selfIndexInEpoch [%v] qc epochId err %v,expected %v ", sm.server.selfIndexInEpoch,
			qc.EpochId, sm.server.smr.getEpochId())
		return false
	}
	if err := sm.server.verifyJustifyQC(qc); err != nil {
		sm.logger.Errorf("server selfIndexInEpoch [%v] verify qc [%v:%v] failed, err %v", sm.server.selfIndexInEpoch,
			qc.Height, qc.Level, err)
		return false
	}
	if qc.Height != uint64(header.GetBlockHeight()) {
		sm.logger.Errorf("server selfIndexInEpoch [%v] mismatch block height or level [%v], expected [%v:%v]",
			sm.server.selfIndexInEpoch, header.GetBlockHeight(), qc.Height, qc.Level)
		return false
	}
	if bytes.Compare(header.GetBlockHash(), qc.BlockID) != 0 {
		sm.logger.Errorf("server selfIndexInEpoch [%v] mismatch block id [%v], expected [%v]",
			qc.BlockID, header.GetBlockHash())
		return false
	}
	blockID := string(header.GetBlockHash())
	if exist := seeBlocks[(blockID)]; exist {
		sm.logger.Errorf("duplicated block [%v:%v] id [%v]", qc.Height, qc.Level, blockID)
		return false
	}
	seeBlocks[blockID] = true

	if err := sm.server.blockVerifier.VerifyBlock(blockPair.Block, protocol.CONSENSUS_VERIFY); err == commonErrors.ErrBlockHadBeenCommited {
		hadCommitBlock, getBlockErr := sm.server.store.GetBlock(header.GetBlockHeight())
		if getBlockErr != nil {
			sm.logger.Errorf("service selfIndexInEpoch [%v] VerifyBlock, block had been committed, "+
				"get block err, %v", sm.server.selfIndexInEpoch, getBlockErr)
			return false
		}
		if !bytes.Equal(hadCommitBlock.GetHeader().GetBlockHash(), header.GetBlockHash()) {
			sm.logger.Errorf("service selfIndexInEpoch [%v] VerifyBlock, commit block failed, block had been "+
				"committed, hash unequal err, %v", sm.server.selfIndexInEpoch, getBlockErr)
			return false
		}
	} else if err != nil {
		sm.logger.Errorf("service selfIndexInEpoch [%v] VerifyBlock failed: invalid block "+
			"from peer [%v] at height [%v] level [%v], err %v", sm.server.selfIndexInEpoch,
			fromPeer, header.GetBlockHeight(), qc.Level, err)
		return false
	}
	return true
}

func (sm *syncManager) insertBlockAndQC(fromPeer uint64, blockPair *chainedbftpb.BlockPair) bool {
	qc := blockPair.QC
	header := blockPair.Block.GetHeader()

	consensusArgs, err := utils.GetConsensusArgsFromBlock(blockPair.Block)
	if err != nil {
		sm.logger.Errorf("service selfIndexInEpoch %v GetConsensusArgsFromBlock err: from peer %v "+
			"at height %v level %v, err %v", sm.server.selfIndexInEpoch, fromPeer, header.GetBlockHeight(), qc.Level, err)
		return false
	}
	txRWSet, err := governance.CheckAndCreateGovernmentArgs(blockPair.Block, sm.server.store, sm.server.proposalCache, sm.server.ledgerCache)
	if err != nil {
		sm.logger.Errorf("service selfIndexInEpoch %v CheckAndCreateGovernmentArgs err: from peer %v at "+
			"height %v level %v, err %v", sm.server.selfIndexInEpoch, fromPeer, header.GetBlockHeight(), qc.Level, err)
		return false
	}

	txRWSetBytes, _ := proto.Marshal(txRWSet)
	ConsensusDataBytes, _ := proto.Marshal(consensusArgs.ConsensusData)
	if !bytes.Equal(txRWSetBytes, ConsensusDataBytes) {
		sm.logger.Errorf("service selfIndexInEpoch %v processProposal: invalid Consensus Args from proposer"+
			" %v at height %v level %v, err %v", sm.server.selfIndexInEpoch, fromPeer, header.GetBlockHeight(), qc.Level, err)
		return false
	}
	if executorErr := sm.server.chainStore.insertBlock(blockPair.Block); executorErr != nil {
		sm.logger.Errorf("service selfIndexInEpoch [%v] insertBlock: execute a block [%v] failed, err %v",
			sm.server.selfIndexInEpoch, header.GetBlockHeight(), executorErr)
		return false
	}
	if executorErr := sm.server.chainStore.insertQC(qc); executorErr != nil {
		sm.logger.Errorf("service selfIndexInEpoch [%v] insertQC: execute a qc [%v] failed, err %v",
			sm.server.selfIndexInEpoch, header.GetBlockHeight(), executorErr)
		return false
	}
	sm.server.smr.updateLockedQC(qc)
	sm.server.commitBlocksByQC(qc)
	sm.server.processCertificates(qc, nil)
	return true
}
