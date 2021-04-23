/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync/atomic"
	"time"

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
	"go.etcd.io/etcd/pkg/fileutil"
	etcdraft "go.etcd.io/etcd/raft"
	"go.etcd.io/etcd/raft/raftpb"
	"go.etcd.io/etcd/wal"
	"go.etcd.io/etcd/wal/walpb"

	"go.etcd.io/etcd/etcdserver/api/snap"
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
	node          etcdraft.Node
	raftStorage   *etcdraft.MemoryStorage
	wal           *wal.WAL
	waldir        string
	snapdir       string
	snapshotter   *snap.Snapshotter
	confState     raftpb.ConfState
	snapshotIndex uint64
	appliedIndex  uint64
	idToNetId     map[uint64]string

	proposedBlockC chan *common.Block
	verifyResultC  chan *consensus.VerifyResult
	blockInfoC     chan *common.BlockInfo
	blockVerifier  protocol.BlockVerifier
	blockCommitter protocol.BlockCommitter
}

// ConsensusRaftImplConfig contains initialization config for ConsensusRaftImpl
type ConsensusRaftImplConfig struct {
	ChainID        string
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
	consensus.logger.Infof("New ConsensusRaftImpl[%s]", config.ChainID)
	consensus.chainID = config.ChainID
	consensus.singer = config.Singer
	consensus.ac = config.Ac
	consensus.ledgerCache = config.LedgerCache
	consensus.chainConf = config.ChainConf
	consensus.msgbus = config.MsgBus
	consensus.Id = consensus.detectLocalOrgId()
	consensus.waldir = path.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, consensus.chainID, walDir)
	consensus.snapdir = path.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, consensus.chainID, snapDir)

	consensus.proposedBlockC = make(chan *common.Block, DefaultChanCap)
	consensus.verifyResultC = make(chan *consensuspb.VerifyResult, DefaultChanCap)
	consensus.blockInfoC = make(chan *common.BlockInfo, DefaultChanCap)
	consensus.blockVerifier = config.BlockVerifier
	consensus.blockCommitter = config.BlockCommitter

	return consensus, nil
}

// Start starts the raft instance
func (consensus *ConsensusRaftImpl) Start() error {
	consensus.logger.Infof("ConsensusRaftImpl starting")
	consensus.correlateIdAndNetId()
	if !fileutil.Exist(consensus.snapdir) {
		if err := os.Mkdir(consensus.snapdir, 0750); err != nil {
			consensus.logger.Fatalf("[%d] cannot create dir for snapshot: %v", consensus.Id, err)
			return err
		}
	}
	consensus.snapshotter = snap.New(consensus.logger.SugaredLogger.Desugar(), consensus.snapdir)
	walExist := wal.Exist(consensus.waldir)
	consensus.wal = consensus.replayWAL()

	orgs := consensus.chainConf.ChainConfig().Consensus.Nodes
	peers := make([]etcdraft.Peer, len(orgs))
	for i := range orgs {
		peers[i] = etcdraft.Peer{ID: uint64(i + 1)}
	}
	c := &etcdraft.Config{
		ID:              consensus.Id,
		ElectionTick:    10,
		HeartbeatTick:   5,
		Storage:         consensus.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		// CheckQuorum:     true,
		Logger: consensus.logger,
	}

	if walExist {
		consensus.node = etcdraft.RestartNode(c)
	} else {
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
	return nil
}

// OnMessage receives messages from msgbus
func (consensus *ConsensusRaftImpl) OnMessage(message *msgbus.Message) {
	// consensus.logger.Debugf("OnMessage receive topic: %s", message.Topic)

	switch message.Topic {
	case msgbus.ProposedBlock:
		if block, ok := message.Payload.(*common.Block); ok {
			consensus.proposedBlockC <- block
		}
	case msgbus.RecvConsensusMsg:
		if msg, ok := message.Payload.(*netpb.NetMsg); ok {
			raftMsg := raftpb.Message{}
			raftMsg.Unmarshal(msg.Payload)
			consensus.node.Step(context.Background(), raftMsg)
		} else {
			panic(fmt.Errorf("receive message failed, error message type"))
		}
		// case msgbus.VerifyResult:
		//   if verifyResult, ok := message.Payload.(*consensus.VerifyResult); ok {
		//     consensus.logger.Debugf("verify result: %s", verifyResult.Code)
		//     consensus.verifyResultC <- verifyResult
		//   }
		// case msgbus.BlockInfo:
		//   if blockInfo, ok := message.Payload.(*common.BlockInfo); ok {
		//     consensus.blockInfoC <- blockInfo
		//   } else {
		//     panic(fmt.Errorf("error message type"))
		//   }
	}
}

func (consensus *ConsensusRaftImpl) OnQuit() {
	// do nothing
	//panic("implement me")
}

func (consensus *ConsensusRaftImpl) saveSnap(snap raftpb.Snapshot) error {
	consensus.logger.Infof("saveSnap metadata: %v", snap.Metadata)
	walSnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
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
		consensus.logger.Fatalf("id: %d raftStorage Snapshot error", consensus.Id, err)
	}
	consensus.confState = snap.Metadata.ConfState
	consensus.snapshotIndex = snap.Metadata.Index
	consensus.appliedIndex = snap.Metadata.Index

	block := consensus.ledgerCache.GetLastCommittedBlock()
	if block.AdditionalData != nil {
		additionalData := &AdditionalData{}
		json.Unmarshal(block.AdditionalData.ExtraData[protocol.RAFTAddtionalDataKey], additionalData)
		consensus.appliedIndex = additionalData.AppliedIndex
	}
	consensus.logger.Infof("id: %d begin serve with snapshotIndex: %v, appliedIndex: %v",
		consensus.Id, consensus.snapshotIndex, consensus.appliedIndex)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			consensus.node.Tick()
		case ready := <-consensus.node.Ready():
			consensus.logger.Debugf("id: %d receive from raft, softState: %v, hardState: %v",
				consensus.Id, ready.SoftState, ready.HardState)
			consensus.wal.Save(ready.HardState, ready.Entries)
			if !etcdraft.IsEmptySnap(ready.Snapshot) {
				consensus.saveSnap(ready.Snapshot)
				consensus.raftStorage.ApplySnapshot(ready.Snapshot)
				consensus.publishSnapshot(ready.Snapshot)
			}

			consensus.raftStorage.Append(ready.Entries)
			consensus.sendMessages(ready.Messages)
			consensus.publishEntries(consensus.entriesToApply(ready.CommittedEntries))
			consensus.maybeTriggerSnapshot()
			consensus.node.Advance()
			if ready.SoftState != nil {
				consensus.sendProposeState(atomic.LoadUint64(&ready.SoftState.Lead) == consensus.Id)
			}
		case block := <-consensus.proposedBlockC:
			// Add hash and signature to block
			hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
			if err != nil {
				consensus.logger.Errorf("[%s]sign block failed, %s", consensus.Id, err)
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
				consensus.logger.Errorf("[%d] get serialize member failed: %v", consensus.Id, err)
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

func (consensus *ConsensusRaftImpl) publishEntries(ents []raftpb.Entry) bool {
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

			// add appliedIndex to block
			additionalData := &AdditionalData{}
			json.Unmarshal(block.AdditionalData.ExtraData[protocol.RAFTAddtionalDataKey], additionalData)
			additionalData.AppliedIndex = ents[i].Index
			data, _ := json.Marshal(additionalData)
			block.AdditionalData.ExtraData[protocol.RAFTAddtionalDataKey] = data

			consensus.logger.Debugf("commit block: %d-%x index: %v", block.Header.BlockHeight, block.Header.BlockHash, additionalData.AppliedIndex)
			consensus.commitBlock(block)

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			cc.Unmarshal(ents[i].Data)
			consensus.confState = *consensus.node.ApplyConfChange(cc)
		}

		consensus.appliedIndex = ents[i].Index
	}
	return true
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
	consensus.logger.Infof("getSnapshot data: %v", data)
	return data, err
}

func (consensus *ConsensusRaftImpl) maybeTriggerSnapshot() {
	if consensus.appliedIndex-consensus.snapshotIndex <= snapCount {
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
	if consensus.appliedIndex > compactIndex {
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

		netId, ok := consensus.idToNetId[m.To]
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

func (consensus *ConsensusRaftImpl) detectLocalOrgId() uint64 {
	orgs := consensus.chainConf.ChainConfig().Consensus.Nodes
	orgid := consensus.ac.GetLocalOrgId()

	var id uint64 = 1
	for _, org := range orgs {
		if org.OrgId == orgid {
			return id
		}
		id += 1
	}
	panic(fmt.Errorf("not found org in chainconf"))
}

func (consensus *ConsensusRaftImpl) correlateIdAndNetId() {
	consensus.idToNetId = make(map[uint64]string)
	var id uint64 = 1
	nodes := consensus.chainConf.ChainConfig().Consensus.Nodes
	for _, node := range nodes {
		nid := node.NodeId[0]
		consensus.idToNetId[id] = nid
		id += 1
	}
	consensus.logger.Infof("raft id to netid: %v", consensus.idToNetId)
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

	// consensus.msgbus.PublishSafe(msgbus.VerifyBlock, block)
	// verifyResult := <-consensus.verifyResultC
	// if verifyResult.Code == consensus.VerifyResult_FAIL {
	//   consensus.logger.Fatalf("verify block: %d-%x fail: %s", block.Header.BlockHeight, block.Header.BlockHash, verifyResult.Msg)
	// }

	// consensus.msgbus.PublishSafe(msgbus.CommitBlock, block)
	// blockInfo := <-consensus.blockInfoC

	// if block.Header.BlockHeight != blockInfo.Block.Header.BlockHeight ||
	//   !bytes.Equal(block.Header.BlockHash, blockInfo.Block.Header.BlockHash) {
	//   consensus.logger.Fatalf("commit block: %d-%x unmatch with: %d-%x",
	//     block.Header.BlockHeight, block.Header.BlockHash,
	//     blockInfo.Block.Header.BlockHeight, blockInfo.Block.Header.BlockHash)
	// }

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
