/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package chainedbft

import (
	"chainmaker.org/chainmaker/common/v2/wal"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"github.com/gogo/protobuf/proto"
)

func (cbi *ConsensusChainedBftImpl) saveWalEntry(msgType chainedbftpb.MessageType, msg *chainedbftpb.ConsensusMsg) {
	lastIndex, err := cbi.wal.LastIndex()
	if err != nil {
		cbi.logger.Fatalf("get lastWrite index from walFile failed, reason: %s", err)
	}
	bz, err := proto.Marshal(&chainedbftpb.WalEntry{MsgType: msgType, Msg: msg, LastSnapshotIndex: cbi.lastCommitWalIndex})
	if err != nil {
		cbi.logger.Fatalf("proto marshal msg failed, reason: %s, msgType: %s, msgContent:%v", err, msgType, msg)
	}
	if err := cbi.wal.Write(lastIndex+1, bz); err != nil {
		cbi.logger.Fatalf("write msg failed, reason: %s, msgType: %s, msgContent:%v", err, msgType, msg)
	}
}

func (cbi *ConsensusChainedBftImpl) replayWal() (hasWalEntry bool) {
	defer func() {
		cbi.doneReplayWal = true
	}()

	cbi.logger.Infof("start replay wal")
	lastIndex, err := cbi.wal.LastIndex()
	if err != nil {
		cbi.logger.Fatalf("get lastWrite index from walFile failed, reason: %s", err)
	}

	data, err := cbi.wal.Read(lastIndex)
	if err == wal.ErrNotFound {
		cbi.logger.Info("no content in wal file")
		return false
	}
	msg := chainedbftpb.WalEntry{}
	if err := proto.Unmarshal(data, &msg); err != nil {
		cbi.logger.Errorf("proto unmarshal failed, reason: %s", err)
		return false
	}
	cbi.logger.Infof("lastIndex: %d,lastCommitIndex: %d", lastIndex, msg.LastSnapshotIndex)
	cbi.lastCommitWalIndex = msg.LastSnapshotIndex
	for index := msg.LastSnapshotIndex; index <= lastIndex; index++ {
		data, err := cbi.wal.Read(index)
		if err != nil {
			cbi.logger.Errorf("read content from wal file failed, readIndex: %d, reason: %s", index, err)
			continue
		}
		if err := proto.Unmarshal(data, &msg); err != nil {
			cbi.logger.Errorf("proto unmarshal failed, reason: %s", err)
			continue
		}
		switch msg.Msg.Payload.Type {
		case chainedbftpb.MessageType_PROPOSAL_MESSAGE:
			if err := cbi.processProposal(msg.Msg); err == nil {
				cbi.addProposalWalIndexByReplay(msg.Msg.Payload.GetProposalMsg().ProposalData.Height, index)
			}
		case chainedbftpb.MessageType_VOTE_MESSAGE:
			cbi.processVote(msg.Msg)
		}
	}
	cbi.logger.Infof("end replay wal")
	return true
}

func (cbi *ConsensusChainedBftImpl) updateWalIndexAndTruncFile(commitHeight uint64) {
	var (
		nextProposalIndex uint64
		ok                bool
	)
	if val, exist := cbi.proposalWalIndex.Load(uint64(commitHeight + 1)); exist {
		if nextProposalIndex, ok = val.(uint64); !ok {
			return
		}
	} else {
		return
	}
	cbi.proposalWalIndex.Delete(commitHeight)
	cbi.logger.Infof("commit block height: %d, nextProposalIndex: %d", commitHeight, nextProposalIndex)
	cbi.lastCommitWalIndex = nextProposalIndex
	if commitHeight%5 == 0 {
		if err := cbi.wal.TruncateFront(cbi.lastCommitWalIndex); err != nil {
			cbi.logger.Fatalf("truncate wal file failed [%d], reason: %s", cbi.lastCommitWalIndex, err)
		}
	}
}

func (cbi *ConsensusChainedBftImpl) addProposalWalIndex(proposalHeight uint64) {
	var (
		err       error
		lastIndex uint64
	)
	if _, exist := cbi.proposalWalIndex.Load(proposalHeight); !exist {
		if lastIndex, err = cbi.wal.LastIndex(); err != nil {
			cbi.logger.Fatalf("get lastIndex from walFile failed, reason: %s", err)
		} else {
			cbi.proposalWalIndex.Store(proposalHeight, lastIndex+1)
			cbi.logger.Debugf("store proposal: %d walIndex: %d", proposalHeight, lastIndex+1)
		}
	}
}

func (cbi *ConsensusChainedBftImpl) addProposalWalIndexByReplay(proposalHeight, walIndex uint64) {
	if _, exist := cbi.proposalWalIndex.Load(proposalHeight); !exist {
		cbi.proposalWalIndex.Store(proposalHeight, walIndex)
	}
}
