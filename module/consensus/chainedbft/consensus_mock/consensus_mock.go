/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consensus_mock

import (
	"chainmaker.org/chainmaker-go/consensus/government"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	netPb "chainmaker.org/chainmaker-go/pb/protogo/net"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/gogo/protobuf/proto"
)

type MockLedger struct {
	chainId            string                      // 链ID标识
	lastProposedBlock  map[int64][]*commonPb.Block // 当前提案的区块，同一块高度可能会产生多个提案区块
	lastCommittedBlock *commonPb.Block             // 账本最新区块链
	rwMu               sync.RWMutex
}

func NewLedger(chainid string, block *commonPb.Block) *MockLedger {
	l := &MockLedger{
		chainId:            chainid,
		lastProposedBlock:  make(map[int64][]*commonPb.Block),
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

func (l *MockLedger) CurrentHeight() (int64, error) {
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
	commitEventC chan int64
}

func NewMockCommitter(ledgerCache protocol.LedgerCache,
	store *MockBlockchainStore,
	msgbus msgbus.MessageBus,
	log *logger.CMLogger,
	commitEventC chan int64) *MockCommitter {
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
	cm.store.WriteObject(government.GovernmentContractName, rwset)
	// chain.proposedCache.ClearProposedBlock(block.Header.BlockHeight)
	// chain.proposedCache.ResetProposedThisRound()
	cm.msgBus.Publish(msgbus.BlockInfo, block) // synchronize new block height to consensus and sync module
	cm.commitEventC <- height
	cm.log.Infof("block(%d,%s) accepted", block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	return nil
}

type MockProposer struct {
	sync.Mutex

	chainid string
	id      string
	// proposer    protocol.SigningMember
	ledgerCache protocol.LedgerCache
	msgBus      msgbus.MessageBus
	log         *logger.CMLogger
	isProposer  bool // whether current node can propose block now
	// idle         bool        // whether current node is proposing or not

}

func (p *MockProposer) ClearProposedBlockAt(height int64) {
	panic("implement me")
}

func (p *MockProposer) GetProposedBlocksAt(height int64) []*commonPb.Block {
	panic("implement me")
}

func (p *MockProposer) GetProposedBlock(b *commonPb.Block) (*commonPb.Block, map[string]*commonPb.TxRWSet) {
	panic("implement me")
}

func (p *MockProposer) SetProposedBlock(b *commonPb.Block, rwSetMap map[string]*commonPb.TxRWSet, selfProposed bool) error {
	panic("implement me")
}

func (p *MockProposer) GetSelfProposedBlockAt(height int64) *commonPb.Block {
	panic("implement me")
}

func (p *MockProposer) GetProposedBlockByHashAndHeight(hash []byte, height int64) (*commonPb.Block, map[string]*commonPb.TxRWSet) {
	panic("implement me")
}

func (p *MockProposer) HasProposedBlockAt(height int64) bool {
	panic("implement me")
}

func (p *MockProposer) IsProposedAt(height int64) bool {
	panic("implement me")
}

func (p *MockProposer) SetProposedAt(height int64) {
	panic("implement me")
}

func (p *MockProposer) ResetProposedAt(height int64) {
	panic("implement me")
}

func (p *MockProposer) KeepProposedBlock(hash []byte, height int64) []*commonPb.Block {
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

func (b *MockProposer) CreateBlock(height int64, perHash []byte) *commonPb.Block {
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
			Proposer:     []byte(b.id),
		},
		Dag: &commonPb.DAG{},
		Txs: []*commonPb.Transaction{
			{
				Header: &commonPb.TxHeader{
					ChainId: b.chainid,
				},
			},
		},
	}

	// blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainid, height)))
	blockHash := []byte(fmt.Sprintf("%s-%d-%s", b.chainid, height, time.Now()))
	block.Header.BlockHash = blockHash[:]

	// txHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", blockHash, 0)))
	// block.Txs[0].Header.TxId = string(txHash[:])

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

	height            int64
	canPropose        bool
	reachingConsensus bool
	Ledger            protocol.LedgerCache
	commitEventC      chan int64
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
	commitEventC := make(chan int64, 1)
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

func (ce *MockCoreEngine) GetHeight() int64 {
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
			block := ce.Proposer.CreateBlock(int64(bp.Height), bp.PreHash)
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

func NewMockMockBlockchainStore(gensis *commonPb.Block, cf *chainconf.ChainConf) *MockBlockchainStore {
	bs := &MockBlockchainStore{
		objectMap: make(map[string][]byte, 0),
	}
	bs.blockList = append(bs.blockList, gensis)
	config := cf.ChainConfig()
	bconfig, _ := proto.Marshal(config)
	bs.objectMap[commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()] = bconfig
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

func (bs *MockBlockchainStore) GetBlockAt(height int64) (*commonPb.Block, error) {
	if height > int64(len(bs.blockList)-1) {
		return nil, fmt.Errorf("has not block")
	}
	return bs.blockList[height], nil
}
