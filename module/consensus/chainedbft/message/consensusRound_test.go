/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package message

import (
	"testing"

	"chainmaker.org/chainmaker/utils/v2"

	"github.com/stretchr/testify/require"

	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

func TestCheckVoteDoneWithBlock2(t *testing.T) {
	levelVotes := newConsensusRound(4, 1)

	// add two votes by node1;
	// node1 first vote block; second vote newView
	// 1. blk = 1
	// 2. blk + newView = 2
	// 3. blk + newView = 1
	// 4. blk = 2
	// 5. blk + newView = 2
	// 6. blk + newView = 3
	blkID := []byte(utils.GetRandTxId())
	node1VoteBlk := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockId:   blkID,
			AuthorIdx: 1,
		},
	}
	node2VoteBlkAndTimeOut := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			NewView:   true,
			BlockId:   blkID,
			AuthorIdx: 2,
		},
	}
	node1VoteBlkAndTimeOut := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockId:   blkID,
			NewView:   true,
			AuthorIdx: 1,
		},
	}
	node2VoteBlk := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockId:   blkID,
			AuthorIdx: 2,
		},
	}
	node2VoteBlkAndTimeOut2 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockId:   blkID,
			NewView:   true,
			AuthorIdx: 2,
		},
	}
	node3VoteBlkAndTimeOut := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockId:   blkID,
			NewView:   true,
			AuthorIdx: 3,
		},
	}
	add, err := levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &node1VoteBlk},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &node2VoteBlkAndTimeOut},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &node1VoteBlkAndTimeOut},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// add two votes by node2;
	// node2 first vote block; second vote newView
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &node2VoteBlk},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &node2VoteBlkAndTimeOut2},
		},
	}, 3)
	require.False(t, add, "add vote failed")
	require.NoError(t, err, "shouldn't error")

	_, _, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.False(t, done)

	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &node3VoteBlkAndTimeOut},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	doneBlkID, isNewView, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.False(t, isNewView)
	require.True(t, done)
	require.EqualValues(t, doneBlkID, blkID)

	votes := levelVotes.getQCVotes(1)
	require.EqualValues(t, 3, len(votes))
}

func TestCheckVoteDoneWithBlock(t *testing.T) {
	levelVotes := newConsensusRound(4, 1)

	// 1. add newView vote1 with level1
	voteNewView1 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			NewView:   true,
			AuthorIdx: 1,
		},
	}
	add, err := levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteNewView1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 2. add BlockId vote2 with level1
	voteBlock1 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 2, BlockId: []byte(utils.GetRandTxId()),
		},
	}
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteBlock1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 3. add BlockId vote3 with level1
	voteBlock2 := voteBlock1
	voteBlock2.VoteData.AuthorIdx = 3
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteBlock2},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 4. check vote done should be false
	_, _, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.False(t, done, "should not be done")

	// 5. add BlockId vote4 with level1
	voteBlock3 := voteBlock1
	voteBlock3.VoteData.AuthorIdx = 4
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteBlock3},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 7. check vote done should be false
	voteBlockId, voteNewView, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.True(t, done, "should be done")
	require.False(t, voteNewView, "should vote newview")
	require.EqualValues(t, voteBlockId, voteBlock2.VoteData.BlockId)
}

func TestCheckVoteDoneWithNewView(t *testing.T) {
	levelVotes := newConsensusRound(4, 1)

	// 1. add newView vote1 with level1
	voteNewView1 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			NewView:   true,
			AuthorIdx: 1,
		},
	}
	add, err := levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteNewView1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 2. add newView vote2 with level1
	voteNewView2 := voteNewView1
	voteNewView2.VoteData.AuthorIdx = 2
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteNewView2},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 3. check vote done should be false
	_, _, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.False(t, done, "should not be done")

	// 4. add BlockId vote3 with level1
	voteBlock := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 3, BlockId: []byte(utils.GetRandTxId()),
		},
	}
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteBlock},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 5. check vote done should be false
	_, _, done = levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.False(t, done, "should not be done")

	// 6. add newView vote4 with level1
	voteNewView3 := voteNewView1
	voteNewView3.VoteData.AuthorIdx = 4
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteNewView3},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 7. check vote done should be false
	voteBlockId, voteNewView, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VOTE_MESSAGE)
	require.True(t, done, "should be done")
	require.True(t, voteNewView, "should vote newview")
	require.Nil(t, voteBlockId, "should BlockId is null")
}

func TestInsertVote(t *testing.T) {
	levelVotes := newConsensusRound(4, 1)

	// 1. add newView vote1 with level1
	voteNewView1 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			NewView:   true,
			AuthorIdx: 1,
		},
	}
	add, err := levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteNewView1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 2. add same vote should error
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteNewView1},
		},
	}, 3)
	require.False(t, add, "add vote failed")
	require.NoError(t, err, "shouldn be add  error")

	// add different vote in same level and same author
	voteBlock := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 1, BlockId: []byte(utils.GetRandTxId()),
		},
	}
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VOTE_MESSAGE,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{
				VoteMsg: &voteBlock},
		},
	}, 3)
	require.True(t, add, "add vote failed")
	require.NoError(t, err, "shouldn be add vote error")
}

func TestInsertProposal(t *testing.T) {
	levelVotes := newConsensusRound(4, 1)

	// Each Consensus Level allows only one proposal to be added

	// 1. add first proposal in level1
	add, err := levelVotes.insertProposal(1, &chainedbftpb.ConsensusMsg{})
	require.True(t, add, "add proposal success")
	require.NoError(t, err, "shouldn't error")

	// 2. add two proposal in level1
	add, err = levelVotes.insertProposal(1, &chainedbftpb.ConsensusMsg{})
	require.False(t, add, "add proposal failed")
	require.Error(t, err, "shouldn be add proposal error")
}
