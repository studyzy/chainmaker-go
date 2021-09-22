/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"testing"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNodeStatusMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}})
	sch := newScheduler(NewMockSender(), mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. the peer status is old
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 90}})
	require.EqualValues(t, 90, sch.peers["node1"])
	require.EqualValues(t, 0, len(sch.blockStates))
	require.EqualValues(t, 101, sch.pendingRecvHeight)

	// 2. the peer status is new
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 150}})
	require.EqualValues(t, 50, len(sch.blockStates))
	require.EqualValues(t, 150, sch.peers["node1"])
	require.EqualValues(t, 101, sch.pendingRecvHeight)

	// 3. modify the ledger status
	mockLedger.SetLastCommittedBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 180}})

	// 4. receive the peer status is old, and update the pendingRecvHeight
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 151}})
	require.EqualValues(t, 151, sch.peers["node1"])
	require.EqualValues(t, 101, sch.pendingRecvHeight)

	// 5. malicious node to broadcast old status
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 90}})
	require.EqualValues(t, 0, len(sch.peers))

	// 6. receive another peer status
	_, _ = sch.handler(NodeStatusMsg{from: "node2", msg: syncPb.BlockHeightBCM{BlockHeight: 100}})
	require.EqualValues(t, 100, sch.peers["node2"])

	// 7. repeat receive same peer status
	_, _ = sch.handler(NodeStatusMsg{from: "node2", msg: syncPb.BlockHeightBCM{BlockHeight: 100}})
	require.EqualValues(t, 100, sch.peers["node2"])

	// 8. fired dataDetection task
	_, _ = sch.handler(DataDetection{})
	require.EqualValues(t, 181, sch.pendingRecvHeight)

}

func TestNextHeightToReq(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}})
	sch := newScheduler(NewMockSender(), mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. add block status
	for i := uint64(0); i < 5; i++ {
		sch.blockStates[100+i] = newBlock
	}

	// 2. get the next request block height when min < pendingRecvHeight
	require.EqualValues(t, -1, sch.nextHeightToReq())
	require.EqualValues(t, 4, len(sch.blockStates))

	// 3. get the next request block height when min > pendingRecvHeight
	mockLedger.SetLastCommittedBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 90}})
	require.EqualValues(t, 101, sch.nextHeightToReq())

	// 4. modify pendingRecvHeight and get the next request block height when min == pendingRecvHeight
	sch.pendingRecvHeight = 101
	require.EqualValues(t, 101, sch.nextHeightToReq())
	require.EqualValues(t, 4, len(sch.blockStates))
}

func TestIsNeedSync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}})
	sch := newScheduler(NewMockSender(), mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. add peer status
	sch.peers["node1"] = 110
	sch.peers["node2"] = 90

	// 2. peer max height = 110, node's height = 100
	require.True(t, sch.isNeedSync())

	// 3. modify peer status to old and check result
	sch.peers["node1"] = 80
	require.False(t, sch.isNeedSync())

	// 4. modify peer status to the next neighbour height with node's
	sch.peers["node1"] = 101
	sch.lastRequest = time.Now()
	require.False(t, sch.isNeedSync())

	time.Sleep(3 * time.Second)
	require.True(t, sch.isNeedSync())
}

func TestSchedulerMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSender := NewMockSender()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}})
	sch := newScheduler(mockSender, mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. add peer status
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 151}})

	// 2. try scheduler
	_, _ = sch.handler(SchedulerMsg{})

	// 3. check status in mockSender
	require.EqualValues(t, 2, len(sch.pendingTime))
	require.EqualValues(t, 2, len(sch.pendingBlocks))
	require.EqualValues(t, "msgType: 2, to: node1", mockSender.msgs[0])
}

func TestSyncedBlockMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 5}})
	sch := newScheduler(NewMockSender(), mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	bz, _ := proto.Marshal(&syncPb.SyncBlockBatch{
		Data: &syncPb.SyncBlockBatch_BlockBatch{BlockBatch: &syncPb.BlockBatch{Batches: []*commonPb.Block{
			{Header: &commonPb.BlockHeader{BlockHeight: 11}},
			{Header: &commonPb.BlockHeader{BlockHeight: 12}},
		}}},
	})

	// 1. add syncedBlockMsg and check process result
	ret, _ := sch.handler(&SyncedBlockMsg{msg: bz, from: "node1"})
	require.Nil(t, ret)

	// 2. add peer status
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 151}})
	ret, _ = sch.handler(&SyncedBlockMsg{msg: bz, from: "node1"})
	require.NotNil(t, ret)
	require.EqualValues(t, "node1", ret.(*ReceivedBlocks).from)
}

func TestProcessedBlockResp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 5}})
	sch := newScheduler(NewMockSender(), mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. add ok process result and check result
	_, err := sch.handler(ProcessedBlockResp{height: 6, status: ok, from: "node1"})
	require.NoError(t, err)
	require.EqualValues(t, 0, len(sch.blockStates))
	require.EqualValues(t, 7, sch.pendingRecvHeight)

	// 2. add process result and check result
	_, err = sch.handler(ProcessedBlockResp{height: 7, status: hasProcessed, from: "node1"})
	require.NoError(t, err)
	require.EqualValues(t, 0, len(sch.blockStates))
	require.EqualValues(t, 8, sch.pendingRecvHeight)

	// 3. add failed process result and check result
	_, err = sch.handler(ProcessedBlockResp{height: 7, status: validateFailed, from: "node1"})
	require.NoError(t, err)
	require.EqualValues(t, 1, len(sch.blockStates))
	require.EqualValues(t, 8, sch.pendingRecvHeight)
}

func TestLivenessMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 5}})
	sch := newScheduler(NewMockSender(), mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. no any status
	_, _ = sch.handler(LivenessMsg{})

	// 2. modify status
	sch.pendingTime[sch.pendingRecvHeight] = time.Now()
	sch.blockStates[sch.pendingRecvHeight] = pendingBlock

	// 3. the request did not time out
	_, _ = sch.handler(LivenessMsg{})
	require.EqualValues(t, pendingBlock, sch.blockStates[sch.pendingRecvHeight])

	// 4. time out with the request
	time.Sleep(sch.peerReqTimeout)
	_, _ = sch.handler(LivenessMsg{})
	require.EqualValues(t, 0, len(sch.pendingBlocks))
	require.EqualValues(t, newBlock, sch.blockStates[sch.pendingRecvHeight])
}

func TestSchedulerFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSender := NewMockSender()
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 10}})
	sch := newScheduler(mockSender, mockLedger, 100, time.Second, time.Second*3, 2, logger.GetLogger(logger.MODULE_SYNC))

	// 1. add peers status
	_, _ = sch.handler(NodeStatusMsg{from: "node1", msg: syncPb.BlockHeightBCM{BlockHeight: 100}})
	require.EqualValues(t, 1, len(sch.peers))
	require.EqualValues(t, 90, len(sch.blockStates))
	_, _ = sch.handler(NodeStatusMsg{from: "node2", msg: syncPb.BlockHeightBCM{BlockHeight: 1000}})
	require.EqualValues(t, 2, len(sch.peers))
	require.EqualValues(t, 100, len(sch.blockStates))
	for _, state := range sch.blockStates {
		require.EqualValues(t, newBlock, state)
	}

	// 2. check next height to request
	require.EqualValues(t, 11, sch.nextHeightToReq())

	// 3. try scheduler and check status
	_, _ = sch.handler(SchedulerMsg{})
	require.EqualValues(t, pendingBlock, sch.blockStates[11])
	require.EqualValues(t, pendingBlock, sch.blockStates[12])

	// 4. process received blocks msg and check result
	bz, err := proto.Marshal(&syncPb.SyncBlockBatch{
		Data: &syncPb.SyncBlockBatch_BlockBatch{BlockBatch: &syncPb.BlockBatch{Batches: []*commonPb.Block{
			{Header: &commonPb.BlockHeader{BlockHeight: 11}},
			{Header: &commonPb.BlockHeader{BlockHeight: 12}},
		}}},
	})
	require.NoError(t, err)
	_, err = sch.handler(&SyncedBlockMsg{from: "node1", msg: bz})
	require.NoError(t, err)
	require.EqualValues(t, receivedBlock, sch.blockStates[11])
	require.EqualValues(t, receivedBlock, sch.blockStates[12])

	// 5. process block resp in handle
	_, err = sch.handler(ProcessedBlockResp{from: "node1", status: ok, height: 11})
	require.NoError(t, err)
	require.EqualValues(t, 99, len(sch.blockStates))
	_, err = sch.handler(ProcessedBlockResp{from: "node1", status: ok, height: 12})
	require.NoError(t, err)
	require.EqualValues(t, 98, len(sch.blockStates))
}
