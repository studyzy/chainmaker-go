package chainedbft

import (
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	"github.com/gogo/protobuf/proto"
	"github.com/tidwall/wal"
)

func (cbi *ConsensusChainedBftImpl) saveWalEntry(msgType chainedbftpb.MessageType, msg *chainedbftpb.ConsensusMsg) {
	lastIndex, err := cbi.wal.LastIndex()
	if err != nil {
		cbi.logger.Fatalf("get lastWrite index from walFile failed, reason: %s", err)
	}
	cbi.logger.Debugf("save walEntry index: %d", lastIndex+1)
	bz, err := proto.Marshal(&chainedbftpb.WalEntry{MsgType: msgType, Msg: msg, LastSnapshotIndex: cbi.lastCommitWalIndex})
	if err != nil {
		cbi.logger.Fatalf("json marshal msg failed, reason: %s, msgType: %s, msgContent:%v", err, msgType, msg)
	}
	if err := cbi.wal.Write(lastIndex+1, bz); err != nil {
		cbi.logger.Fatalf("json marshal msg failed, reason: %s, msgType: %s, msgContent:%v", err, msgType, msg)
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
		cbi.logger.Fatalf("json unmarshal failed, reason: %s", err)
	}
	cbi.logger.Infof("lastCommitIndex: %d", msg.LastSnapshotIndex)
	cbi.lastCommitWalIndex = msg.LastSnapshotIndex
	for index := msg.LastSnapshotIndex; index <= lastIndex; index++ {
		data, err := cbi.wal.Read(index)
		if err != nil {
			cbi.logger.Fatalf("read content from wal file failed, readIndex: %d, reason: %s", index, err)
		}
		if err := proto.Unmarshal(data, &msg); err != nil {
			cbi.logger.Fatalf("json unmarshal failed, reason: %s", err)
		}
		switch msg.Msg.Payload.Type {
		case chainedbftpb.MessageType_ProposalMessage:
			if err := cbi.processProposal(msg.Msg); err == nil {
				cbi.addProposalWalIndexByReplay(msg.Msg.Payload.GetProposalMsg().ProposalData.Height, index)
			}
		case chainedbftpb.MessageType_VoteMessage:
			cbi.processVote(msg.Msg)
		}
	}
	cbi.logger.Infof("end replay wal")
	return true
}

func (cbi *ConsensusChainedBftImpl) updateWalIndexAndTruncFile(commitHeight int64) {
	var nextProposalIndex uint64
	if val, exist := cbi.proposalWalIndex.Load(uint64(commitHeight + 1)); exist {
		nextProposalIndex = val.(uint64)
	} else {
		cbi.proposalWalIndex.Range(func(key, value interface{}) bool {
			cbi.logger.Debugf("updateWalIndexAndTruncFile proposalHeight: %v, walIndex: %v", key, value)
			return true
		})
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
	defer func() {
		cbi.logger.Debugf("store proposal: %d walIndex: %d", proposalHeight, lastIndex)
	}()
	if _, exist := cbi.proposalWalIndex.Load(proposalHeight); !exist {
		if lastIndex, err = cbi.wal.LastIndex(); err != nil {
			cbi.logger.Fatalf("get lastIndex from walFile failed, reason: %s", err)
		} else {
			cbi.logger.Debugf("set proposalHeight walIndex: %d", lastIndex+1)
			cbi.proposalWalIndex.Store(proposalHeight, lastIndex+1)
		}
	}
}

func (cbi *ConsensusChainedBftImpl) addProposalWalIndexByReplay(proposalHeight, walIndex uint64) {
	if _, exist := cbi.proposalWalIndex.Load(proposalHeight); !exist {
		cbi.proposalWalIndex.Store(proposalHeight, walIndex)
	}
}
