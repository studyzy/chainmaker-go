/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package message

import (
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

//consensusRound caches consensus msg per round
type consensusRound struct {
	size   int                                             //The size of validators
	height uint64                                          //Bound to a block height
	msgs   map[uint64]map[chainedbft.MessageType]*votePool //format: [round][Type] = votePool;
	// store all votes(proposal/newView) msg
	proposals map[uint64]*chainedbft.ConsensusMsg //format: [round] = ConsensusMsg; only store proposal msg
}

//newConsensusRound initializes a consensus round with given params
func newConsensusRound(size int, height uint64) *consensusRound {
	return &consensusRound{
		size:      size,
		height:    height,
		msgs:      make(map[uint64]map[chainedbft.MessageType]*votePool),
		proposals: make(map[uint64]*chainedbft.ConsensusMsg),
	}
}

//checkAnyVotes checks whether self have received any minVotesForQc votes with given round and voteType
func (cr *consensusRound) checkAnyVotes(round uint64, voteType chainedbft.MessageType, minVotesForQc int) bool {
	roundMsgs, ok := cr.msgs[round]
	if !ok {
		return false
	}
	votes, ok := roundMsgs[voteType]
	if !ok {
		return false
	}
	return len(votes.votes) >= minVotesForQc
}

//insertVote inserts a vote msg to vote pool
func (cr *consensusRound) insertVote(round uint64, msg *chainedbft.ConsensusMsg, minVotesForQc int) (bool, error) {
	if _, ok := cr.msgs[round]; !ok {
		cr.msgs[round] = make(map[chainedbft.MessageType]*votePool)
		cr.msgs[round][chainedbft.MessageType_VOTE_MESSAGE] = newVotePool(cr.size)
	}
	roundMsgs := cr.msgs[round]
	return roundMsgs[msg.Payload.Type].insertVote(msg, minVotesForQc)
}

//insertProposal inserts a proposal to proposal list
func (cr *consensusRound) insertProposal(round uint64, msg *chainedbft.ConsensusMsg) (bool, error) {
	if _, ok := cr.proposals[round]; ok {
		return false, fmt.Errorf("duplicated proposal message")
	}
	cr.proposals[round] = msg
	return true, nil
}

//getProposal returns a proposal at round
func (cr *consensusRound) getProposal(round uint64) *chainedbft.ConsensusMsg {
	return cr.proposals[round]
}

//getVotes returns all of votes at given round
func (cr *consensusRound) getQCVotes(round uint64) []*chainedbft.VoteData {
	if _, ok := cr.msgs[round]; !ok {
		return nil
	}
	return cr.msgs[round][chainedbft.MessageType_VOTE_MESSAGE].getQCVotes()
}

//getLastValidRound returns the latest valid round at which enough votes received
func (cr *consensusRound) getLastValidRound() int64 {
	lastValidRound := int64(-1)
	for round := range cr.msgs {
		_, _, done := cr.checkVoteDone(round, chainedbft.MessageType_VOTE_MESSAGE)
		if done && lastValidRound < int64(round) {
			lastValidRound = int64(round)
		}
	}
	return lastValidRound
}

//checkVoteDone checks whether self have received enough votes with given vote type at round
func (cr *consensusRound) checkVoteDone(round uint64,
	voteType chainedbft.MessageType) (blkID []byte, isNewView bool, done bool) {
	if _, ok := cr.msgs[round]; !ok {
		return nil, false, false
	}
	return cr.msgs[round][voteType].checkVoteDone()
}
