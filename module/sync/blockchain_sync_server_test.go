/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"testing"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func getNodeStatusReq(t *testing.T) []byte {
	msg := &syncPb.SyncMsg{Type: syncPb.SyncMsg_NODE_STATUS_REQ}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	return bz
}

func getNodeStatusResp(t *testing.T, height uint64) []byte {
	msg := &syncPb.BlockHeightBCM{BlockHeight: height}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_NODE_STATUS_RESP, Payload: bz}
	bz, err = msg2.Marshal()
	require.NoError(t, err)
	return bz
}

func getBlockReq(t *testing.T, height, batchSize uint64) []byte {
	msg := &syncPb.BlockSyncReq{BlockHeight: height, BatchSize: batchSize}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_REQ, Payload: bz}
	bz, err = msg2.Marshal()
	require.NoError(t, err)
	return bz
}

func getBlockResp(t *testing.T, height uint64) []byte {
	msg := &syncPb.SyncBlockBatch{
		Data: &syncPb.SyncBlockBatch_BlockBatch{BlockBatch: &syncPb.BlockBatch{Batches: []*commonPb.Block{
			{Header: &commonPb.BlockHeader{BlockHeight: height}},
		}}},
	}
	bz, err := msg.Marshal()
	require.NoError(t, err)
	msg2 := &syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_RESP, Payload: bz}
	bz, err = msg2.Marshal()
	require.NoError(t, err)
	return bz
}

func initTestSync(t *testing.T) (protocol.SyncService, func()) {
	ctrl := gomock.NewController(t)
	mockNet := newMockNet(ctrl)
	mockMsgBus := newMockMessageBus(ctrl)
	mockVerify := newMockVerifier(ctrl)
	mockStore := newMockBlockChainStore(ctrl)
	mockLedger := newMockLedgerCache(ctrl, &commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 10}})
	mockCommit := newMockCommitter(ctrl, mockLedger)
	sync := NewBlockChainSyncServer("chain1", mockNet, mockMsgBus, mockStore, mockLedger, mockVerify, mockCommit)
	require.NoError(t, sync.Start())
	return sync, func() {
		sync.Stop()
		ctrl.Finish()
	}
}

func TestBlockChainSyncServer_Start(t *testing.T) {
	sync, fn := initTestSync(t)
	defer fn()

	// consume message
	implSync := sync.(*BlockChainSyncServer)
	bz := getNodeStatusResp(t, 110)
	require.NoError(t, implSync.blockSyncMsgHandler("node1", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	bz = getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(3 * time.Millisecond)
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 2, blockStates num: 110, "+
		"pendingBlocks num: 0, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 11, queue num: 0", implSync.processor.getServiceState())

	bz = getBlockResp(t, 11)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 2, blockStates num: 110, pendingBlocks num: 0, receivedBlocks num: 1",
		implSync.scheduler.getServiceState())
	time.Sleep(time.Second)
	require.EqualValues(t, "pendingBlockHeight: 12, queue num: 0", implSync.processor.getServiceState())
}

func TestSyncMsg_NODE_STATUS_REQ(t *testing.T) {
	sync, fn := initTestSync(t)
	defer fn()
	implSync := sync.(*BlockChainSyncServer)

	// 1. req node status
	require.NoError(t, implSync.blockSyncMsgHandler("node1", getNodeStatusReq(t), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 1, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node1]", implSync.net.(*MockNet).sendMsgs[0])

	require.NoError(t, implSync.blockSyncMsgHandler("node2", getNodeStatusReq(t), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 2, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node2]", implSync.net.(*MockNet).sendMsgs[1])
}

func TestSyncBlock_Req(t *testing.T) {
	sync, fn := initTestSync(t)
	defer fn()
	implSync := sync.(*BlockChainSyncServer)

	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 99}}, nil)
	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}}, nil)
	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 101}}, nil)

	require.NoError(t, implSync.blockSyncMsgHandler("node1", getBlockReq(t, 99, 1), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 1, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node1]", implSync.net.(*MockNet).sendMsgs[0])

	require.NoError(t, implSync.blockSyncMsgHandler("node2", getBlockReq(t, 100, 2), netPb.NetMsg_SYNC_BLOCK_MSG))
	//require.EqualValues(t, 3, len(implSync.net.(*MockNet).sendMsgs))
	//require.EqualValues(t, "msgType: 6, to: [node2]", implSync.net.(*MockNet).sendMsgs[1])

	require.Error(t, implSync.blockSyncMsgHandler("node2", getBlockReq(t, 110, 2), netPb.NetMsg_SYNC_BLOCK_MSG))
}

func TestSyncMsg_NODE_STATUS_RESP(t *testing.T) {
	sync, fn := initTestSync(t)
	defer fn()
	implSync := sync.(*BlockChainSyncServer)

	bz := getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(3 * time.Second)
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 1, blockStates num: 110, "+
		"pendingBlocks num: 110, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 11, queue num: 0", implSync.processor.getServiceState())
}

func TestSyncMsg_BLOCK_SYNC_RESP(t *testing.T) {
	sync, fn := initTestSync(t)
	defer fn()
	implSync := sync.(*BlockChainSyncServer)

	// 1. add peer status
	bz := getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))

	// 2. receive block
	bz = getBlockResp(t, 11)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(time.Second * 3)
	require.EqualValues(t, "pendingRecvHeight: 12, peers num: 1, blockStates num: 109, "+
		"pendingBlocks num: 109, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 12, queue num: 0", implSync.processor.getServiceState())
}
