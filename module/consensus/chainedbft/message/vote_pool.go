/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package message

import (
	"bytes"
	"fmt"
	"sort"

	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
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
	lockedBlockId []byte //The +2/3 voted for vp block

	votes        map[uint64]*chainedbft.VoteData //format: [author index] = voteData; store all vote from author
	votedNewView map[uint64]*chainedbft.VoteData //format: [author index] = voteData;
	// only store newView vote from author
	votedBlockId map[string]map[uint64]*chainedbft.VoteData //format: [block hash][author index] = voteData;
	// only store proposal vote from author
}

//newVotePool initializes a votePool with given params
func newVotePool(size int) *votePool {
	return &votePool{
		newViewNum:    0,
		lockedBlockId: nil,
		lockedNewView: false,
		votes:         make(map[uint64]*chainedbft.VoteData, size),
		votedNewView:  make(map[uint64]*chainedbft.VoteData, size),
		votedBlockId:  make(map[string]map[uint64]*chainedbft.VoteData, size),
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

func (vp *votePool) addVoteIfNeed(vote *chainedbft.VoteData) {
	lastVote := vp.votes[vote.AuthorIdx]
	if lastVote == nil {
		vp.votes[vote.AuthorIdx] = vote
		return
	}
	if vote.NewView && !lastVote.NewView {
		lastVote.NewView = vote.NewView
	}
	if len(vote.BlockId) > 0 && len(lastVote.BlockId) == 0 {
		lastVote.BlockId = vote.BlockId
	}
}

func (vp *votePool) insertVoteData(vote *chainedbft.VoteData, minVotesForQc int) (bool, error) {
	if vote == nil {
		return false, fmt.Errorf("nil vote data")
	}
	if isValid, err := vp.checkDuplicationVote(vote); err != nil || !isValid {
		return false, err
	}

	vp.addVoteIfNeed(vote)
	// process NewView vote
	if vote.NewView {
		vp.votedNewView[vote.AuthorIdx] = vote
		if !vp.lockedNewView && len(vp.votedNewView) >= minVotesForQc {
			vp.lockedNewView = true
		}
	}

	// process block vote
	if len(vote.BlockId) == 0 {
		return true, nil
	}
	blockId := string(vote.BlockId)
	if _, ok := vp.votedBlockId[blockId]; !ok {
		vp.votedBlockId[blockId] = make(map[uint64]*chainedbft.VoteData, 1)
	}
	vp.votedBlockId[blockId][vote.AuthorIdx] = vote
	if vp.lockedBlockId == nil && len(vp.votedBlockId[blockId]) >= minVotesForQc {
		//Over 2/3 votes for same block and executed state root
		vp.lockedBlockId = vote.BlockId
	}
	return true, nil
}

func (vp *votePool) checkDuplicationVote(vote *chainedbft.VoteData) (isValid bool, err error) {
	lastVote, ok := vp.votes[vote.AuthorIdx]
	if !ok {
		return true, nil
	}
	if lastVote.BlockId != nil && vote.BlockId != nil && !bytes.Equal(lastVote.BlockId, vote.BlockId) {
		return false, fmt.Errorf("different votes block from same level %d, oldBlockId: %x, newBlockId: %x",
			vote.Level, lastVote.BlockId, vote.BlockId)
	} else if lastVote.NewView == vote.NewView && bytes.Equal(lastVote.BlockId, vote.BlockId) {
		return false, nil
	}
	return true, nil
}

//checkVoteDone checks whether a valid block or nil block voted by +2/3 nodes
func (vp *votePool) checkVoteDone() (blkID []byte, isNewView bool, done bool) {
	if vp.lockedBlockId != nil {
		return vp.lockedBlockId, false, true
	}

	if vp.lockedNewView {
		return nil, true, true
	}
	return nil, false, false
}

//getVotes returns all of cached votes
func (vp *votePool) getQCVotes() []*chainedbft.VoteData {
	indexes := make([]uint64, 0, len(vp.votes))
	votes := make([]*chainedbft.VoteData, 0, len(vp.votes))
	if len(vp.lockedBlockId) > 0 {
		blkVotes := vp.votedBlockId[string(vp.lockedBlockId)]
		for index := range blkVotes {
			indexes = append(indexes, index)
		}
		sort.Sort(orderIndexes(indexes))
		for _, index := range indexes {
			votes = append(votes, blkVotes[index])
		}
		return votes
	}
	if vp.lockedNewView {
		for index := range vp.votedNewView {
			indexes = append(indexes, index)
		}
		sort.Sort(orderIndexes(indexes))
		for _, index := range indexes {
			votes = append(votes, vp.votedNewView[index])
		}
		return votes
	}
	return nil
}
