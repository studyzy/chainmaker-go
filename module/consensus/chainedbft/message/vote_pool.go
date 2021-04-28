/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package message

import (
	"bytes"
	"fmt"
	"sort"

	chainedbft "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
)

type orderIndexes []uint64

//Len returns the size of orderIndexes
func (oi orderIndexes) Len() int { return len(oi) }

//Swap swaps the ith object with jth object in indexedPeers
func (oi orderIndexes) Swap(i, j int) { oi[i], oi[j] = oi[j], oi[i] }

//Less checks the ith object's index < the jth object's index
func (oi orderIndexes) Less(i, j int) bool { return oi[i] < oi[j] }

//votePool caches the vote msg with all validators for one level
type votePool struct {
	newViewNum    int    //The number of new view voted
	lockedNewView bool   //Indicates whether move to new view
	lockedBlockID []byte //The +2/3 voted for vp block

	votes        map[uint64]*chainedbft.VoteData            //format: [author index] = voteData; store all vote from author
	votedNewView map[uint64]*chainedbft.VoteData            //format: [author index] = voteData; only store newView vote from author
	votedBlockID map[string]map[uint64]*chainedbft.VoteData //format: [block hash][author index] = voteData; only store proposal vote from author
}

//newVotePool initializes a votePool with given params
func newVotePool(size int) *votePool {
	return &votePool{
		newViewNum:    0,
		lockedBlockID: nil,
		lockedNewView: false,
		votes:         make(map[uint64]*chainedbft.VoteData, size),
		votedNewView:  make(map[uint64]*chainedbft.VoteData, size),
		votedBlockID:  make(map[string]map[uint64]*chainedbft.VoteData, size),
	}
}

//insertVote inserts a vote msg
func (vp *votePool) insertVote(msg *chainedbft.ConsensusMsg, minVotesForQc int) (bool, error) {
	voteMsg := msg.Payload.GetVoteMsg()
	if voteMsg == nil {
		return false, fmt.Errorf("nil vote msg")
	}
	return vp.insertVoteData(voteMsg.VoteData, minVotesForQc)
}

func (vp *votePool) insertVoteData(vote *chainedbft.VoteData, minVotesForQc int) (bool, error) {
	if vote == nil {
		return false, fmt.Errorf("nil vote data")
	}
	if err := vp.checkDuplicationVote(vote); err != nil {
		return false, err
	}

	vp.votes[vote.AuthorIdx] = vote
	// process NewView vote
	if vote.NewView {
		vp.votedNewView[vote.AuthorIdx] = vote
		vp.newViewNum++
		if !vp.lockedNewView && vp.newViewNum >= minVotesForQc {
			vp.lockedNewView = true
		}
	}

	// process block vote
	if len(vote.BlockID) == 0 {
		return true, nil
	}
	blockID := string(vote.BlockID)
	if _, ok := vp.votedBlockID[blockID]; !ok {
		vp.votedBlockID[blockID] = make(map[uint64]*chainedbft.VoteData, 1)
	}
	vp.votedBlockID[blockID][vote.AuthorIdx] = vote
	if vp.lockedBlockID == nil && len(vp.votedBlockID[blockID]) >= minVotesForQc {
		//Over 2/3 votes for same block and executed state root
		vp.lockedBlockID = vote.BlockID
	}
	return true, nil
}

func (vp *votePool) checkDuplicationVote(vote *chainedbft.VoteData) error {
	lastVote, ok := vp.votes[vote.AuthorIdx]
	if !ok {
		return nil
	}

	if lastVote.NewView != vote.NewView {
		return fmt.Errorf("authorIdx[%d] has different types of votes for the same level %d, lastVote: %s, newVote: %s",
			vote.AuthorIdx, vote.Level, lastVote, vote)
	} else if len(lastVote.BlockID) > 0 && len(vote.BlockID) > 0 && bytes.Compare(lastVote.BlockID, vote.BlockID) != 0 {
		return fmt.Errorf("authorIdx[%d] has different proposal vote for the same level %d, lastVoteBlockID: %x, "+
			"newVoteBlockID: %x", vote.AuthorIdx, vote.Level, lastVote.BlockID, vote.BlockID)
	}
	return fmt.Errorf("duplicate vote for the newView")
}

//checkVoteDone checks whether a valid block or nil block voted by +2/3 nodes
func (vp *votePool) checkVoteDone() ([]byte, bool, bool) {
	if vp.lockedBlockID != nil {
		return vp.lockedBlockID, false, true
	}

	if vp.lockedNewView {
		return nil, true, true
	}
	return nil, false, false
}

//getVotes returns all of cached votes
func (vp *votePool) getVotes() []*chainedbft.VoteData {
	votes := make([]*chainedbft.VoteData, 0, len(vp.votes))
	indexes := make([]uint64, 0, len(vp.votes))
	for index := range vp.votes {
		indexes = append(indexes, index)
	}
	sort.Sort(orderIndexes(indexes))
	for _, index := range indexes {
		votes = append(votes, vp.votes[index])
	}

	return votes
}
