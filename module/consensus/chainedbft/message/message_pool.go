/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package message

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

//MsgPool manages all of consensus messages received for protocol
type MsgPool struct {
	sync.RWMutex
	size int                        //The size of validators
	msgs map[uint64]*consensusRound //format: [height] = [consensusRound];
	// Stores consensus information at the same height and different levels
	cachedLen     uint64 //Cache the max length of latest heights
	minVotesForQc int    //Quorum certification
}

//NewMsgPool initializes a msg pool to manage all of consensus msgs for protocol
func NewMsgPool(cachedLen uint64, size, minVotesForQc int) *MsgPool {
	return &MsgPool{
		size:          size,
		cachedLen:     cachedLen,
		minVotesForQc: minVotesForQc,
		msgs:          make(map[uint64]*consensusRound),
	}
}

//InsertVote is an external api to cache a vote msg with given height and round
func (mp *MsgPool) InsertVote(height uint64, round uint64, voteMsg *chainedbft.ConsensusMsg) (bool, error) {
	if voteMsg == nil || voteMsg.Payload == nil {
		return false, fmt.Errorf("nil vote msg or nil payload")
	}
	if voteMsg.Payload.Type != chainedbft.MessageType_VOTE_MESSAGE {
		return false, fmt.Errorf("wrong vote type %v", voteMsg.Payload.Type)
	}

	mp.Lock()
	defer mp.Unlock()
	if _, ok := mp.msgs[height]; !ok {
		mp.msgs[height] = newConsensusRound(mp.size, height)
	}
	return mp.msgs[height].insertVote(round, voteMsg, mp.minVotesForQc)
}

//InsertProposal is an external api to cache a proposal msg with given height and round
func (mp *MsgPool) InsertProposal(height uint64, round uint64, msg *chainedbft.ConsensusMsg) (bool, error) {
	if msg == nil || msg.Payload == nil {
		return false, fmt.Errorf("try to insert nil proposal")
	}

	mp.Lock()
	defer mp.Unlock()
	if _, ok := mp.msgs[height]; !ok {
		mp.msgs[height] = newConsensusRound(mp.size, height)
	}
	return mp.msgs[height].insertProposal(round, msg)
}

//GetProposal is an external api to get a proposal with given height and round
func (mp *MsgPool) GetProposal(height uint64, round uint64) *chainedbft.ConsensusMsg {
	mp.RLock()
	defer mp.RUnlock()
	if _, ok := mp.msgs[height]; !ok {
		return nil
	}
	return mp.msgs[height].getProposal(round)
}

//GetVotes is an external api to get votes at given height and round
func (mp *MsgPool) GetQCVotes(height uint64, round uint64) []*chainedbft.VoteData {
	mp.RLock()
	defer mp.RUnlock()
	if _, ok := mp.msgs[height]; !ok {
		return nil
	}
	return mp.msgs[height].getQCVotes(round)
}

//CheckAnyVotes is an external api to check whether self have received minVotesForQc votes
func (mp *MsgPool) CheckAnyVotes(height uint64, round uint64) bool {
	mp.RLock()
	defer mp.RUnlock()

	if _, ok := mp.msgs[height]; !ok {
		return false
	}
	return mp.msgs[height].checkAnyVotes(round, chainedbft.MessageType_VOTE_MESSAGE, mp.minVotesForQc)
}

//CheckVotesDone is an external api to check whether self have received enough votes for a valid block or change view
func (mp *MsgPool) CheckVotesDone(height uint64, round uint64) ([]byte, bool, bool) {
	mp.RLock()
	defer mp.RUnlock()

	if _, ok := mp.msgs[height]; !ok {
		return nil, false, false
	}
	return mp.msgs[height].checkVoteDone(round, chainedbft.MessageType_VOTE_MESSAGE)
}

//GetLastValidRound is an external api to get latest valid round at height
func (mp *MsgPool) GetLastValidRound(height uint64) int64 {
	mp.RLock()
	defer mp.RUnlock()
	if _, ok := mp.msgs[height]; !ok {
		return -1
	}
	return mp.msgs[height].getLastValidRound()
}

//OnBlockSealed is an external api to cleanup the old messages
func (mp *MsgPool) OnBlockSealed(height uint64) {
	mp.Lock()
	defer mp.Unlock()

	if height <= mp.cachedLen {
		return
	}
	toFreeHeight := make([]uint64, 0)
	for h := range mp.msgs {
		if h < height-mp.cachedLen {
			toFreeHeight = append(toFreeHeight, h)
		}
	}
	for _, h := range toFreeHeight {
		delete(mp.msgs, h)
	}
}

//Cleanup cleans up the cached messages
func (mp *MsgPool) Cleanup() {
	mp.Lock()
	defer mp.Unlock()
	mp.msgs = make(map[uint64]*consensusRound)
}

//Reset resets msg pool
func (mp *MsgPool) Reset(size, minVotesForQc int) {
	mp.Lock()
	defer mp.Unlock()
	mp.minVotesForQc = minVotesForQc
	mp.size = size
	mp.msgs = make(map[uint64]*consensusRound)
}
