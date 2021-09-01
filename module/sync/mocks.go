/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	mbusmock "chainmaker.org/chainmaker/common/v2/msgbus/mock"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/protocol/v2"
)

var errStr = "implement me"

type netMsg struct {
	msgType netPb.NetMsg_MsgType
	bz      []byte
}

type MockNet struct {
	broadcastMsgs []netMsg
	sendMsgs      []string
}

func NewMockNet() *MockNet {
	return &MockNet{broadcastMsgs: make([]netMsg, 0, 100)}
}

func (m MockNet) ChainId() string {
	panic(errStr)
}

func (m *MockNet) BroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	m.broadcastMsgs = append(m.broadcastMsgs, netMsg{msgType: msgType, bz: msg})
	return nil
}

func (m *MockNet) Subscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	return nil
}

func (m MockNet) CancelSubscribe(msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m MockNet) ConsensusBroadcastMsg(msg []byte, msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m MockNet) ConsensusSubscribe(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	panic(errStr)
}

func (m MockNet) CancelConsensusSubscribe(msgType netPb.NetMsg_MsgType) error {
	panic(errStr)
}

func (m *MockNet) SendMsg(msg []byte, msgType netPb.NetMsg_MsgType, to ...string) error {
	m.sendMsgs = append(m.sendMsgs, fmt.Sprintf("msgType: %d, to: %v", msgType, to))
	return nil
}

func (m MockNet) ReceiveMsg(msgType netPb.NetMsg_MsgType, handler protocol.MsgHandler) error {
	return nil
}

func (m MockNet) Start() error {
	panic(errStr)
}

func (m MockNet) Stop() error {
	panic(errStr)
}

func (m MockNet) GetNodeUidByCertId(certId string) (string, error) {
	panic(errStr)
}

func (m MockNet) GetChainNodesInfoProvider() protocol.ChainNodesInfoProvider {
	panic(errStr)
}

type MockStore struct {
	blocks map[uint64]*commonPb.Block
}

func (m MockStore) GetContractByName(name string) (*commonPb.Contract, error) {
	panic("implement me")
}

func (m MockStore) GetContractBytecode(name string) ([]byte, error) {
	panic("implement me")
}

func (m MockStore) GetHeightByHash(blockHash []byte) (uint64, error) {
	panic("implement me")
}

func (m MockStore) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	panic("implement me")
}

func (m MockStore) GetTxHeight(txId string) (uint64, error) {
	panic("implement me")
}

func (m MockStore) GetArchivedPivot() uint64 {
	return 0
}

func (m MockStore) ArchiveBlock(archiveHeight uint64) error {
	panic("implement me")
}

func (m MockStore) RestoreBlocks(serializedBlocks [][]byte) error {
	panic("implement me")
}

func (m MockStore) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	panic(errStr)
}

func (m MockStore) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	panic(errStr)
}

func (m MockStore) ExecDdlSql(contractName, sql string) error {
	panic(errStr)
}
func (m MockStore) GetLastChainConfig() (*configPb.ChainConfig, error) {
	panic(errStr)
}
func (m MockStore) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic(errStr)
}

func (m MockStore) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic(errStr)
}

func (m MockStore) CommitDbTransaction(txName string) error {
	panic(errStr)
}

func (m MockStore) RollbackDbTransaction(txName string) error {
	panic(errStr)
}

func (m MockStore) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	panic(errStr)
}

func (m MockStore) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	panic(errStr)
}

func (m MockStore) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	panic(errStr)
}

func (m MockStore) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
	panic(errStr)
}

func (m MockStore) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
	panic(errStr)
}

func NewMockStore() *MockStore {
	return &MockStore{blocks: make(map[uint64]*commonPb.Block)}
}

func (m MockStore) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	panic(errStr)
}
func (m MockStore) GetTopicTableColumn(tableName string) ([]string, error) {
	panic(errStr)
}
func (m MockStore) BlockExists(blockHash []byte) (bool, error) {
	panic(errStr)

}

func (m MockStore) GetBlock(height uint64) (*commonPb.Block, error) {
	if blk, exist := m.blocks[height]; exist {
		return blk, nil
	}
	return nil, fmt.Errorf("block not find")
}

func (m MockStore) GetBlockWithRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
	panic(errStr)
}

func (m MockStore) TxExists(txId string) (bool, error) {
	panic(errStr)
}

func (m MockStore) GetTxConfirmedTime(txId string) (int64, error) {
	panic(errStr)
}

func (m *MockStore) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	m.blocks[block.Header.BlockHeight] = block
	return nil
}

func (m MockStore) GetLastConfigBlock() (*commonPb.Block, error) {
	panic(errStr)
}

func (m MockStore) GetBlockByTx(txId string) (*commonPb.Block, error) {
	panic(errStr)
}

func (m MockStore) GetBlockWithTxRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
	panic(errStr)
}

func (m MockStore) GetTx(txId string) (*commonPb.Transaction, error) {
	panic(errStr)
}

func (m MockStore) HasTx(txId string) (bool, error) {
	panic(errStr)
}

func (m MockStore) GetLastBlock() (*commonPb.Block, error) {
	panic(errStr)
}

func (m MockStore) ReadObject(contractName string, key []byte) ([]byte, error) {
	panic(errStr)
}

func (m MockStore) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	panic(errStr)
}

func (m MockStore) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
	panic(errStr)
}

func (m MockStore) GetDBHandle(dbName string) protocol.DBHandle {
	panic(errStr)
}

func (m MockStore) Close() error {
	panic(errStr)
}

type MockVerifier struct {
}

func NewMockVerifier() *MockVerifier {
	return &MockVerifier{}
}

func (m MockVerifier) VerifyBlock(block *commonPb.Block, mode protocol.VerifyMode) error {
	return nil
}

func (m MockVerifier) GetLastProposedBlock(b *commonPb.Block) (*commonPb.Block, map[string]*commonPb.TxRWSet) {
	panic(errStr)
}

type MockCommit struct {
	cache protocol.LedgerCache
}

func NewMockCommit(cache protocol.LedgerCache) *MockCommit {
	return &MockCommit{cache: cache}
}

func (m *MockCommit) AddBlock(blk *commonPb.Block) error {
	m.cache.SetLastCommittedBlock(blk)
	return nil
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
	mockLedger.EXPECT().CurrentHeight().DoAndReturn(func() (uint64, error) {
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
	blocks := make(map[uint64]*commonPb.Block)
	mockStore.EXPECT().PutBlock(gomock.Any(), gomock.Any()).DoAndReturn(
		func(blk *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
			blocks[blk.Header.BlockHeight] = blk
			return nil
		}).AnyTimes()
	mockStore.EXPECT().GetBlock(gomock.Any()).DoAndReturn(func(height uint64) (*commonPb.Block, error) {
		if blk, exist := blocks[height]; exist {
			return blk, nil
		}
		return nil, fmt.Errorf("block not find")
	}).AnyTimes()
	mockStore.EXPECT().GetArchivedPivot().AnyTimes()
	return mockStore
}
