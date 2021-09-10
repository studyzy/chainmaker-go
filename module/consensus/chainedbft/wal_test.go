/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package chainedbft

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker-go/consensus/chainedbft/liveness"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/message"
	"chainmaker.org/chainmaker/common/v2/wal"
	"chainmaker.org/chainmaker/logger/v2"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/stretchr/testify/require"
)

func TestBaseWriteWal(t *testing.T) {
	// 0. create file
	testDir := "test_wal"
	walFile, err := wal.Open(testDir, nil)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	lastIndex, err := walFile.LastIndex()
	require.NoError(t, err)
	firstIndex := lastIndex

	// 1. write data to wal file
	require.NoError(t, walFile.Write(lastIndex+1, []byte("hello1")))
	require.NoError(t, walFile.Write(lastIndex+2, []byte("hello2")))
	require.NoError(t, walFile.Write(lastIndex+3, []byte("hello3")))
	require.NoError(t, walFile.Write(lastIndex+4, []byte("hello4")))

	// 2. read last index
	lastIndex, err = walFile.LastIndex()
	require.NoError(t, err)
	require.EqualValues(t, firstIndex+4, lastIndex)

	// 3. read content from wal file
	count := 0
	var data []byte
	for i := firstIndex + 1; i < lastIndex+1; i++ {
		data, err = walFile.Read(i)
		require.NoError(t, err)
		count++
		fmt.Println(i, ": ", string(data))
	}
	require.EqualValues(t, 4, count)

	data, err = walFile.Read(lastIndex)
	require.NoError(t, err)
	require.EqualValues(t, data, []byte("hello4"))

	// trunc wal file
	require.NoError(t, walFile.TruncateFront(2))
	count = 0
	for i := uint64(2); i < lastIndex+1; i++ {
		data, err := walFile.Read(i)
		require.NoError(t, err)
		count++
		fmt.Println(i, ": ", string(data))
	}
	require.EqualValues(t, 3, count)
}

func TestSaveWal(t *testing.T) {
	cbi := &ConsensusChainedBftImpl{
		smr:     &chainedbftSMR{paceMaker: &liveness.Pacemaker{}},
		msgPool: message.NewMsgPool(10, 10, 3),
	}
	dirPath := filepath.Join("./", "test_chain", WalDirSuffix)
	walFile, err := wal.Open(dirPath, nil)
	defer os.RemoveAll(dirPath)
	require.NoError(t, err)
	cbi.wal = walFile
	cbi.logger = logger.GetLogger("aa")
	cbi.protocolMsgCh = make(chan *chainedbftpb.ConsensusMsg, 2)
	lastIndex, err := cbi.wal.LastIndex()
	require.NoError(t, err)
	require.EqualValues(t, 0, lastIndex)
	cbi.lastCommitWalIndex = 1
	// 1. add entry: vote and proposal
	voteBlock := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 1, BlockId: []byte(utils.GetRandTxId()),
		},
	}
	cbi.saveWalEntry(chainedbftpb.MessageType_PROPOSAL_MESSAGE, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteBlock},
		},
	})

	proposalMsg := &chainedbftpb.ProposalMsg{
		ProposalData: &chainedbftpb.ProposalData{
			Level: 1, Height: 1, Proposer: []byte("nodeId1"), ProposerIdx: 1,
		},
	}
	cbi.saveWalEntry(chainedbftpb.MessageType_PROPOSAL_MESSAGE, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_PROPOSAL_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_ProposalMsg{
				ProposalMsg: proposalMsg},
		},
	})

	// 2. check index
	lastIndex, err = cbi.wal.LastIndex()
	require.NoError(t, err)
	require.EqualValues(t, 2, lastIndex)
	for i := uint64(1); i <= lastIndex; i++ {
		data, err := cbi.wal.Read(i)
		require.NoError(t, err)
		fmt.Println(string(data))
	}

	// 3. replay wal file
	//cbi.replayWal()
	//wg := sync.WaitGroup{}
	//wg.Add(2)
	//go func() {
	//	for {
	//		select {
	//		case msg := <-cbi.protocolMsgCh:
	//			wg.Done()
	//			fmt.Println(msg)
	//		}
	//	}
	//}()
	//wg.Wait()
}
