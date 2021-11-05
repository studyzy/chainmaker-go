/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thoas/go-funk"
	"go.uber.org/zap"

	"chainmaker.org/chainmaker/chainconf/v2"
	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	netpb "chainmaker.org/chainmaker/pb-go/v2/net"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/raftwal/v2/wal"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"go.etcd.io/etcd/client/pkg/v3/fileutil"
	etcdraft "go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/etcd/server/v3/wal/walpb"
)

var (
	DefaultChanCap          = 1000
	walDir                  = "raftwal"
	snapDir                 = "snap"
	snapshotCatchUpEntriesN = uint64(5)
	defaultSnapCount        = uint64(10)
	isStarted               sync.Map
	instances               sync.Map
)

// mustMarshal marshals protobuf message to byte slice or panic
func mustMarshal(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
}

// mustUnmarshal unmarshals from byte slice to protobuf message or panic
func mustUnmarshal(b []byte, msg proto.Message) {
	if err := proto.Unmarshal(b, msg); err != nil {
		panic(err)
	}
}

// ConsensusRaftImpl is the implementation of Raft algorithm
// and it implements the ConsensusEngine interface.
type ConsensusRaftImpl struct {
	sync.RWMutex
	logger        *logger.CMLogger
	chainID       string
	singer        protocol.SigningMember
	ac            protocol.AccessControlProvider
	ledgerCache   protocol.LedgerCache
	chainConf     protocol.ChainConf
	msgbus        msgbus.MessageBus
	closeC        chan struct{}
	Id            uint64
	peers         []uint64
	isLeader      bool
	node          etcdraft.Node
	raftStorage   *etcdraft.MemoryStorage
	wal           *wal.WAL
	waldir        string
	asyncWalSave  bool
	snapdir       string
	snapCount     uint64
	snapshotter   *snap.Snapshotter
	confState     raftpb.ConfState
	snapshotIndex uint64
	appliedIndex  uint64
	proposedIndex uint64
	idToNodeId    sync.Map

	proposedBlockC chan *common.Block
	verifyResultC  chan *consensus.VerifyResult
	blockInfoC     chan *common.BlockInfo
	confChangeC    chan raftpb.ConfChange
	walSaveC       chan interface{}
	wg             sync.WaitGroup
	blockVerifier  protocol.BlockVerifier
	blockCommitter protocol.BlockCommitter
}

// ConsensusRaftImplConfig contains initialization config for ConsensusRaftImpl
type ConsensusRaftImplConfig struct {
	ChainID        string
	NodeId         string
	Singer         protocol.SigningMember
	Ac             protocol.AccessControlProvider
	LedgerCache    protocol.LedgerCache
	BlockVerifier  protocol.BlockVerifier
	BlockCommitter protocol.BlockCommitter
	ChainConf      protocol.ChainConf
	MsgBus         msgbus.MessageBus
}

// New creates a raft consensus instance
func New(config ConsensusRaftImplConfig) (*ConsensusRaftImpl, error) {
	consensus := &ConsensusRaftImpl{}
	lg := logger.GetLoggerByChain(logger.MODULE_CONSENSUS, config.ChainID)
	if started, ok := isStarted.Load(consensus.Id); ok && started.(bool) {
		if ins, ok := instances.Load(consensus.Id); ok && ins.(*ConsensusRaftImpl) != nil {
			lg.Infof("ConsensusRaftImpl[%x] is already exist, need to do nothing", consensus.Id)
			return ins.(*ConsensusRaftImpl), nil
		}
		isStarted.Delete(consensus.Id)
	}
	consensus.logger = lg
	consensus.chainID = config.ChainID
	consensus.singer = config.Singer
	consensus.ac = config.Ac
	consensus.ledgerCache = config.LedgerCache
	consensus.chainConf = config.ChainConf
	consensus.msgbus = config.MsgBus
	consensus.closeC = make(chan struct{})
	consensus.Id = computeRaftIdFromNodeId(config.NodeId)

	consensus.snapCount = localconf.ChainMakerConfig.ConsensusConfig.RaftConfig.SnapCount
	if consensus.snapCount == 0 {
		consensus.snapCount = defaultSnapCount
	}
	consensus.asyncWalSave = localconf.ChainMakerConfig.ConsensusConfig.RaftConfig.AsyncWalSave
	consensus.waldir = path.Join(localconf.ChainMakerConfig.GetStorePath(), consensus.chainID, walDir)
	consensus.snapdir = path.Join(localconf.ChainMakerConfig.GetStorePath(), consensus.chainID, snapDir)

	consensus.proposedBlockC = make(chan *common.Block, DefaultChanCap)
	consensus.verifyResultC = make(chan *consensuspb.VerifyResult, DefaultChanCap)
	consensus.blockInfoC = make(chan *common.BlockInfo, DefaultChanCap)
	consensus.confChangeC = make(chan raftpb.ConfChange, DefaultChanCap)
	consensus.walSaveC = make(chan interface{}, DefaultChanCap)
	consensus.blockVerifier = config.BlockVerifier
	consensus.blockCommitter = config.BlockCommitter

	consensus.logger.Infof("New ConsensusRaftImpl[%x]", consensus.Id)
	instances.Store(consensus.Id, consensus)
	return consensus, nil
}

// Start starts the raft instance
func (consensus *ConsensusRaftImpl) Start() error {
	if started, ok := isStarted.Load(consensus.Id); ok && started.(bool) {
		consensus.logger.Infof("ConsensusRaftImpl[%x] is already started, need to do nothing", consensus.Id)
		return nil
	}
	consensus.logger.Infof("ConsensusRaftImpl[%x] starting", consensus.Id)
	if !fileutil.Exist(consensus.snapdir) {
		if err := os.Mkdir(consensus.snapdir, 0750); err != nil {
			consensus.logger.Fatalf("[%x] cannot create dir for snapshot: %v", consensus.Id, err)
			return err
		}
	}
	consensus.snapshotter = snap.New(consensus.logger.Logger().Desugar(), consensus.snapdir)
	walExist := wal.Exist(consensus.waldir)
	consensus.wal = consensus.replayWAL()

	var idToNodes map[uint64]string
	consensus.peers, idToNodes = consensus.getPeersFromChainConf()
	for id, node := range idToNodes {
		consensus.idToNodeId.Store(id, node)
	}
	c := &etcdraft.Config{
		ID:              consensus.Id,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         consensus.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		// CheckQuorum:     true,
		Logger: NewLogger(consensus.logger.Logger()),
	}

	height, err := consensus.ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}

	if walExist || height != 0 {
		consensus.logger.Infof("[%x] restart raft walExist: %v, height: %v", consensus.Id, walExist, height)
		consensus.node = etcdraft.RestartNode(c)
	} else {
		consensus.logger.Infof("[%x] start raft walExist: %v, height: %v", consensus.Id, walExist, height)
		peers := []etcdraft.Peer{}
		for _, p := range consensus.peers {
			peers = append(peers, etcdraft.Peer{ID: p})
		}
		consensus.node = etcdraft.StartNode(c, peers)
	}
	consensus.wg = sync.WaitGroup{}
	consensus.wg.Add(1)
	go consensus.AsyncWalSave()
	go consensus.serve()
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.RecvConsensusMsg, consensus)
	_ = chainconf.RegisterVerifier(consensus.chainID, consensuspb.ConsensusType_RAFT, consensus)
	isStarted.Store(consensus.Id, true)

	return nil
}

// Start stops the raft instance
func (consensus *ConsensusRaftImpl) Stop() error {
	consensus.logger.Infof("ConsensusRaftImpl stopping")
	return nil
}

// OnMessage receives messages from msgbus
func (consensus *ConsensusRaftImpl) OnMessage(message *msgbus.Message) {
	switch message.Topic {
	case msgbus.ProposedBlock:
		if proposedBlock, ok := message.Payload.(*consensuspb.ProposalBlock); ok {
			consensus.proposedBlockC <- proposedBlock.Block
		}
	case msgbus.RecvConsensusMsg:
		if msg, ok := message.Payload.(*netpb.NetMsg); ok {
			raftMsg := raftpb.Message{}
			if err := raftMsg.Unmarshal(msg.Payload); err != nil {
				consensus.logger.Panicf("[%x] unmarshal message %v", consensus.Id, err)
			}
			consensus.logger.DebugDynamic(func() string {
				return fmt.Sprintf("[%x] receive message %v", consensus.Id, describeMessage(raftMsg))
			})
			if err := consensus.node.Step(context.Background(), raftMsg); err != nil {
				consensus.logger.Errorf("[%x] step message %v, err: %v", consensus.Id, describeMessage(raftMsg), err)
			}
		} else {
			panic(fmt.Errorf("receive message failed, error message type"))
		}
	}
}

func (consensus *ConsensusRaftImpl) OnQuit() {
	// do nothing
}

func (consensus *ConsensusRaftImpl) saveSnap(snap raftpb.Snapshot) error {
	consensus.logger.InfoDynamic(func() string {
		return fmt.Sprintf("saveSnap %v", describeSnapshot(snap))
	})
	walSnap := walpb.Snapshot{
		Index:     snap.Metadata.Index,
		Term:      snap.Metadata.Term,
		ConfState: &snap.Metadata.ConfState,
	}

	if err := consensus.wal.SaveSnapshot(walSnap); err != nil {
		return err
	}
	if err := consensus.snapshotter.SaveSnap(snap); err != nil {
		return err
	}
	if err := consensus.wal.ReleaseLockTo(snap.Metadata.Index); err != nil {
		return err
	}
	go consensus.PurgeFile(consensus.waldir)
	return nil
}

func (consensus *ConsensusRaftImpl) PurgeFile(dirname string) {
	fileNames, err := fileutil.ReadDir(dirname)
	if err != nil {
		return
	}
	//当wal文件不超过2条时候，不处理
	if len(fileNames) <= 2 {
		return
	}

	walFileNames := make([]string, 0)
	for _, fileName := range fileNames {
		if strings.HasSuffix(fileName, ".wal") {
			walFileNames = append(walFileNames, fileName)
		}
	}

	for len(walFileNames) > 1 {
		f := filepath.Join(dirname, walFileNames[0])
		l, err := fileutil.TryLockFile(f, os.O_WRONLY, fileutil.PrivateFileMode)
		if err != nil {
			break
		}
		consensus.logger.Infof("[%x] PurgeFile Start: %s", consensus.Id, zap.String("fileName", l.Name()))
		if err = os.Remove(f); err != nil {
			return
		}
		if err = l.Close(); err != nil {
			consensus.logger.Infof("[%x] failed to unlock/close: %s", consensus.Id, zap.String("path", l.Name()), zap.Error(err))
			return
		}
		consensus.logger.Infof("[%x] PurgeFile Success: %s", consensus.Id, zap.String("fileName", l.Name()))
		walFileNames = walFileNames[1:]
	}
}

func (consensus *ConsensusRaftImpl) serve() {
	snapshot, err := consensus.raftStorage.Snapshot()
	if err != nil {
		consensus.logger.Fatalf("[%x] raftStorage Snapshot error", consensus.Id, err)
	}
	consensus.confState = snapshot.Metadata.ConfState
	consensus.snapshotIndex = snapshot.Metadata.Index
	consensus.appliedIndex = snapshot.Metadata.Index
	consensus.logger.InfoDynamic(func() string {
		return fmt.Sprintf("[%x] begin serve with snap: %v, appliedIndex: %v",
			consensus.Id, describeSnapshot(snapshot), consensus.appliedIndex)
	})

	tickTime := localconf.ChainMakerConfig.ConsensusConfig.RaftConfig.Ticker
	if tickTime == 0 {
		tickTime = time.Nanosecond
	}
	ticker := time.NewTicker(tickTime * time.Second)
	defer func() {
		ticker.Stop()
		consensus.wal.Close()
		consensus.node.Stop()
		consensus.msgbus.UnRegister(msgbus.ProposedBlock, consensus)
		consensus.msgbus.UnRegister(msgbus.RecvConsensusMsg, consensus)
		isStarted.Delete(consensus.Id)
		instances.Delete(consensus.Id)
	}()

	for {
		select {
		case <-ticker.C:
			consensus.node.Tick()
			consensus.logger.Debugf("[%x] status: %s", consensus.Id, consensus.node.Status())
		case ready := <-consensus.node.Ready():
			if exit := consensus.NodeReady(ready); exit {
				consensus.logger.Debugf("exit consensus when process ready message")
				return
			}
		case block := <-consensus.proposedBlockC:
			consensus.ProposeBlock(block)
		case cc := <-consensus.confChangeC:
			consensus.logger.DebugDynamic(func() string {
				return fmt.Sprintf("[%x] ProposeConfChange %v", consensus.Id, describeConfChange(cc))
			})
			if err := consensus.node.ProposeConfChange(context.TODO(), cc); err != nil {
				consensus.logger.Panicf("[%x] propose config change error: %v", consensus.Id, err)
			}
		}
	}
}

func (consensus *ConsensusRaftImpl) NodeReady(ready etcdraft.Ready) (exit bool) {
	consensus.logger.DebugDynamic(func() string {
		return fmt.Sprintf("[%x] receive from raft ready, %v", consensus.Id, describeReady(ready))
	})
	if !etcdraft.IsEmptySnap(ready.Snapshot) && ready.Snapshot.Metadata.Index <= consensus.appliedIndex {
		consensus.logger.Fatalf("snapshot index: %v should > appliedIndex: %v",
			ready.Snapshot.Metadata.Index, consensus.appliedIndex)
	}
	//异步WalSave
	if consensus.asyncWalSave {
		consensus.walSaveC <- ready
	} else {
		consensus.processWalAndSnap(ready)
	}

	ok, configChanged := consensus.publishEntries(consensus.entriesToApply(ready.CommittedEntries))
	if !ok {
		for len(consensus.walSaveC) != 0 {
			time.Sleep(500 * time.Millisecond)
		}
		close(consensus.closeC)
		consensus.wg.Wait()
		consensus.maybeTriggerSnapshot(configChanged)
		consensus.logger.Infof("[%x] is deleted from consensus nodes", consensus.Id)
		return true
	}
	if consensus.asyncWalSave {
		consensus.walSaveC <- configChanged
	} else {
		consensus.maybeTriggerSnapshot(configChanged)
	}
	if ready.SoftState != nil {
		consensus.isLeader = atomic.LoadUint64(&ready.SoftState.Lead) == consensus.Id
	}
	consensus.sendProposeState(consensus.isLeader)
	return false
}

func (consensus *ConsensusRaftImpl) ProposeBlock(block *common.Block) {
	consensus.Lock()
	defer consensus.Unlock()
	if block.Header.BlockHeight <= consensus.proposedIndex {
		consensus.logger.Debugf("[%x] got proposed block in wrong height:%d",
			consensus.Id, block.Header.BlockHeight)
		return
	}
	// Add hash and signature to block
	hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
	if err != nil {
		consensus.logger.Errorf("[%x] sign block failed, %s", consensus.Id, err)
		return
	}
	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig
	if block.AdditionalData == nil {
		block.AdditionalData = &common.AdditionalData{
			ExtraData: make(map[string][]byte),
		}
	}

	serializeMember, err := consensus.singer.GetMember()
	if err != nil {
		consensus.logger.Fatalf("[%x] get serialize member failed: %v", consensus.Id, err)
		return
	}

	signature := &common.EndorsementEntry{
		Signer:    serializeMember,
		Signature: sig,
	}

	additionalData := AdditionalData{
		Signature: mustMarshal(signature),
	}

	data, _ := json.Marshal(additionalData)
	block.AdditionalData.ExtraData[protocol.RAFTAddtionalDataKey] = data
	data = mustMarshal(block)
	consensus.logger.Debugf("[%x] propose block height：%+v", consensus.Id, block.Header.BlockHeight)
	if err := consensus.node.Propose(context.TODO(), data); err != nil {
		consensus.logger.Panicf("[%x] propose error: %v", consensus.Id, err)
	}
	consensus.proposedIndex = block.Header.BlockHeight
}

func (consensus *ConsensusRaftImpl) processWalAndSnap(ready etcdraft.Ready) {
	if !etcdraft.IsEmptyHardState(ready.HardState) || len(ready.Entries) != 0 {
		consensus.logger.Infof("[%x]Save wal: %v", consensus.Id, describeReady(ready))
		if err := consensus.wal.Save(ready.HardState, ready.Entries); err != nil {
			consensus.logger.Panicf("[%x] save wal: %v, error: %v", consensus.Id, describeReady(ready), err)
		}
	}
	if !etcdraft.IsEmptySnap(ready.Snapshot) {
		if err := consensus.saveSnap(ready.Snapshot); err != nil {
			consensus.logger.Panicf("[%x] save snap error: %v", consensus.Id, err)
		}
		if err := consensus.raftStorage.ApplySnapshot(ready.Snapshot); err != nil {
			consensus.logger.Panicf("[%x] apply snapshot error: %v", consensus.Id, err)
		}
		consensus.publishSnapshot(ready.Snapshot)
	}
	if err := consensus.raftStorage.Append(ready.Entries); err != nil {
		consensus.logger.Panicf("[%x] storage append entries error: %v", consensus.Id, err)
	}
	consensus.sendMessages(ready.Messages)
	consensus.node.Advance()
}

func (consensus *ConsensusRaftImpl) AsyncWalSave() {
	for {
		select {
		case item := <-consensus.walSaveC:
			if ready, ok := item.(etcdraft.Ready); ok {
				consensus.processWalAndSnap(ready)
			} else if configChanged, ok := item.(bool); ok {
				consensus.maybeTriggerSnapshot(configChanged)
			} else {
				consensus.logger.Panicf("[%x] AsyncWalSave got an invalid item: %v", item)
			}
		case <-consensus.closeC:
			consensus.wg.Done()
			return
		}
	}
}

func (consensus *ConsensusRaftImpl) entriesToApply(ents []raftpb.Entry) (nents []raftpb.Entry) {
	if len(ents) == 0 {
		return ents
	}

	firstIdx := ents[0].Index
	if firstIdx > consensus.appliedIndex+1 {
		consensus.logger.Fatalf("first index of committed entry[%d] should <= progress.appliedIndex[%d]+1",
			firstIdx, consensus.appliedIndex)
	}
	consensus.logger.Debugf("appliedIndex: %d, firstIndex: %d, entry num: %d", consensus.appliedIndex, firstIdx, len(ents))
	if consensus.appliedIndex-firstIdx+1 < uint64(len(ents)) {
		nents = ents[consensus.appliedIndex-firstIdx+1:]
	}
	return nents
}

func (consensus *ConsensusRaftImpl) publishEntries(ents []raftpb.Entry) (ok bool, configChanged bool) {
	configChanged = false
	for i := range ents {
		consensus.logger.Debugf("publishEntries term: %d, index: %d, type: %v",
			ents[i].Term, ents[i].Index, ents[i].Type)
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				break
			}
			block := new(common.Block)
			mustUnmarshal(ents[i].Data, block)
			consensus.logger.Debugf("publishEntries term: %d, index: %d, block(%d-%x)",
				ents[i].Term, ents[i].Index, block.Header.BlockHeight, block.Header.BlockHash)
			consensus.commitBlock(block)
			if utils.IsConfBlock(block) {
				consensus.processConfigChange()
			}

		case raftpb.EntryConfChange:
			configChanged = true
			var cc raftpb.ConfChange
			if err := cc.Unmarshal(ents[i].Data); err != nil {
				consensus.logger.Panicf("[%x] unmarshal config change error: %v", consensus.Id, err)
			}
			consensus.confState = *consensus.node.ApplyConfChange(cc)
			var idToNodes map[uint64]string
			consensus.peers, idToNodes = consensus.getPeersFromChainConf()
			for id, node := range idToNodes {
				consensus.idToNodeId.Store(id, node)
			}
			switch cc.Type {
			// todo. may be check the delete node logic
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == consensus.Id {
					consensus.appliedIndex = ents[i].Index
					return false, configChanged
				}
			}
		}
		consensus.appliedIndex = ents[i].Index
	}
	return true, configChanged
}

func (consensus *ConsensusRaftImpl) publishSnapshot(snapshot raftpb.Snapshot) {
	if etcdraft.IsEmptySnap(snapshot) {
		return
	}

	consensus.logger.Infof("publishSnapshot metadata: %v", snapshot.Metadata)
	consensus.confState = snapshot.Metadata.ConfState
	consensus.snapshotIndex = snapshot.Metadata.Index
	if snapshot.Metadata.Index > consensus.appliedIndex {
		consensus.appliedIndex = snapshot.Metadata.Index
	}

	snapshotData := &SnapshotHeight{}
	json.Unmarshal(snapshot.Data, snapshotData)

	for {
		// Loop until catch up to snapshotData.Height from Sync module
		current, _ := consensus.ledgerCache.CurrentHeight()
		consensus.logger.Debugf("publishSnapshot current height: %d, snapshot height: %d", current, snapshotData.Height)
		if current >= snapshotData.Height {
			break
		}
		time.Sleep(500 * time.Microsecond)
	}
}

func (consensus *ConsensusRaftImpl) getSnapshot() ([]byte, error) {
	height, err := consensus.ledgerCache.CurrentHeight()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(SnapshotHeight{
		Height: height,
	})
	consensus.logger.Infof("getSnapshot data: %s", data)
	return data, err
}

func (consensus *ConsensusRaftImpl) maybeTriggerSnapshot(configChanged bool) {
	if consensus.appliedIndex-consensus.snapshotIndex <= consensus.snapCount && !configChanged {
		return
	}

	data, err := consensus.getSnapshot()
	if err != nil {
		consensus.logger.Fatalf("get snapshot error: %v", err)
	}

	snap, err := consensus.raftStorage.CreateSnapshot(consensus.appliedIndex, &consensus.confState, data)
	if err != nil {
		consensus.logger.Fatalf("create snapshot error: %v", err)
	}

	if err := consensus.saveSnap(snap); err != nil {
		consensus.logger.Fatalf("save snapshot error: %v", err)
	}

	consensus.logger.Infof("trigger snapshot appliedIndex: %v, data: %v, snapshotIndex: %v",
		consensus.appliedIndex, string(data), consensus.snapshotIndex)
	//consensus.snapshotIndex = consensus.appliedIndex
	consensus.snapshotIndex = snap.Metadata.Index

	if consensus.snapshotIndex < snapshotCatchUpEntriesN+1 {
		return
	}

	compactIndex := consensus.snapshotIndex - snapshotCatchUpEntriesN
	first, _ := consensus.raftStorage.FirstIndex()
	if compactIndex <= first {
		return
	}
	if err := consensus.raftStorage.Compact(compactIndex); err != nil {
		last, _ := consensus.raftStorage.LastIndex()
		consensus.logger.Fatalf("compact snapshot error: %v, compact "+
			"index: %d, first: %d, last: %d", err, compactIndex, first, last)
	}

	consensus.logger.Infof("compact entries compactIndex:%d", compactIndex)
}

func (consensus *ConsensusRaftImpl) sendMessages(msgs []raftpb.Message) {
	for _, m := range msgs {
		if m.To == 0 {
			consensus.logger.Errorf("send message to 0")
			continue
		}

		consensus.logger.DebugDynamic(func() string {
			return fmt.Sprintf("[%x] send message %v", consensus.Id, describeMessage(m))
		})

		value, ok := consensus.idToNodeId.Load(m.To)
		if !ok {
			consensus.logger.Errorf("send message to %v without net connection", m.To)
		} else {
			netId, ok := value.(string)
			if !ok {
				consensus.logger.Errorf("wrong type in idToNodeId")
				continue
			}
			data, err := m.Marshal()
			if err != nil {
				consensus.logger.Errorf("marshal message error: %v", err)
				continue
			}
			netMsg := &netpb.NetMsg{
				Payload: data,
				Type:    netpb.NetMsg_CONSENSUS_MSG,
				To:      netId,
			}
			consensus.msgbus.Publish(msgbus.SendConsensusMsg, netMsg)
		}
	}
}

func (consensus *ConsensusRaftImpl) loadSnapshot() *raftpb.Snapshot {
	snapshot, err := consensus.snapshotter.Load()
	if err != nil && err != snap.ErrNoSnapshot {
		consensus.logger.Fatalf("load snapshot error: %v", err)
	}
	if snapshot == nil {
		consensus.logger.Infof("loadSnapshot snapshot is nil")
	} else {
		consensus.logger.Infof("loadSnapshot snapshot metadata index: %v", snapshot.Metadata.Index)
	}
	return snapshot
}

func (consensus *ConsensusRaftImpl) replayWAL() *wal.WAL {
	if !wal.Exist(consensus.waldir) {
		if err := os.Mkdir(consensus.waldir, 0750); err != nil {
			consensus.logger.Fatalf("cannot create wal dir: %v", err)
		}

		w, err := wal.Create(consensus.logger.Logger().Desugar(), consensus.waldir, nil)
		if err != nil {
			consensus.logger.Fatalf("create wal error: %v", err)
		}
		w.Close()
	}

	snapshot := consensus.loadSnapshot()

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}

	w, err := wal.Open(consensus.logger.Logger().Desugar(), consensus.waldir, walsnap,
		consensus.chainConf.ChainConfig().Block.BlockSize*1024*1024)
	if err != nil {
		consensus.logger.Fatalf("open wal error: %v", err)
	}
	_, state, ents, err := w.ReadAll()
	if err != nil {
		consensus.logger.Fatalf("read wal error: %v", err)
	}
	consensus.raftStorage = etcdraft.NewMemoryStorage()
	if snapshot != nil {
		if err := consensus.raftStorage.ApplySnapshot(*snapshot); err != nil {
			consensus.logger.Panicf("[%x] apply snapshot error: %v", consensus.Id, err)
		}
	}
	if err := consensus.raftStorage.SetHardState(state); err != nil {
		consensus.logger.Panicf("[%x] SetHardState error: %v", consensus.Id, err)
	}
	if err := consensus.raftStorage.Append(ents); err != nil {
		consensus.logger.Panicf("[%x] storage append error: %v", consensus.Id, err)
	}
	consensus.logger.Infof("replayWAL walsnap index: %v, len(ents): %v", walsnap.Index, len(ents))
	return w
}

func (consensus *ConsensusRaftImpl) commitBlock(block *common.Block) {
	for {
		err := consensus.blockVerifier.VerifyBlock(block, protocol.CONSENSUS_VERIFY)
		consensus.logger.Debugf("verify block: %d-%x error: %v", block.Header.BlockHeight, block.Header.BlockHash, err)
		if err == nil {
			break
		}
		if err == commonErrors.ErrBlockHadBeenCommited {
			return
		} else if err != nil {
			consensus.logger.Errorf("verify block: %d-%x fail: %s", block.Header.BlockHeight, block.Header.BlockHash, err)
			time.Sleep(time.Millisecond * 10)
		}
	}

	err := consensus.blockCommitter.AddBlock(block)
	consensus.logger.Debugf("commit block: %d-%x error: %v", block.Header.BlockHeight, block.Header.BlockHash, err)
	if err != nil && err != commonErrors.ErrBlockHadBeenCommited {
		consensus.logger.Fatalf("commit block: %d-%x fail: %s", block.Header.BlockHeight, block.Header.BlockHash, err)
	}
}

func (consensus *ConsensusRaftImpl) sendProposeState(isProposer bool) {
	consensus.logger.Infof("sendProposeState isProposer: %v", isProposer)
	consensus.msgbus.PublishSafe(msgbus.ProposeState, isProposer)
}

// Verify implements interface of struct Verifier,
// This interface is used to verify the validity of parameters,
// it executes before consensus.
func (consensus *ConsensusRaftImpl) Verify(
	consensusType consensuspb.ConsensusType,
	chainConfig *config.ChainConfig) error {
	return nil
}

func (consensus *ConsensusRaftImpl) getPeersFromChainConf() ([]uint64, map[uint64]string) {
	var (
		peers      []uint64
		idToNodeId = make(map[uint64]string)
		builder    strings.Builder
	)

	fmt.Fprintf(&builder, "[")
	for _, org := range consensus.chainConf.ChainConfig().Consensus.Nodes {
		for _, nodeId := range org.NodeId {
			id := computeRaftIdFromNodeId(nodeId)
			idToNodeId[id] = nodeId
			peers = append(peers, id)
			fmt.Fprintf(&builder, "%s: %x, ", nodeId, id)
		}
	}
	fmt.Fprintf(&builder, "]")

	consensus.logger.InfoDynamic(func() string {
		return fmt.Sprintf("[%x] getPeersFromChainConf peers: %v", consensus.Id, builder.String())
	})
	sort.Slice(peers, func(i, j int) bool {
		return peers[i] < peers[j]
	})
	return peers, idToNodeId
}

func (consensus *ConsensusRaftImpl) processConfigChange() bool {
	peers, idToNodes := consensus.getPeersFromChainConf()
	removed, added := computeUpdatedNodes(consensus.peers, peers)
	consensus.logger.Debugf("[%x] processConfigChange removed: %v, added: %v", consensus.Id, removed, added)

	if consensus.isLeader {
		for _, node := range removed {
			cc := raftpb.ConfChange{
				Type:   raftpb.ConfChangeRemoveNode,
				NodeID: node,
			}
			consensus.confChangeC <- cc
		}
		for _, node := range added {
			cc := raftpb.ConfChange{
				Type:   raftpb.ConfChangeAddNode,
				NodeID: node,
			}
			consensus.confChangeC <- cc
		}
		consensus.peers = peers
		for id, node := range idToNodes {
			consensus.idToNodeId.Store(id, node)
		}
	}
	return len(removed) != 0 || len(added) != 0
}

// VerifyBlockSignatures verifies whether the signatures in block
// is qulified with the consensus algorithm. It should return nil
// error when verify successfully, and return corresponding error
// when failed.
func VerifyBlockSignatures(block *common.Block) error {
	if block == nil || block.Header == nil ||
		block.AdditionalData == nil || block.AdditionalData.ExtraData == nil {
		return fmt.Errorf("invalid block")
	}
	byt, ok := block.AdditionalData.ExtraData[protocol.RAFTAddtionalDataKey]
	if !ok {
		return fmt.Errorf("block.AdditionalData.ExtraData[RAFTAddtionalDataKey] not exist")
	}

	additionalData := &AdditionalData{}
	json.Unmarshal(byt, additionalData)

	endorsement := new(common.EndorsementEntry)
	mustUnmarshal(additionalData.Signature, endorsement)

	if !bytes.Equal(block.Header.Signature, endorsement.Signature) {
		return fmt.Errorf("block.AdditionalData.ExtraData[RAFTAddtionalDataKey] not exist")
	}
	return nil
}

func computeRaftIdFromNodeId(nodeId string) uint64 {
	return uint64(binary.BigEndian.Uint64([]byte(nodeId[len(nodeId)-8:])))
}

func computeUpdatedNodes(oldSet, newSet []uint64) (removed []uint64, added []uint64) {
	removedSet, addedSet := funk.Difference(oldSet, newSet)

	return removedSet.([]uint64), addedSet.([]uint64)
}
