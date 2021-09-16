/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consensus_mock

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"

	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	systemPb "chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"

	"github.com/gogo/protobuf/proto"
)

type MockLedger struct {
	chainId            string                       // 链ID标识
	lastProposedBlock  map[uint64][]*commonPb.Block // 当前提案的区块，同一块高度可能会产生多个提案区块
	lastCommittedBlock *commonPb.Block              // 账本最新区块链
	rwMu               sync.RWMutex
}

func NewLedger(chainid string, block *commonPb.Block) *MockLedger {
	l := &MockLedger{
		chainId:            chainid,
		lastProposedBlock:  make(map[uint64][]*commonPb.Block),
		lastCommittedBlock: block,
	}
	l.lastProposedBlock[0] = append(l.lastProposedBlock[0], block)
	return l
}

func (l *MockLedger) SetLastProposedBlock(block *commonPb.Block) {
	h := block.Header.BlockHeight
	l.lastProposedBlock[h] = append(l.lastProposedBlock[h], block)
}

func (l *MockLedger) SetLastCommittedBlock(block *commonPb.Block) {
	l.lastCommittedBlock = block
}

func (l *MockLedger) GetLastCommittedBlock() *commonPb.Block {
	return l.lastCommittedBlock
}

func (l *MockLedger) CurrentHeight() (uint64, error) {
	return l.lastCommittedBlock.Header.BlockHeight, nil
}

type MockVerifier struct {
	ledgerCache protocol.LedgerCache
	msgBus      msgbus.MessageBus
	log         *logger.CMLogger
}

func NewMockVerifier(ledgerCache protocol.LedgerCache, msgBus msgbus.MessageBus, log *logger.CMLogger) *MockVerifier {
	return &MockVerifier{
		ledgerCache: ledgerCache,
		msgBus:      msgBus,
		log:         log,
	}
}

func (v *MockVerifier) VerifyBlock(block *commonPb.Block, mode protocol.VerifyMode) error {
	if block == nil {
		return errors.New("invalid block, yield verify")
	}

	return nil
}

func (v *MockVerifier) GetLastProposedBlock(block *commonPb.Block) (*commonPb.Block, map[string]*commonPb.TxRWSet) {
	panic("GetLastProposedBlock not implement")
}

type MockCommitter struct {
	ledgerCache  protocol.LedgerCache
	store        *MockBlockchainStore
	msgBus       msgbus.MessageBus
	log          *logger.CMLogger
	commitEventC chan uint64
}

func NewMockCommitter(ledgerCache protocol.LedgerCache,
	store *MockBlockchainStore,
	msgbus msgbus.MessageBus,
	log *logger.CMLogger,
	commitEventC chan uint64) *MockCommitter {
	return &MockCommitter{
		ledgerCache:  ledgerCache,
		store:        store,
		msgBus:       msgbus,
		log:          log,
		commitEventC: commitEventC,
	}
}

func (cm *MockCommitter) AddBlock(block *commonPb.Block) error {
	height := block.Header.BlockHeight
	currentHeight, err := cm.ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}
	if height <= currentHeight {
		cm.log.Infof("block(%d,%x) has already put", block.Header.BlockHeight, block.Header.BlockHash)
		return nil
	}
	// lastProposed, rwSetMap := cm.proposedCache.GetLastProposedBlock(block)
	// if lastProposed == nil {
	// 	cm.log.Warnf("block(%d,%x) is not verified", block.Header.BlockHeight, block.Header.BlockHash)
	// 	return errors.New("block is not verified")
	// }

	cm.ledgerCache.SetLastCommittedBlock(block)
	cm.store.SaveBlock(block)

	consensusArgs, err := utils.GetConsensusArgsFromBlock(block)
	if err != nil {
		return err
	}
	if err != nil {
		return errors.New("consensusArgs.ConsensusData is nil")
	}
	rwset, _ := proto.Marshal(consensusArgs.ConsensusData)
	cm.store.WriteObject(systemPb.SystemContract_GOVERNANCE.String(), rwset)
	// chain.proposedCache.ClearProposedBlock(block.Header.BlockHeight)
	// chain.proposedCache.ResetProposedThisRound()
	cm.msgBus.Publish(msgbus.BlockInfo, block) // synchronize new block height to consensus and sync module
	cm.commitEventC <- height
	cm.log.Infof("block(%d,%s) accepted", block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	return nil
}

type MockProposer struct {
	sync.Mutex

	id         string
	chainid    string
	isProposer bool // whether current node can propose block now

	log         *logger.CMLogger
	msgBus      msgbus.MessageBus
	ledgerCache protocol.LedgerCache
}

func (p *MockProposer) DiscardAboveHeight(baseHeight int64) []*commonPb.Block {
	panic("implement me")
}

func (p *MockProposer) ClearTheBlock(block *commonPb.Block) {
	panic("implement me")
}

func (p *MockProposer) ClearProposedBlockAt(height uint64) {
	panic("implement me")
}

func (p *MockProposer) GetProposedBlocksAt(height uint64) []*commonPb.Block {
	panic("implement me")
}

func (p *MockProposer) GetProposedBlock(b *commonPb.Block) (*commonPb.Block, map[string]*commonPb.TxRWSet) {
	panic("implement me")
}

func (p *MockProposer) SetProposedBlock(b *commonPb.Block, rwSetMap map[string]*commonPb.TxRWSet, selfProposed bool) error {
	panic("implement me")
}

func (p *MockProposer) GetSelfProposedBlockAt(height uint64) *commonPb.Block {
	panic("implement me")
}

func (p *MockProposer) GetProposedBlockByHashAndHeight(hash []byte, height uint64) (*commonPb.Block, map[string]*commonPb.TxRWSet) {
	panic("implement me")
}

func (p *MockProposer) HasProposedBlockAt(height uint64) bool {
	panic("implement me")
}

func (p *MockProposer) IsProposedAt(height uint64) bool {
	panic("implement me")
}

func (p *MockProposer) SetProposedAt(height uint64) {
	panic("implement me")
}

func (p *MockProposer) ResetProposedAt(height uint64) {
	panic("implement me")
}

func (p *MockProposer) KeepProposedBlock(hash []byte, height uint64) []*commonPb.Block {
	panic("implement me")
}

func NewMockProposer(chainid string,
	id string,
	// singer protocol.SigningMember,
	ledgerCache protocol.LedgerCache,
	msgbus msgbus.MessageBus, log *logger.CMLogger) *MockProposer {
	return &MockProposer{
		chainid: chainid,
		id:      id,
		// proposer:    singer,
		ledgerCache: ledgerCache,
		msgBus:      msgbus,
		log:         log,
		isProposer:  false,
	}
}

func (b *MockProposer) CreateBlock(height uint64, perHash []byte) *commonPb.Block {
	b.Lock()
	defer b.Unlock()

	b.log.Infof("start createBlock")
	// proposer, _ := b.proposer.Serialize()
	// lastblock := b.ledgerCache.GetLastCommittedBlock()
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:      b.chainid,
			BlockHeight:  height,
			Signature:    []byte(""),
			BlockHash:    []byte(""),
			PreBlockHash: perHash,
			Proposer:     &accesscontrol.Member{MemberInfo: []byte(b.id)},
		},
		Dag: &commonPb.DAG{},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId: b.chainid,
				},
			},
		},
	}

	// blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainid, height)))
	blockHash := []byte(fmt.Sprintf("%s-%d-%s", b.chainid, height, time.Now()))
	block.Header.BlockHash = blockHash[:]

	// txHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", blockHash, 0)))
	// block.Txs[0].Payload.TxId = string(txHash[:])

	return block
}

type MockCoreEngine struct {
	sync.Mutex
	msgbus.DefaultSubscriber

	t             *testing.T
	id            string
	chainid       string
	msgbus        msgbus.MessageBus
	Proposer      *MockProposer
	Verifer       *MockVerifier
	Committer     *MockCommitter
	blockProposer *MockProposer

	height            uint64
	canPropose        bool
	reachingConsensus bool
	Ledger            protocol.LedgerCache
	commitEventC      chan uint64
	isCreateBlock     bool

	logger *logger.CMLogger
}

func NewMockCoreEngine(t *testing.T, id string,
	chainid string,
	mb msgbus.MessageBus,
	ledger protocol.LedgerCache,
	store *MockBlockchainStore,
	iscreate bool) *MockCoreEngine {
	logger := logger.GetLogger(logger.MODULE_CORE)
	commitEventC := make(chan uint64, 1)
	ce := &MockCoreEngine{
		t:             t,
		id:            id,
		msgbus:        mb,
		Proposer:      NewMockProposer(chainid, id, ledger, mb, logger),
		Verifer:       NewMockVerifier(ledger, mb, logger),
		Committer:     NewMockCommitter(ledger, store, mb, logger, commitEventC),
		Ledger:        ledger,
		height:        1,
		canPropose:    false,
		commitEventC:  commitEventC,
		isCreateBlock: iscreate,

		logger: logger,
	}

	ce.msgbus.Register(msgbus.BuildProposal, ce)
	// ce.msgbus.Register(msgbus.VerifyBlock, ce)
	// ce.msgbus.Register(msgbus.CommitBlock, ce)
	return ce
}

func (ce *MockCoreEngine) Loop() {
	for {
		select {
		case height, ok := <-ce.commitEventC:
			if !ok {
				continue
			}
			ce.Lock()
			if height != ce.height {
				ce.logger.Errorf("core height not equal committer height: %v, %v", ce.height, height)
			}
			ce.height = height + 1
			ce.Unlock()
		}

	}
}

func (ce *MockCoreEngine) GetID() string {
	return ce.id
}

func (ce *MockCoreEngine) GetHeight() uint64 {
	return ce.height
}

func (ce *MockCoreEngine) String() string {
	ce.Lock()
	defer ce.Unlock()

	return ce.ToStringWithoutLock()
}

func (ce *MockCoreEngine) ToStringWithoutLock() string {
	return fmt.Sprintf("id: %s, height: %d, canPropose: %v, reachingConsensus: %v",
		ce.id, ce.height, ce.canPropose, ce.reachingConsensus)
}

func (ce *MockCoreEngine) OnMessage(msg *msgbus.Message) {
	ce.logger.Infof("MockCoreEngine %s OnMessage topic: %s", ce.id, msg.Topic)

	switch msg.Topic {
	case msgbus.BuildProposal:
		ce.logger.Infof("MockCoreEngine %s, msg.topic: %s, msg.BuildProposal: %v",
			ce.ToStringWithoutLock(), msg.Topic, msg.Payload.(*chainedbftpb.BuildProposal))
		ce.Lock()
		bp := msg.Payload.(*chainedbftpb.BuildProposal)
		ce.canPropose = bp.IsProposer
		if ce.canPropose && ce.isCreateBlock {
			block := ce.Proposer.CreateBlock(uint64(bp.Height), bp.PreHash)
			ce.logger.Infof("MockCoreEngine id: %s ProposedBlock block: %v", ce.id, block)
			ce.msgbus.Publish(msgbus.ProposedBlock, block)
		}
		ce.Unlock()

	default:
		ce.t.Errorf("error msg topic: %d", msg.Topic)
	}
}

type MockNet struct {
	msgbus.DefaultSubscriber

	id    string
	buses map[string]msgbus.MessageBus

	logger *logger.CMLogger
}

func NewMockNet(id string, buses map[string]msgbus.MessageBus) *MockNet {
	net := &MockNet{
		id:     id,
		buses:  buses,
		logger: logger.GetLogger(logger.MODULE_NET),
	}

	net.buses[id].Register(msgbus.SendConsensusMsg, net)
	return net
}

func (net *MockNet) OnMessage(msg *msgbus.Message) {
	net.logger.Infof("MockNet %s receive topic: %s", net.id, msg.Topic)
	switch msg.Topic {
	case msgbus.SendConsensusMsg:
		consMsg := msg.Payload.(*netPb.NetMsg)
		to := consMsg.To
		net.logger.Infof("MockNet %s publish %s to: %s", net.id, msgbus.RecvConsensusMsg, to)
		if _, ok := net.buses[to]; ok {
			net.buses[to].Publish(msgbus.RecvConsensusMsg, consMsg)
		}
	}
}

type MockProtocolNetService struct {
	mock.MockNetService
	nodeids map[string]string //certid -> nodeid
}

func NewMockProtocolNetService(nodeids map[string]string) *MockProtocolNetService {
	return &MockProtocolNetService{
		nodeids: nodeids,
	}
}

func (ns *MockProtocolNetService) GetNodeUidByCertId(certid string) (string, error) {
	return ns.nodeids[certid], nil
}

func (ns *MockProtocolNetService) AppendNodeID(certid string, nodeid string) error {
	ns.nodeids[certid] = nodeid
	return nil
}

type MockBlockchainStore struct {
	mock.MockBlockchainStore

	objectMap map[string][]byte
	blockList []*commonPb.Block

	rwMu sync.RWMutex
}

func (bs *MockBlockchainStore) GetHeightByHash(blockHash []byte) (uint64, error) {
	panic("implement me")
}

func (bs *MockBlockchainStore) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	panic("implement me")
}

func (bs *MockBlockchainStore) GetTxHeight(txId string) (uint64, error) {
	panic("implement me")
}

func (bs *MockBlockchainStore) GetArchivedPivot() uint64 {
	panic("implement me")
}

func (bs *MockBlockchainStore) ArchiveBlock(archiveHeight uint64) error {
	panic("implement me")
}

func (bs *MockBlockchainStore) RestoreBlocks(serializedBlocks [][]byte) error {
	panic("implement me")
}

func NewMockMockBlockchainStore(gensis *commonPb.Block, cf *chainconf.ChainConf) *MockBlockchainStore {
	bs := &MockBlockchainStore{
		objectMap: make(map[string][]byte, 0),
	}
	bs.blockList = append(bs.blockList, gensis)
	config := cf.ChainConfig()
	bconfig, _ := proto.Marshal(config)
	bs.objectMap[systemPb.SystemContract_CHAIN_CONFIG.String()] = bconfig
	return bs
}

func (bs *MockBlockchainStore) WriteObject(contractName string, object []byte) error {
	bs.objectMap[contractName] = object
	return nil
}

func (bs *MockBlockchainStore) SaveBlock(block *commonPb.Block) error {
	bs.blockList = append(bs.blockList, block)
	return nil
}

func (bs *MockBlockchainStore) ReadObject(contractName string, key []byte) ([]byte, error) {
	return bs.objectMap[contractName], nil
}

func (bs *MockBlockchainStore) GetLastBlock() (*commonPb.Block, error) {
	return bs.blockList[len(bs.blockList)-1], nil
}

func (bs *MockBlockchainStore) GetBlockAt(height uint64) (*commonPb.Block, error) {
	if height > uint64(len(bs.blockList)-1) {
		return nil, fmt.Errorf("has not block")
	}
	return bs.blockList[height], nil
}
