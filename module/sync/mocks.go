/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"chainmaker.org/chainmaker/common/msgbus"
	mbusmock "chainmaker.org/chainmaker/common/msgbus/mock"
	"chainmaker.org/chainmaker/protocol/mock"
	"fmt"
	"github.com/golang/mock/gomock"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	netPb "chainmaker.org/chainmaker/pb-go/net"
	syncPb "chainmaker.org/chainmaker/pb-go/sync"

	"chainmaker.org/chainmaker/protocol"
)

var errStr = "implement me"

type netMsg struct {
	msgType netPb.NetMsg_MsgType
	bz      []byte
}

type MockSender struct {
	msgs []string
}

func NewMockSender() *MockSender {
	return &MockSender{}
}

func (m MockSender) broadcastMsg(msgType syncPb.SyncMsg_MsgType, msg []byte) error {
	panic(errStr)
}

func (m *MockSender) sendMsg(msgType syncPb.SyncMsg_MsgType, msg []byte, to string) error {
	m.msgs = append(m.msgs, fmt.Sprintf("msgType: %d, to: %s", msgType, to))
	return nil
}

type MockVerifyAndCommit struct {
	cache       protocol.LedgerCache
	receiveItem []*commonPb.Block
}

func NewMockVerifyAndCommit(cache protocol.LedgerCache) *MockVerifyAndCommit {
	return &MockVerifyAndCommit{cache: cache}
}

func (m *MockVerifyAndCommit) validateAndCommitBlock(block *commonPb.Block) processedBlockStatus {
	m.receiveItem = append(m.receiveItem, block)
	m.cache.SetLastCommittedBlock(block)
	return ok
}

func newMockLedgerCache(ctrl *gomock.Controller, blk *commonPb.Block) protocol.LedgerCache {
	mockLedger := mock.NewMockLedgerCache(ctrl)
	lastCommitBlk := blk
	mockLedger.EXPECT().GetLastCommittedBlock().DoAndReturn(func() *commonPb.Block {
		return lastCommitBlk
	}).AnyTimes()
	mockLedger.EXPECT().CurrentHeight().DoAndReturn(func() (int64, error) {
		return lastCommitBlk.Header.BlockHeight, nil
	}).AnyTimes()
	mockLedger.EXPECT().SetLastCommittedBlock(gomock.Any()).DoAndReturn(func(blk *commonPb.Block) {
		lastCommitBlk = blk
	}).AnyTimes()
	return mockLedger
}

func newMockMessageBus(ctrl *gomock.Controller) msgbus.MessageBus {
	mockMsgBus := mbusmock.NewMockMessageBus(ctrl)
	mockMsgBus.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	return mockMsgBus
}

func newMockNet(ctrl *gomock.Controller) protocol.NetService {
	mockNet := mock.NewMockNetService(ctrl)
	broadcastMsgs := make([]netMsg, 0)
	sendMsgs := make([]string, 0)
	mockNet.EXPECT().BroadcastMsg(gomock.Any(), gomock.Any()).DoAndReturn(
		func(msg []byte, msgType netPb.NetMsg_MsgType) error {
			broadcastMsgs = append(broadcastMsgs, netMsg{msgType: msgType, bz: msg})
			return nil
		}).AnyTimes()
	mockNet.EXPECT().SendMsg(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(msg []byte, msgType netPb.NetMsg_MsgType, to ...string) error {
			sendMsgs = append(sendMsgs, fmt.Sprintf("msgType: %d, to: %v", msgType, to))
			return nil
		}).AnyTimes()
	mockNet.EXPECT().Subscribe(gomock.Any(), gomock.Any()).AnyTimes()
	mockNet.EXPECT().ReceiveMsg(gomock.Any(), gomock.Any()).AnyTimes()

	return mockNet
}

func newMockVerifier(ctrl *gomock.Controller) protocol.BlockVerifier {
	mockVerify := mock.NewMockBlockVerifier(ctrl)
	mockVerify.EXPECT().VerifyBlock(gomock.Any(), gomock.Any()).AnyTimes()
	return mockVerify
}

func newMockCommitter(ctrl *gomock.Controller, mockLedger protocol.LedgerCache) protocol.BlockCommitter {
	mockCommit := mock.NewMockBlockCommitter(ctrl)
	mockCommit.EXPECT().AddBlock(gomock.Any()).DoAndReturn(func(blk *commonPb.Block) error {
		mockLedger.SetLastCommittedBlock(blk)
		return nil
	}).AnyTimes()
	return mockCommit
}

func newMockBlockChainStore(ctrl *gomock.Controller) protocol.BlockchainStore {
	mockStore := mock.NewMockBlockchainStore(ctrl)
	blocks := make(map[int64]*commonPb.Block)
	mockStore.EXPECT().PutBlock(gomock.Any(), gomock.Any()).DoAndReturn(
		func(blk *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
			blocks[blk.Header.BlockHeight] = blk
			return nil
		}).AnyTimes()
	mockStore.EXPECT().GetBlock(gomock.Any()).DoAndReturn(func(height int64) (*commonPb.Block, error) {
		if blk, exist := blocks[height]; exist {
			return blk, nil
		}
		return nil, fmt.Errorf("block not find")
	}).AnyTimes()
	mockStore.EXPECT().GetArchivedPivot().AnyTimes()
	return mockStore
}
