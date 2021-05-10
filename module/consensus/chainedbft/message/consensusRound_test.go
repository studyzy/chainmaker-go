package message

import (
	"testing"

	"chainmaker.org/chainmaker-go/utils"

	"github.com/stretchr/testify/require"

	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
)

func TestCheckVoteDoneWithBlock2(t *testing.T) {
	levelVotes := newConsensusRound(4, 1)

	// add two votes by node1;
	// node1 first vote block; second vote newView
	blkID := []byte(utils.GetRandTxId())
	node1VoteBlk := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockID:   blkID,
			AuthorIdx: 1,
		},
	}
	node2VoteBlkAndTimeOut := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			NewView:   true,
			BlockID:   blkID,
			AuthorIdx: 2,
		},
	}
	node1VoteBlkAndTimeOut := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockID:   blkID,
			NewView:   true,
			AuthorIdx: 1,
		},
	}
	node2VoteBlk := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockID:   blkID,
			AuthorIdx: 2,
		},
	}
	node2VoteBlkAndTimeOut2 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level:     1,
			Height:    1,
			BlockID:   blkID,
			NewView:   true,
			AuthorIdx: 2,
		},
	}
	add, err := levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&node1VoteBlk},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&node2VoteBlkAndTimeOut},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&node1VoteBlkAndTimeOut},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// add two votes by node2;
	// node2 first vote block; second vote newView
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&node2VoteBlk},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&node2VoteBlkAndTimeOut2},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	_, _, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VoteMessage)
	require.False(t, done)
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
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteNewView1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 2. add blockID vote2 with level1
	voteBlock1 := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 2, BlockID: []byte(utils.GetRandTxId()),
		},
	}
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteBlock1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 3. add blockID vote3 with level1
	voteBlock2 := voteBlock1
	voteBlock2.VoteData.AuthorIdx = 3
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteBlock2},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 4. check vote done should be false
	_, _, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VoteMessage)
	require.False(t, done, "should not be done")

	// 5. add blockID vote4 with level1
	voteBlock3 := voteBlock1
	voteBlock3.VoteData.AuthorIdx = 4
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteBlock3},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 7. check vote done should be false
	voteBlockID, voteNewView, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VoteMessage)
	require.True(t, done, "should be done")
	require.False(t, voteNewView, "should vote newview")
	require.EqualValues(t, voteBlockID, voteBlock2.VoteData.BlockID)
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
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteNewView1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 2. add newView vote2 with level1
	voteNewView2 := voteNewView1
	voteNewView2.VoteData.AuthorIdx = 2
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteNewView2},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 3. check vote done should be false
	_, _, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VoteMessage)
	require.False(t, done, "should not be done")

	// 4. add blockID vote3 with level1
	voteBlock := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 3, BlockID: []byte(utils.GetRandTxId()),
		},
	}
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteBlock},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 5. check vote done should be false
	_, _, done = levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VoteMessage)
	require.False(t, done, "should not be done")

	// 6. add newView vote4 with level1
	voteNewView3 := voteNewView1
	voteNewView3.VoteData.AuthorIdx = 4
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteNewView3},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 7. check vote done should be false
	voteBlockID, voteNewView, done := levelVotes.checkVoteDone(1, chainedbftpb.MessageType_VoteMessage)
	require.True(t, done, "should be done")
	require.True(t, voteNewView, "should vote newview")
	require.Nil(t, voteBlockID, "should blockId is null")
}

func TestInsertWithNet(t *testing.T) {

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
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteNewView1},
		},
	}, 3)
	require.True(t, add, "add vote success")
	require.NoError(t, err, "shouldn't error")

	// 2. add same vote should error
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteNewView1},
		},
	}, 3)
	require.False(t, add, "add vote failed")
	require.NoError(t, err, "shouldn be add  error")

	// add different vote in same level and same author
	voteBlock := chainedbftpb.VoteMsg{
		VoteData: &chainedbftpb.VoteData{
			Level: 1, Height: 1, AuthorIdx: 1, BlockID: []byte(utils.GetRandTxId()),
		},
	}
	add, err = levelVotes.insertVote(1, &chainedbftpb.ConsensusMsg{
		Payload: &chainedbftpb.ConsensusPayload{
			Type: chainedbftpb.MessageType_VoteMessage,
			Data: &chainedbftpb.ConsensusPayload_VoteMsg{&voteBlock},
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
