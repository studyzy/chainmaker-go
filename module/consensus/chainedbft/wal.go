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

func (cbi *ConsensusChainedBftImpl) replayWal() {
	lastIndex, err := cbi.wal.LastIndex()
	if err != nil {
		cbi.logger.Fatalf("get lastWrite index from walFile failed, reason: %s", err)
	}

	data, err := cbi.wal.Read(lastIndex)
	if err == wal.ErrNotFound {
		cbi.logger.Info("no content in wal file")
		return
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
			cbi.logger.Fatalf("read content from wal file failed, reason: %s", err)
		}
		if err := proto.Unmarshal(data, &msg); err != nil {
			cbi.logger.Fatalf("json unmarshal failed, reason: %s", err)
		}
		cbi.onConsensusMsg(msg.Msg)
	}
}

func (cbi *ConsensusChainedBftImpl) updateWalIndexAndTruncFile(commitHeight int64) {
	var nextProposalIndex uint64
	if val, exist := cbi.proposalWalIndex.Load(commitHeight + 1); exist {
		nextProposalIndex = val.(uint64)
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
