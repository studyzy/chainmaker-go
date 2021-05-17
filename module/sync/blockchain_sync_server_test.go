/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	netPb "chainmaker.org/chainmaker-go/pb/protogo/net"
	syncPb "chainmaker.org/chainmaker-go/pb/protogo/sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/protocol"

	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/require"
)

func getNodeStatusReq(t *testing.T) []byte {
	bz, err := proto.Marshal(&syncPb.SyncMsg{Type: syncPb.SyncMsg_NODE_STATUS_REQ})
	require.NoError(t, err)
	return bz
}

func getNodeStatusResp(t *testing.T, height int64) []byte {
	bz, err := proto.Marshal(&syncPb.BlockHeightBCM{BlockHeight: height})
	require.NoError(t, err)
	bz, err = proto.Marshal(&syncPb.SyncMsg{Type: syncPb.SyncMsg_NODE_STATUS_RESP, Payload: bz})
	require.NoError(t, err)
	return bz
}

func getBlockReq(t *testing.T, height, batchSize int64) []byte {
	bz, err := proto.Marshal(&syncPb.BlockSyncReq{BlockHeight: height, BatchSize: batchSize})
	require.NoError(t, err)
	bz, err = proto.Marshal(&syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_REQ, Payload: bz})
	require.NoError(t, err)
	return bz
}

func getBlockResp(t *testing.T, height int64) []byte {
	bz, err := proto.Marshal(&syncPb.SyncBlockBatch{
		Data: &syncPb.SyncBlockBatch_BlockBatch{BlockBatch: &syncPb.BlockBatch{Batchs: []*commonPb.Block{
			{Header: &commonPb.BlockHeader{BlockHeight: height}},
		}}},
	})
	require.NoError(t, err)
	bz, err = proto.Marshal(&syncPb.SyncMsg{Type: syncPb.SyncMsg_BLOCK_SYNC_RESP, Payload: bz})
	require.NoError(t, err)
	return bz
}

func initTestSync(t *testing.T) protocol.SyncService {
	mockNet := NewMockNet()
	mockStore := NewMockStore()
	mockVerify := NewMockVerifier()
	mockLedger := NewMockLedgerCache(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 10}})
	mockCommit := NewMockCommit(mockLedger)
	sync := NewBlockChainSyncServer("chain1", mockNet, nil, mockStore, mockLedger, mockVerify, mockCommit)
	err := sync.Start()
	require.NoError(t, err)

	return sync
}

func TestBlockChainSyncServer_Start(t *testing.T) {
	sync := initTestSync(t)
	defer sync.Stop()

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
	sync := initTestSync(t)
	defer sync.Stop()
	implSync := sync.(*BlockChainSyncServer)

	// 1. req node status
	require.NoError(t, implSync.blockSyncMsgHandler("node1", getNodeStatusReq(t), netPb.NetMsg_SYNC_BLOCK_MSG))
	require.EqualValues(t, 1, len(implSync.net.(*MockNet).sendMsgs))
	require.EqualValues(t, "msgType: 6, to: [node1]", implSync.net.(*MockNet).sendMsgs[0])

	require.NoError(t, implSync.blockSyncMsgHandler("node2", getNodeStatusReq(t), netPb.NetMsg_SYNC_BLOCK_MSG))
	require.EqualValues(t, 2, len(implSync.net.(*MockNet).sendMsgs))
	require.EqualValues(t, "msgType: 6, to: [node2]", implSync.net.(*MockNet).sendMsgs[1])
}

func TestSyncBlock_Req(t *testing.T) {
	sync := initTestSync(t)
	defer sync.Stop()
	implSync := sync.(*BlockChainSyncServer)

	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 99}}, nil)
	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 100}}, nil)
	_ = implSync.blockChainStore.PutBlock(&commonPb.Block{Header: &commonPb.BlockHeader{BlockHeight: 101}}, nil)

	require.NoError(t, implSync.blockSyncMsgHandler("node1", getBlockReq(t, 99, 1), netPb.NetMsg_SYNC_BLOCK_MSG))
	require.EqualValues(t, 1, len(implSync.net.(*MockNet).sendMsgs))
	require.EqualValues(t, "msgType: 6, to: [node1]", implSync.net.(*MockNet).sendMsgs[0])

	require.NoError(t, implSync.blockSyncMsgHandler("node2", getBlockReq(t, 100, 2), netPb.NetMsg_SYNC_BLOCK_MSG))
	require.EqualValues(t, 3, len(implSync.net.(*MockNet).sendMsgs))
	require.EqualValues(t, "msgType: 6, to: [node2]", implSync.net.(*MockNet).sendMsgs[1])

	require.Error(t, implSync.blockSyncMsgHandler("node2", getBlockReq(t, 110, 2), netPb.NetMsg_SYNC_BLOCK_MSG))
}

func TestSyncMsg_NODE_STATUS_RESP(t *testing.T) {
	sync := initTestSync(t)
	defer sync.Stop()
	implSync := sync.(*BlockChainSyncServer)

	bz := getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	time.Sleep(3 * time.Millisecond)
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 1, blockStates num: 110, "+
		"pendingBlocks num: 0, receivedBlocks num: 0", implSync.scheduler.getServiceState())
	require.EqualValues(t, "pendingBlockHeight: 11, queue num: 0", implSync.processor.getServiceState())
}

func TestSyncMsg_BLOCK_SYNC_RESP(t *testing.T) {
	sync := initTestSync(t)
	defer sync.Stop()
	implSync := sync.(*BlockChainSyncServer)

	// 1. add peer status
	bz := getNodeStatusResp(t, 120)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))

	// 2. receive block
	bz = getBlockResp(t, 11)
	require.NoError(t, implSync.blockSyncMsgHandler("node2", bz, netPb.NetMsg_SYNC_BLOCK_MSG))
	require.EqualValues(t, "pendingRecvHeight: 11, peers num: 1, blockStates num: 110, "+
		"pendingBlocks num: 0, receivedBlocks num: 1", implSync.scheduler.getServiceState())
	time.Sleep(time.Second)
	require.EqualValues(t, "pendingBlockHeight: 12, queue num: 0", implSync.processor.getServiceState())
}
