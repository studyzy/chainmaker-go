/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	netpb "chainmaker.org/chainmaker/pb-go/v2/net"
)

type Blocker struct {
	sync.Mutex
	chainid     string
	maxBlockNum int
	blockNum    int
}

func newBlocker(chainid string, maxBlockNum int) *Blocker {
	return &Blocker{
		chainid:     chainid,
		maxBlockNum: maxBlockNum,
	}
}

func (b *Blocker) createBlock(height uint64) *common.Block {
	b.Lock()
	defer b.Unlock()

	clog.Infof("createBlock chainid: %s, maxBlockNum: %d, blockNum: %d, height: %d",
		b.chainid, b.maxBlockNum, b.blockNum, height)

	b.blockNum++

	block := &common.Block{
		Header: &common.BlockHeader{
			ChainId:     b.chainid,
			BlockHeight: height,
			Signature:   []byte(""),
			BlockHash:   []byte(""),
		},
		Dag: &common.DAG{},
		Txs: []*common.Transaction{
			{
				Payload: &common.Payload{
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

type mockCoreEngine struct {
	sync.Mutex
	msgbus.DefaultSubscriber

	t                 *testing.T
	id                string
	msgbus            msgbus.MessageBus
	blocker           *Blocker
	height            uint64
	canPropose        bool
	reachingConsensus bool
	verifiedBlocks    []*common.Block
	commitedBlocks    []*common.Block
	commitEventC      chan uint64
}

func newMockCoreEngine(t *testing.T, id string, mb msgbus.MessageBus, blocker *Blocker) *mockCoreEngine {
	ce := &mockCoreEngine{
		t:            t,
		id:           id,
		msgbus:       mb,
		blocker:      blocker,
		height:       1,
		canPropose:   false,
		commitEventC: make(chan uint64),
	}

	ce.msgbus.Register(msgbus.ProposeState, ce)
	ce.msgbus.Register(msgbus.VerifyBlock, ce)
	ce.msgbus.Register(msgbus.CommitBlock, ce)
	return ce
}

func (ce *mockCoreEngine) String() string {
	ce.Lock()
	defer ce.Unlock()

	return ce.ToStringWithoutLock()
}

func (ce *mockCoreEngine) ToStringWithoutLock() string {
	return fmt.Sprintf("id: %s, height: %d, canPropose: %v, reachingConsensus: %v",
		ce.id, ce.height, ce.canPropose, ce.reachingConsensus)
}

func (ce *mockCoreEngine) OnMessage(msg *msgbus.Message) {
	ce.Lock()
	defer ce.Unlock()

	clog.Infof("mockCoreEngine %s OnMessage topic: %s", ce.id, msg.Topic)

	switch msg.Topic {
	case msgbus.ProposeState:
		clog.Infof("mockCoreEngine %s, msg.topic: %s, msg.canPropose: %v",
			ce.ToStringWithoutLock(), msg.Topic, msg.Payload.(bool))
		ce.canPropose = msg.Payload.(bool)
		if ce.canPropose && !ce.reachingConsensus {
			block := ce.blocker.createBlock(ce.height)
			clog.Debugf("mockCoreEngine %s, block==nil: %v", ce.ToStringWithoutLock(), block == nil)
			ce.reachingConsensus = true
			clog.Infof("mockCoreEngine id: %s ProposedBlock block: %v", ce.id, block)
			ce.msgbus.Publish(msgbus.ProposedBlock, block)
		}
	case msgbus.VerifyBlock:
		verifyResultMsg := &consensuspb.VerifyResult{
			VerifiedBlock: msg.Payload.(*common.Block),
			Code:          consensuspb.VerifyResult_SUCCESS,
		}

		ce.reachingConsensus = true
		block := msg.Payload.(*common.Block)
		ce.verifiedBlocks = append(ce.verifiedBlocks, block)
		clog.Infof("mockCoreEngine %s %s %v", ce.id, msg.Topic, block.Header.BlockHeight)
		ce.msgbus.Publish(msgbus.VerifyResult, verifyResultMsg)
	case msgbus.CommitBlock:
		block := msg.Payload.(*common.Block)
		clog.Infof("mockCoreEngine %s, topic: %s, blockHeight: %d", ce.ToStringWithoutLock(), msg.Topic, block.Header.BlockHeight)
		ce.commitedBlocks = append(ce.commitedBlocks, block)
		ce.reachingConsensus = false
		ce.height++
		ce.commitEventC <- block.Header.BlockHeight
	default:
		ce.t.Errorf("error msg topic: %d", msg.Topic)
	}
}

type mockNet struct {
	msgbus.DefaultSubscriber

	id    string
	buses map[string]msgbus.MessageBus
}

func newMockNet(id string, buses map[string]msgbus.MessageBus) *mockNet {
	net := &mockNet{
		id:    id,
		buses: buses,
	}

	net.buses[id].Register(msgbus.SendConsensusMsg, net)
	return net
}

func (net *mockNet) OnMessage(msg *msgbus.Message) {
	clog.Infof("mockNet %s receive topic: %s", net.id, msg.Topic)
	switch msg.Topic {
	case msgbus.SendConsensusMsg:
		consMsg := msg.Payload.(*netpb.NetMsg)
		to := consMsg.To
		clog.Infof("mockNet %s publish %s to: %s", net.id, msgbus.RecvConsensusMsg, to)
		if _, ok := net.buses[to]; ok {
			net.buses[to].Publish(msgbus.RecvConsensusMsg, consMsg)
		}
	}
}

/*
func TestConsensusTBFTImpl_OneNode_KeepGrowing(t *testing.T) {
	chainid := "TestConsensusTBFTImpl_OneNode_KeepGrowing"

	chainConfig := &configPb.ChainConfig{
		Crypto: &pb.CryptoConfig{
			Hash: crypto.CRYPTO_ALGO_SHA256,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	signer := mock.NewMockSigningMember(ctrl)
	signer.EXPECT().Sign(gomock.Any(), gomock.Any()).Return([]byte("hash"), nil).AnyTimes()
	ledgerCache := mock.NewMockLedgerCache(ctrl)
	ledgerCache.EXPECT().CurrentHeight().Return(int64(1), nil).AnyTimes()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()

	msgBus := msgbus.NewMessageBus()
	var maxHeight int = 100
	nodeid := "node0"
	blocker := newBlocker(chainid, maxHeight)
	ce := newMockCoreEngine(t, nodeid, msgBus, blocker)

	consensus, _ := New(chainid, nodeid, []string{nodeid}, signer, ledgerCache, msgBus, chainConf)
	if err := consensus.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Init() error = %v", err)
	}

	timer := time.NewTimer(time.Duration(2*maxHeight) * time.Second)
	commitBlockNum := 0

Loop:
	for {
		select {
		case <-ce.commitEventC:
			commitBlockNum++

			if commitBlockNum == maxHeight {
				break Loop
			}

		case <-timer.C:
			t.Errorf("ce: %s timeout", ce)
		}
	}

	ce.Lock()
	if len(ce.commitedBlocks) < maxHeight {
		t.Errorf("len(ce.commitedBlocks): %d, expected maxHeight: %d", len(ce.commitedBlocks), maxHeight)
	}
	for i, block := range ce.commitedBlocks {
		if block.Header.BlockHeight != int64(i+1) {
			t.Errorf("ce: %s, height: %d, expected height: %d", ce, block.Header.BlockHeight, i)
		}
	}
	ce.Unlock()
}

func TestConsensusTBFTImpl_FourNode_KeepGrowing(t *testing.T) {
	chainid := "TestConsensusTBFTImpl_FourNode_KeepGrowing"

	chainConfig := &configPb.ChainConfig{
		Crypto: &pb.CryptoConfig{
			Hash: crypto.CRYPTO_ALGO_SHA256,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	signer := mock.NewMockSigningMember(ctrl)
	signer.EXPECT().Sign(gomock.Any(), gomock.Any()).Return([]byte("hash"), nil).AnyTimes()
	ledgerCache := mock.NewMockLedgerCache(ctrl)
	ledgerCache.EXPECT().CurrentHeight().Return(int64(1), nil).AnyTimes()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()

	nodeId0 := "node0"
	nodeId1 := "node1"
	nodeId2 := "node2"
	nodeId3 := "node3"

	msgbus0 := msgbus.NewMessageBus()
	msgbus1 := msgbus.NewMessageBus()
	msgbus2 := msgbus.NewMessageBus()
	msgbus3 := msgbus.NewMessageBus()

	buses := make(map[string]msgbus.MessageBus)
	buses[nodeId0] = msgbus0
	buses[nodeId1] = msgbus1
	buses[nodeId2] = msgbus2
	buses[nodeId3] = msgbus3

	newMockNet(nodeId0, buses)
	newMockNet(nodeId1, buses)
	newMockNet(nodeId2, buses)
	newMockNet(nodeId3, buses)

	var maxHeight int = 100
	blocker := newBlocker(chainid, maxHeight)
	ce0 := newMockCoreEngine(t, nodeId0, msgbus0, blocker)
	ce1 := newMockCoreEngine(t, nodeId1, msgbus1, blocker)
	ce2 := newMockCoreEngine(t, nodeId2, msgbus2, blocker)
	ce3 := newMockCoreEngine(t, nodeId3, msgbus3, blocker)

	node0, _ := New(
		chainid,
		nodeId0,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus0,
		chainConf)
	if err := node0.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Init() error = %v", err)
	}

	node1, _ := New(
		chainid,
		nodeId1,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus1,
		chainConf)
	if err := node1.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Init() error = %v", err)
	}

	node2, _ := New(
		chainid,
		nodeId2,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus2,
		chainConf)
	if err := node2.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Start() error = %v", err)
	}

	node3, _ := New(
		chainid,
		nodeId3,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus3,
		chainConf)
	if err := node3.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Start() error = %v", err)
	}

	var wg sync.WaitGroup
	ces := []*mockCoreEngine{ce0, ce1, ce2, ce3}
	wg.Add(len(ces))
	for _, ce := range ces {
		go func(ce *mockCoreEngine) {
			defer wg.Done()
			timer := time.NewTimer(time.Duration(2*maxHeight) * time.Second)
			commitBlockNum := 0

		Loop:
			for {
				select {
				case <-ce.commitEventC:
					commitBlockNum++

					if commitBlockNum == maxHeight {
						break Loop
					}

				case <-timer.C:
					t.Errorf("ce: %s timeout", ce)
					return
				}
			}

			ce.Lock()
			if len(ce.commitedBlocks) < maxHeight {
				t.Errorf("len(ce.commitedBlocks): %d, expected maxHeight: %d", len(ce.commitedBlocks), maxHeight)
			}
			for i, block := range ce.commitedBlocks {
				if block.Header.BlockHeight != int64(i+1) {
					t.Errorf("ce: %s, height: %d, expected height: %d", ce, block.Header.BlockHeight, i)
				}
			}
			ce.Unlock()
		}(ce)
	}

	wg.Wait()
}

func TestConsensusTBFTImpl_FourNode_KeepGrowing_FollowerDown(t *testing.T) {
	chainid := "TestConsensusTBFTImpl_FourNode_KeepGrowing_FollowerDown"

	chainConfig := &configPb.ChainConfig{
		Crypto: &pb.CryptoConfig{
			Hash: crypto.CRYPTO_ALGO_SHA256,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	signer := mock.NewMockSigningMember(ctrl)
	signer.EXPECT().Sign(gomock.Any(), gomock.Any()).Return([]byte("hash"), nil).AnyTimes()
	ledgerCache := mock.NewMockLedgerCache(ctrl)
	ledgerCache.EXPECT().CurrentHeight().Return(int64(1), nil).AnyTimes()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()

	nodeId0 := "node0"
	nodeId1 := "node1"
	nodeId2 := "node2"
	nodeId3 := "node3"

	msgbus0 := msgbus.NewMessageBus()
	msgbus1 := msgbus.NewMessageBus()
	msgbus2 := msgbus.NewMessageBus()

	buses := make(map[string]msgbus.MessageBus)
	buses[nodeId0] = msgbus0
	buses[nodeId1] = msgbus1
	buses[nodeId2] = msgbus2

	newMockNet(nodeId0, buses)
	newMockNet(nodeId1, buses)
	newMockNet(nodeId2, buses)

	var maxHeight int = 100
	blocker := newBlocker(chainid, maxHeight)
	ce0 := newMockCoreEngine(t, nodeId0, msgbus0, blocker)
	ce1 := newMockCoreEngine(t, nodeId1, msgbus1, blocker)
	ce2 := newMockCoreEngine(t, nodeId2, msgbus2, blocker)

	node0, _ := New(
		chainid,
		nodeId0,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus0,
		chainConf)
	if err := node0.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Init() error = %v", err)
	}

	node1, _ := New(
		chainid,
		nodeId1,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus1,
		chainConf)
	if err := node1.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Init() error = %v", err)
	}

	node2, _ := New(
		chainid,
		nodeId2,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus2,
		chainConf)
	if err := node2.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Start() error = %v", err)
	}

	var wg sync.WaitGroup
	ces := []*mockCoreEngine{ce0, ce1, ce2}
	wg.Add(len(ces))
	for _, ce := range ces {
		go func(ce *mockCoreEngine) {
			defer wg.Done()
			timer := time.NewTimer(time.Duration(2*maxHeight) * time.Second)
			commitBlockNum := 0

		Loop:
			for {
				select {
				case <-ce.commitEventC:
					commitBlockNum++

					if commitBlockNum == maxHeight {
						break Loop
					}

				case <-timer.C:
					t.Errorf("ce: %s timeout", ce)
					return
				}
			}

			ce.Lock()
			if len(ce.commitedBlocks) < maxHeight {
				t.Errorf("len(ce.commitedBlocks): %d, expected maxHeight: %d", len(ce.commitedBlocks), maxHeight)
			}
			for i, block := range ce.commitedBlocks {
				if block.Header.BlockHeight != int64(i+1) {
					t.Errorf("ce: %s, height: %d, expected height: %d", ce, block.Header.BlockHeight, i)
				}
			}
			ce.Unlock()
		}(ce)
	}

	wg.Wait()
}

func TestConsensusTBFTImpl_FourNode_KeepGrowing_LeaderDown(t *testing.T) {
	chainid := "TestConsensusTBFTImpl_FourNode_KeepGrowing_LeaderDown"

	chainConfig := &configPb.ChainConfig{
		Crypto: &pb.CryptoConfig{
			Hash: crypto.CRYPTO_ALGO_SHA256,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	signer := mock.NewMockSigningMember(ctrl)
	signer.EXPECT().Sign(gomock.Any(), gomock.Any()).Return([]byte("hash"), nil).AnyTimes()
	ledgerCache := mock.NewMockLedgerCache(ctrl)
	ledgerCache.EXPECT().CurrentHeight().Return(int64(1), nil).AnyTimes()
	chainConf := mock.NewMockChainConf(ctrl)
	chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()

	nodeId0 := "node0"
	nodeId1 := "node1"
	nodeId2 := "node2"
	nodeId3 := "node3"

	msgbus1 := msgbus.NewMessageBus()
	msgbus2 := msgbus.NewMessageBus()
	msgbus3 := msgbus.NewMessageBus()

	buses := make(map[string]msgbus.MessageBus)
	buses[nodeId1] = msgbus1
	buses[nodeId2] = msgbus2
	buses[nodeId3] = msgbus3

	newMockNet(nodeId1, buses)
	newMockNet(nodeId2, buses)
	newMockNet(nodeId3, buses)

	var maxHeight int = 50
	blocker := newBlocker(chainid, maxHeight)
	ce1 := newMockCoreEngine(t, nodeId1, msgbus1, blocker)
	ce2 := newMockCoreEngine(t, nodeId2, msgbus2, blocker)
	ce3 := newMockCoreEngine(t, nodeId3, msgbus3, blocker)

	node1, _ := New(
		chainid,
		nodeId1,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus1,
		chainConf)
	if err := node1.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Init() error = %v", err)
	}

	node2, _ := New(
		chainid,
		nodeId2,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus2,
		chainConf)
	if err := node2.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Start() error = %v", err)
	}

	node3, _ := New(
		chainid,
		nodeId3,
		[]string{nodeId0, nodeId1, nodeId2, nodeId3},
		signer,
		ledgerCache,
		msgbus3,
		chainConf)
	if err := node3.Start(); err != nil {
		t.Errorf("ConsensusTBFTImpl.Start() error = %v", err)
	}

	var wg sync.WaitGroup
	ces := []*mockCoreEngine{ce1, ce2, ce3}
	wg.Add(len(ces))
	for _, ce := range ces {
		go func(ce *mockCoreEngine) {
			defer wg.Done()
			timer := time.NewTimer(time.Duration(10*maxHeight) * time.Second)
			commitBlockNum := 0

		Loop:
			for {
				select {
				case <-ce.commitEventC:
					commitBlockNum++

					if commitBlockNum == maxHeight {
						break Loop
					}

				case <-timer.C:
					t.Errorf("ce: %s timeout", ce)
					return
				}
			}

			ce.Lock()
			if len(ce.commitedBlocks) != maxHeight {
				t.Errorf("len(ce.commitedBlocks): %d, expected maxHeight: %d", len(ce.commitedBlocks), maxHeight)
			}
			for i, block := range ce.commitedBlocks {
				if block.Header.BlockHeight != int64(i+1) {
					t.Errorf("ce: %s, height: %d, expected height: %d", ce, block.Header.BlockHeight, i)
				}
			}
			ce.Unlock()
		}(ce)
	}

	wg.Wait()
}
*/
