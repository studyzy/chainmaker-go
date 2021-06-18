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
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thoas/go-funk"

	"chainmaker.org/chainmaker-go/chainconf"
	commonErrors "chainmaker.org/chainmaker-go/common/errors"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	netpb "chainmaker.org/chainmaker-go/pb/protogo/net"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"go.etcd.io/etcd/client/pkg/v3/fileutil"
	etcdraft "go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/etcd/server/v3/wal"
	"go.etcd.io/etcd/server/v3/wal/walpb"
)

var (
	DefaultChanCap          = 1000
	walDir                  = "raftwal"
	snapDir                 = "snap"
	snapCount               = uint64(10)
	snapshotCatchUpEntriesN = uint64(5)
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
	logger        *Logger
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
	snapdir       string
	snapshotter   *snap.Snapshotter
	confState     raftpb.ConfState
	snapshotIndex uint64
	appliedIndex  uint64
	idToNodeId    map[uint64]string

	proposedBlockC chan *common.Block
	verifyResultC  chan *consensus.VerifyResult
	blockInfoC     chan *common.BlockInfo
	confChangeC    chan raftpb.ConfChange
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
	consensus.logger = NewLogger(lg.Logger())
	consensus.chainID = config.ChainID
	consensus.singer = config.Singer
	consensus.ac = config.Ac
	consensus.ledgerCache = config.LedgerCache
	consensus.chainConf = config.ChainConf
	consensus.msgbus = config.MsgBus
	consensus.closeC = make(chan struct{})
	consensus.Id = computeRaftIdFromNodeId(config.NodeId)
	consensus.waldir = path.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, consensus.chainID, walDir)
	consensus.snapdir = path.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, consensus.chainID, snapDir)
	consensus.idToNodeId = make(map[uint64]string)

	consensus.proposedBlockC = make(chan *common.Block, DefaultChanCap)
	consensus.verifyResultC = make(chan *consensuspb.VerifyResult, DefaultChanCap)
	consensus.blockInfoC = make(chan *common.BlockInfo, DefaultChanCap)
	consensus.confChangeC = make(chan raftpb.ConfChange, DefaultChanCap)
	consensus.blockVerifier = config.BlockVerifier
	consensus.blockCommitter = config.BlockCommitter

	consensus.logger.Infof("New ConsensusRaftImpl[%x]", consensus.Id)
	return consensus, nil
}

// Start starts the raft instance
func (consensus *ConsensusRaftImpl) Start() error {
	consensus.logger.Infof("ConsensusRaftImpl[%x] starting", consensus.Id)
	if !fileutil.Exist(consensus.snapdir) {
		if err := os.Mkdir(consensus.snapdir, 0750); err != nil {
			consensus.logger.Fatalf("[%x] cannot create dir for snapshot: %v", consensus.Id, err)
			return err
		}
	}
	consensus.snapshotter = snap.New(consensus.logger.SugaredLogger.Desugar(), consensus.snapdir)
	walExist := wal.Exist(consensus.waldir)
	consensus.wal = consensus.replayWAL()

	consensus.peers = consensus.getPeersFromChainConf()
	c := &etcdraft.Config{
		ID:              consensus.Id,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         consensus.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		// CheckQuorum:     true,
		Logger: consensus.logger,
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
	go consensus.serve()
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.RecvConsensusMsg, consensus)
	chainconf.RegisterVerifier(consensus.chainID, consensuspb.ConsensusType_RAFT, consensus)

	return nil
}

// Start stops the raft instance
func (consensus *ConsensusRaftImpl) Stop() error {
	consensus.logger.Infof("ConsensusRaftImpl stopping")
	close(consensus.closeC)
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
			raftMsg.Unmarshal(msg.Payload)
			consensus.logger.Debugf("[%x] receive message %v", consensus.Id, describeMessage(raftMsg))
			consensus.node.Step(context.Background(), raftMsg)
		} else {
			panic(fmt.Errorf("receive message failed, error message type"))
		}
	}
}

func (consensus *ConsensusRaftImpl) OnQuit() {
	// do nothing
	//panic("implement me")
}

func (consensus *ConsensusRaftImpl) saveSnap(snap raftpb.Snapshot) error {
	consensus.logger.Infof("saveSnap %v", describeSnapshot(snap))
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
	return consensus.wal.ReleaseLockTo(snap.Metadata.Index)
}

func (consensus *ConsensusRaftImpl) serve() error {
	snap, err := consensus.raftStorage.Snapshot()
	if err != nil {
		consensus.logger.Fatalf("[%x] raftStorage Snapshot error", consensus.Id, err)
	}
	consensus.confState = snap.Metadata.ConfState
	consensus.snapshotIndex = snap.Metadata.Index
	consensus.appliedIndex = snap.Metadata.Index

	// block := consensus.ledgerCache.GetLastCommittedBlock()
	// if block.AdditionalData != nil {
	//   additionalData := &AdditionalData{}
	//   json.Unmarshal(block.AdditionalData.ExtraData[protocol.RAFTAddtionalDataKey], additionalData)
	//   consensus.appliedIndex = additionalData.AppliedIndex
	// }
	consensus.logger.Infof("[%x] begin serve with snap: %v, appliedIndex: %v",
		consensus.Id, describeSnapshot(snap), consensus.appliedIndex)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-consensus.closeC:
			return nil
		case <-ticker.C:
			consensus.node.Tick()
			consensus.logger.Debugf("[%x] status: %s", consensus.Id, consensus.node.Status())
		case ready := <-consensus.node.Ready():
			consensus.logger.Debugf("[%x] receive from raft ready, %v", consensus.Id, describeReady(ready))
			consensus.wal.Save(ready.HardState, ready.Entries)
			if !etcdraft.IsEmptySnap(ready.Snapshot) {
				consensus.saveSnap(ready.Snapshot)
				consensus.raftStorage.ApplySnapshot(ready.Snapshot)
				consensus.publishSnapshot(ready.Snapshot)
			}

			consensus.raftStorage.Append(ready.Entries)
			consensus.sendMessages(ready.Messages)
			ok, configChanged := consensus.publishEntries(consensus.entriesToApply(ready.CommittedEntries))
			if !ok {
				return nil
			}
			consensus.maybeTriggerSnapshot(configChanged)
			if ready.SoftState != nil {
				consensus.isLeader = atomic.LoadUint64(&ready.SoftState.Lead) == consensus.Id
			}
			consensus.node.Advance()
			consensus.sendProposeState(consensus.isLeader)

		case block := <-consensus.proposedBlockC:
			// Add hash and signature to block
			hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
			if err != nil {
				consensus.logger.Errorf("[%x] sign block failed, %s", consensus.Id, err)
			}
			block.Header.BlockHash = hash[:]
			block.Header.Signature = sig
			if block.AdditionalData == nil {
				block.AdditionalData = &common.AdditionalData{
					ExtraData: make(map[string][]byte),
				}
			}

			serializeMember, err := consensus.singer.GetSerializedMember(true)
			if err != nil {
				consensus.logger.Errorf("[%x] get serialize member failed: %v", consensus.Id, err)
				return err
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
			consensus.node.Propose(context.TODO(), data)

		case cc := <-consensus.confChangeC:
			consensus.logger.Debugf("[%x] ProposeConfChange %v", consensus.Id, describeConfChange(cc))
			consensus.node.ProposeConfChange(context.TODO(), cc)
		}
	}
}

func (consensus *ConsensusRaftImpl) entriesToApply(ents []raftpb.Entry) (nents []raftpb.Entry) {
	if len(ents) == 0 {
		return ents
	}

	firstIdx := ents[0].Index
	if firstIdx > consensus.appliedIndex+1 {
		consensus.logger.Fatalf("first index of committed entry[%d] should <= progress.appliedIndex[%d]+1", firstIdx, consensus.appliedIndex)
	}
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
			cc.Unmarshal(ents[i].Data)
			consensus.confState = *consensus.node.ApplyConfChange(cc)
			consensus.peers = consensus.getPeersFromChainConf()

			switch cc.Type {
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == consensus.Id {
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

	if snapshot.Metadata.Index <= consensus.appliedIndex {
		consensus.logger.Fatalf("snapshot index: %v should > appliedIndex: %v", snapshot.Metadata.Index, consensus.appliedIndex)
	}

	consensus.logger.Infof("publishSnapshot metadata: %v", snapshot.Metadata)
	consensus.confState = snapshot.Metadata.ConfState
	consensus.snapshotIndex = snapshot.Metadata.Index
	consensus.appliedIndex = snapshot.Metadata.Index

	snapshotData := &SnapshotHeight{}
	json.Unmarshal(snapshot.Data, snapshotData)
	for {
		// Loop until catch up to snapshotData.Height from Sync module
		current, _ := consensus.ledgerCache.CurrentHeight()
		if current > snapshotData.Height {
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
	snapshotData := SnapshotHeight{
		Height: height,
	}

	data, err := json.Marshal(snapshotData)
	consensus.logger.Infof("getSnapshot data: %s", data)
	return data, err
}

func (consensus *ConsensusRaftImpl) maybeTriggerSnapshot(configChanged bool) {
	if consensus.appliedIndex-consensus.snapshotIndex <= snapCount && !configChanged {
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

	compactIndex := uint64(1)
	if consensus.appliedIndex > snapshotCatchUpEntriesN {
		compactIndex = consensus.appliedIndex - snapshotCatchUpEntriesN
	}

	if err := consensus.raftStorage.Compact(compactIndex); err != nil {
		consensus.logger.Fatalf("compact snapshot error: %v", err)
	}

	consensus.snapshotIndex = consensus.appliedIndex
	consensus.logger.Infof("trigger snapshot appliedIndex: %v, data: %v, compactIndex: %v, snapshotIndex: %v",
		consensus.appliedIndex, string(data), compactIndex, consensus.snapshotIndex)
}

func (consensus *ConsensusRaftImpl) sendMessages(msgs []raftpb.Message) {
	for _, m := range msgs {
		if m.To == 0 {
			consensus.logger.Errorf("send message to 0")
			continue
		}

		consensus.logger.Debugf("[%x] send message %v", consensus.Id, describeMessage(m))

		netId, ok := consensus.idToNodeId[m.To]
		if !ok {
			consensus.logger.Errorf("send message to %v without net connection", m.To)
		} else {
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

		w, err := wal.Create(consensus.logger.SugaredLogger.Desugar(), consensus.waldir, nil)
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

	w, err := wal.Open(consensus.logger.SugaredLogger.Desugar(), consensus.waldir, walsnap)
	if err != nil {
		consensus.logger.Fatalf("open wal error: %v", err)
	}
	_, state, ents, err := w.ReadAll()
	if err != nil {
		consensus.logger.Fatalf("read wal error: %v", err)
	}
	consensus.raftStorage = etcdraft.NewMemoryStorage()
	if snapshot != nil {
		consensus.raftStorage.ApplySnapshot(*snapshot)
	}
	consensus.raftStorage.SetHardState(state)
	consensus.raftStorage.Append(ents)
	consensus.logger.Infof("replayWAL walsnap index: %v, len(ents): %v", walsnap.Index, len(ents))
	return w
}

func (consensus *ConsensusRaftImpl) commitBlock(block *common.Block) error {
	for {
		err := consensus.blockVerifier.VerifyBlock(block, protocol.CONSENSUS_VERIFY)
		consensus.logger.Debugf("verify block: %d-%x error: %v", block.Header.BlockHeight, block.Header.BlockHash, err)
		if err == nil {
			break
		}
		if err == commonErrors.ErrBlockHadBeenCommited {
			return nil
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

	return nil
}

func (consensus *ConsensusRaftImpl) sendProposeState(isProposer bool) {
	consensus.logger.Infof("sendProposeState isProposer: %v", isProposer)
	consensus.msgbus.PublishSafe(msgbus.ProposeState, isProposer)
}

// Verify implements interface of struct Verifier,
// This interface is used to verify the validity of parameters,
// it executes before consensus.
func (consensus *ConsensusRaftImpl) Verify(consensusType consensuspb.ConsensusType, chainConfig *config.ChainConfig) error {
	return nil
}

func (consensus *ConsensusRaftImpl) getPeersFromChainConf() []uint64 {
	orgs := consensus.chainConf.ChainConfig().Consensus.Nodes
	peers := []uint64{}
	idToNodeId := make(map[uint64]string)
	var builder strings.Builder
	fmt.Fprintf(&builder, "[")

	for _, org := range orgs {
		if len(org.NodeId) == 1 {
			nodeId := org.NodeId[0]
			id := computeRaftIdFromNodeId(nodeId)
			idToNodeId[id] = nodeId
			peers = append(peers, id)
			fmt.Fprintf(&builder, "%s: %x, ", nodeId, id)
		}
	}
	fmt.Fprintf(&builder, "]")

	consensus.logger.Infof("[%x] getPeersFromChainConf peers: %v", consensus.Id, builder.String())
	consensus.idToNodeId = idToNodeId
	sort.Slice(peers, func(i, j int) bool {
		return peers[i] < peers[j]
	})
	return peers
}

func (consensus *ConsensusRaftImpl) processConfigChange() {
	peers := consensus.getPeersFromChainConf()
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
	}
}

// VerifyBlockSignatures verifies whether the signatures in block
// is qulified with the consensus algorithm. It should return nil
// error when verify successfully, and return corresponding error
// when failed.
func VerifyBlockSignatures(block *common.Block) error {
	if block == nil || block.Header == nil || block.Header.BlockHeight < 0 ||
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
