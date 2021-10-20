/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"math"
	"sort"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/Workiva/go-datastructures/queue"
	"github.com/gogo/protobuf/proto"
)

type syncSender interface {
	broadcastMsg(msgType syncPb.SyncMsg_MsgType, msg []byte) error
	sendMsg(msgType syncPb.SyncMsg_MsgType, msg []byte, to string) error
}

// scheduler Retrieves block data of specified height from different nodes
type scheduler struct {
	peers             map[string]uint64     // The state of the connected nodes
	blockStates       map[uint64]blockState // Block state for each height (New, Pending, Received)
	pendingTime       map[uint64]time.Time  // The time to send a request for a block data of the specified height
	pendingBlocks     map[uint64]string     // Which the block data of the specified height is being fetched from the node
	receivedBlocks    map[uint64]string     // Block data has been received from the node
	lastRequest       time.Time             // The last time which block request was sent
	pendingRecvHeight uint64                // The next block to be processed, all smaller blocks have been processed

	maxPendingBlocks uint64 // The maximum number of blocks allowed to be processed simultaneously
	// (including: New, Pending, Received);
	BatchesizeInEachReq uint64        // Number of blocks requested per request
	peerReqTimeout      time.Duration // The maximum timeout for a node response
	reqTimeThreshold    time.Duration // When the difference between the height of the node and
	// the latest height of peers is 1, the time interval for requesting

	log    *logger.CMLogger
	sender syncSender
	ledger protocol.LedgerCache
}

func newScheduler(sender syncSender, ledger protocol.LedgerCache,
	maxNum uint64, timeOut, reqTimeThreshold time.Duration, batchesize uint64, log *logger.CMLogger) *scheduler {

	currHeight, err := ledger.CurrentHeight()
	if err != nil {
		return nil
	}
	return &scheduler{
		log:    log,
		ledger: ledger,
		sender: sender,

		peerReqTimeout:      timeOut,
		maxPendingBlocks:    maxNum,
		BatchesizeInEachReq: batchesize,
		reqTimeThreshold:    reqTimeThreshold,

		peers:             make(map[string]uint64),
		blockStates:       make(map[uint64]blockState),
		pendingBlocks:     make(map[uint64]string),
		pendingTime:       make(map[uint64]time.Time),
		receivedBlocks:    make(map[uint64]string),
		pendingRecvHeight: currHeight + 1,
	}
}

func (sch *scheduler) handler(event queue.Item) (queue.Item, error) {
	switch msg := event.(type) {
	case NodeStatusMsg:
		sch.handleNodeStatus(msg)
	case LivenessMsg:
		sch.handleLivinessMsg()
	case SchedulerMsg:
		return sch.handleScheduleMsg()
	case *SyncedBlockMsg:
		return sch.handleSyncedBlockMsg(msg)
	case ProcessedBlockResp:
		return sch.handleProcessedBlockResp(msg)
	case DataDetection:
		sch.handleDataDetection()
	}
	return nil, nil
}

func (sch *scheduler) handleNodeStatus(msg NodeStatusMsg) {
	localCurrBlk := sch.ledger.GetLastCommittedBlock()
	if old, exist := sch.peers[msg.from]; exist {
		if old > msg.msg.BlockHeight || sch.isPeerArchivedTooHeight(localCurrBlk.Header.BlockHeight,
			msg.msg.GetArchivedHeight()) {
			delete(sch.peers, msg.from)
			return
		}
	}
	if sch.isPeerArchivedTooHeight(localCurrBlk.Header.BlockHeight, msg.msg.GetArchivedHeight()) {
		sch.log.Debugf("coming node[%s], status[height: %d, archivedHeight: %d], archived too height to sync, will ignore it",
			msg.from, msg.msg.BlockHeight, msg.msg.GetArchivedHeight())
		return
	}
	sch.log.Debugf("add node[%s], status[height: %d, archivedHeight: %d]", msg.from, msg.msg.BlockHeight,
		msg.msg.ArchivedHeight)
	sch.peers[msg.from] = msg.msg.BlockHeight
	sch.addPendingBlocksAndUpdatePendingHeight(msg.msg.BlockHeight)
}

func (sch *scheduler) addPendingBlocksAndUpdatePendingHeight(peerHeight uint64) {
	if uint64(len(sch.blockStates)) > sch.maxPendingBlocks {
		return
	}
	blk := sch.ledger.GetLastCommittedBlock()
	if blk.Header.BlockHeight >= peerHeight {
		return
	}
	for i := sch.pendingRecvHeight; i <= peerHeight && i < sch.pendingRecvHeight+sch.maxPendingBlocks; i++ {
		if _, exist := sch.blockStates[i]; !exist {
			sch.blockStates[i] = newBlock
		}
	}
}

func (sch *scheduler) handleDataDetection() {
	blk := sch.ledger.GetLastCommittedBlock()
	for height := range sch.blockStates {
		if height < blk.Header.BlockHeight {
			delete(sch.blockStates, height)
			delete(sch.pendingBlocks, height)
			delete(sch.receivedBlocks, height)
			delete(sch.pendingTime, height)
			delete(sch.receivedBlocks, height)
		}
	}
	sch.pendingRecvHeight = blk.Header.BlockHeight + 1
	sch.blockStates[sch.pendingRecvHeight] = newBlock
}

func (sch *scheduler) handleLivinessMsg() {
	reqTime, exist := sch.pendingTime[sch.pendingRecvHeight]
	if exist && time.Since(reqTime) > sch.peerReqTimeout {
		id := sch.pendingBlocks[sch.pendingRecvHeight]
		sch.log.Debugf("block request [height: %d] time out from node[%s]", sch.pendingRecvHeight, id)
		if currBlk := sch.ledger.GetLastCommittedBlock(); currBlk != nil &&
			currBlk.Header.BlockHeight < sch.pendingRecvHeight {
			sch.blockStates[sch.pendingRecvHeight] = newBlock
		}
		delete(sch.peers, id)
		delete(sch.pendingTime, sch.pendingRecvHeight)
		delete(sch.pendingBlocks, sch.pendingRecvHeight)
	}
}

func (sch *scheduler) handleScheduleMsg() (queue.Item, error) {
	var (
		err           error
		bz            []byte
		peer          string
		pendingHeight uint64
	)

	if !sch.isNeedSync() {
		//sch.log.Debugf("no need to sync block")
		return nil, nil
	}
	if pendingHeight = sch.nextHeightToReq(); pendingHeight == math.MaxUint64 {
		sch.log.Debugf("pendingHeight: %d, block status %v", pendingHeight, sch.blockStates)
		return nil, nil
	}
	if bz, err = proto.Marshal(&syncPb.BlockSyncReq{
		BlockHeight: pendingHeight, BatchSize: sch.BatchesizeInEachReq,
	}); err != nil {
		return nil, err
	}

	if peer = sch.selectPeer(pendingHeight); len(peer) == 0 {
		sch.log.Debugf("no peers have block [%d] ", pendingHeight)
		return nil, nil
	}
	sch.lastRequest = time.Now()
	for i := pendingHeight; i <= sch.peers[peer] && i < sch.BatchesizeInEachReq+pendingHeight; i++ {
		sch.blockStates[i] = pendingBlock
		sch.pendingTime[i] = sch.lastRequest
		sch.pendingBlocks[i] = peer
	}
	sch.log.Debugf("request block[height: %d] from node [%s], BatchesSizeInReq: %d", pendingHeight, peer,
		sch.BatchesizeInEachReq)
	if err := sch.sender.sendMsg(syncPb.SyncMsg_BLOCK_SYNC_REQ, bz, peer); err != nil {
		return nil, err
	}
	return nil, nil
}

func (sch *scheduler) nextHeightToReq() uint64 {
	var min uint64 = math.MaxUint64
	for height, status := range sch.blockStates {
		if min > height && status == newBlock {
			min = height
		}
	}
	if min == math.MaxUint64 || min < sch.pendingRecvHeight {
		delete(sch.blockStates, min)
		return math.MaxUint64
	}
	return min
}

func (sch *scheduler) maxHeight() uint64 {
	var max uint64
	for _, height := range sch.peers {
		if max < height {
			max = height
		}
	}
	return max
}

func (sch *scheduler) isNeedSync() bool {
	currHeight, err := sch.ledger.CurrentHeight()
	if err != nil {
		panic(err)
	}
	max := sch.maxHeight()
	// The reason for the interval of 1 block is that the block to
	// be synchronized is being processed by the consensus module.
	return currHeight+1 < max || (currHeight+1 == max && time.Since(sch.lastRequest) > sch.reqTimeThreshold)
}

func (sch *scheduler) selectPeer(pendingHeight uint64) string {
	peers := sch.getHeight(pendingHeight)
	if len(peers) == 0 {
		return ""
	}

	pendingReqInPeers := make(map[int][]string)
	for i := 0; i < len(peers); i++ {
		reqNum := sch.getPendingReqInPeer(peers[i])
		pendingReqInPeers[reqNum] = append(pendingReqInPeers[reqNum], peers[i])
	}
	min := math.MaxInt64
	for num := range pendingReqInPeers {
		if min > num {
			min = num
		}
	}
	peers = pendingReqInPeers[min]
	sort.Strings(peers)
	return peers[0]
}

func (sch *scheduler) getHeight(pendingHeight uint64) []string {
	peers := make([]string, 0, len(sch.peers)/2)
	for id, height := range sch.peers {
		if height >= pendingHeight {
			peers = append(peers, id)
		}
	}
	return peers
}

func (sch *scheduler) getPendingReqInPeer(peer string) int {
	num := 0
	for _, id := range sch.pendingBlocks {
		if id == peer {
			num++
		}
	}
	return num
}

func (sch *scheduler) handleSyncedBlockMsg(msg *SyncedBlockMsg) (queue.Item, error) {
	blkBatch := syncPb.SyncBlockBatch{}
	if err := proto.Unmarshal(msg.msg, &blkBatch); err != nil {
		return nil, err
	}
	if len(blkBatch.GetBlockBatch().Batches) == 0 {
		return nil, nil
	}
	needToProcess := false
	for _, blk := range blkBatch.GetBlockBatch().Batches {
		delete(sch.pendingBlocks, blk.Header.BlockHeight)
		delete(sch.pendingTime, blk.Header.BlockHeight)
		if _, exist := sch.blockStates[blk.Header.BlockHeight]; exist {
			needToProcess = true
			sch.blockStates[blk.Header.BlockHeight] = receivedBlock
			sch.receivedBlocks[blk.Header.BlockHeight] = msg.from
		}
		sch.log.Debugf("received block [height:%d:%x] needToProcess: %v from "+
			"node [%s]", blk.Header.BlockHeight, blk.Header.BlockHash, needToProcess, msg.from)
	}
	if needToProcess {
		return &ReceivedBlocks{
			blks: blkBatch.GetBlockBatch().Batches,
			from: msg.from}, nil
	}
	return nil, nil
}

func (sch *scheduler) handleProcessedBlockResp(msg ProcessedBlockResp) (queue.Item, error) {
	sch.log.Debugf("process block [height:%d] status[%d] from node"+
		" [%s], pendingHeight: %d", msg.height, msg.status, msg.from, sch.pendingRecvHeight)
	delete(sch.receivedBlocks, msg.height)
	if msg.status == ok || msg.status == hasProcessed {
		delete(sch.blockStates, msg.height)
		if msg.height >= sch.pendingRecvHeight {
			sch.pendingRecvHeight = msg.height + 1
			sch.log.Debugf("increase pendingBlockHeight: %d", sch.pendingRecvHeight)
		}
	}
	if msg.status == validateFailed {
		sch.blockStates[msg.height] = newBlock
		delete(sch.peers, msg.from)
	}
	if msg.status == dbErr {
		return nil, fmt.Errorf("query db failed in processor")
	}
	if msg.status == addErr {
		sch.blockStates[msg.height] = newBlock
		delete(sch.peers, msg.from)
		return nil, fmt.Errorf("failed add block to chain")
	}
	return nil, nil
}

func (sch *scheduler) getServiceState() string {
	return fmt.Sprintf("pendingRecvHeight: %d, peers num: %d, blockStates num: %d, "+
		"pendingBlocks num: %d, receivedBlocks num: %d", sch.pendingRecvHeight, len(sch.peers), len(sch.blockStates),
		len(sch.pendingBlocks), len(sch.receivedBlocks))
}

func (sch *scheduler) isPeerArchivedTooHeight(localHeight, peerArchivedHeight uint64) bool {
	return peerArchivedHeight != 0 && localHeight <= peerArchivedHeight
}
