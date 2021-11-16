/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/logger/v2"

	tbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/tbft"
)

//const (
//	nilVoteStr = "nil vote"
//)

var (
	ErrVoteNil              = errors.New("nil vote")
	ErrUnexceptedStep       = errors.New("unexpected step")
	ErrInvalidValidator     = errors.New("invalid validator")
	ErrVoteForDifferentHash = errors.New("vote for different hash")
)

// Proposal represent a proposal to be vote for consensus
type Proposal struct {
	Voter       string
	Height      uint64
	Round       int32
	PolRound    int32
	Block       *common.Block
	Endorsement *common.EndorsementEntry
}

// NewProposal create a new Proposal instance
func NewProposal(voter string, height uint64, round int32, polRound int32, block *common.Block) *Proposal {
	return &Proposal{
		Voter:    voter,
		Height:   height,
		Round:    round,
		PolRound: polRound,
		Block:    block,
	}
}

// NewProposalFromProto create a new Proposal instance from pb
func NewProposalFromProto(p *tbftpb.Proposal) *Proposal {
	if p == nil {
		return nil
	}
	proposal := NewProposal(
		p.Voter,
		p.Height,
		p.Round,
		p.PolRound,
		p.Block,
	)
	proposal.Endorsement = p.Endorsement

	return proposal
}

// ToProto serializes the Proposal instance
func (p *Proposal) ToProto() *tbftpb.Proposal {
	if p == nil {
		return nil
	}
	return &tbftpb.Proposal{
		Voter:       p.Voter,
		Height:      p.Height,
		Round:       p.Round,
		PolRound:    p.PolRound,
		Block:       p.Block,
		Endorsement: p.Endorsement,
	}
}

// Vote represents a vote to proposal
type Vote struct {
	Type        tbftpb.VoteType
	Voter       string
	Height      uint64
	Round       int32
	Hash        []byte
	Endorsement *common.EndorsementEntry
}

// NewVote create a new Vote instance
func NewVote(typ tbftpb.VoteType, voter string, height uint64, round int32, hash []byte) *Vote {
	return &Vote{
		Type:   typ,
		Voter:  voter,
		Height: height,
		Round:  round,
		Hash:   hash,
	}
}

// NewVoteFromProto create a new Vote instance from pb
func NewVoteFromProto(v *tbftpb.Vote) *Vote {
	vote := NewVote(
		v.Type,
		v.Voter,
		v.Height,
		v.Round,
		v.Hash,
	)
	vote.Endorsement = v.Endorsement

	return vote
}

// ToProto convert vote to protobuf message
func (v *Vote) ToProto() *tbftpb.Vote {
	if v == nil {
		return nil
	}

	return &tbftpb.Vote{
		Type:        v.Type,
		Voter:       v.Voter,
		Height:      v.Height,
		Round:       v.Round,
		Hash:        v.Hash,
		Endorsement: v.Endorsement,
	}
}

func (v *Vote) String() string {
	return fmt.Sprintf("Vote{%s-%s(%d/%d)-%x}", v.Type, v.Voter, v.Height, v.Round, v.Hash)
}

// BlockVotes traces the vote from different voter
type BlockVotes struct {
	Votes map[string]*Vote
	Sum   uint64
}

// NewBlockVotes creates a new BlockVotes instance
func NewBlockVotes() *BlockVotes {
	return &BlockVotes{
		Votes: make(map[string]*Vote),
	}
}

// NewBlockVotesFromProto creates a new BlockVotes instance from pb
func NewBlockVotesFromProto(bvProto *tbftpb.BlockVotes) *BlockVotes {
	bv := NewBlockVotes()
	for k, v := range bvProto.Votes {
		vote := NewVoteFromProto(v)
		bv.Votes[k] = vote
	}
	bv.Sum = bvProto.Sum
	return bv
}

// ToProto serializes the BlockVotes instance
func (bv *BlockVotes) ToProto() *tbftpb.BlockVotes {
	if bv == nil {
		return nil
	}

	bvProto := &tbftpb.BlockVotes{
		Votes: make(map[string]*tbftpb.Vote),
		Sum:   bv.Sum,
	}

	for k, v := range bv.Votes {
		bvProto.Votes[k] = v.ToProto()
	}

	return bvProto
}

func (bv *BlockVotes) addVote(vote *Vote) {
	bv.Votes[vote.Voter] = vote
	bv.Sum++
}

// VoteSet wraps tbftpb.VoteSet and validatorSet
type VoteSet struct {
	logger       *logger.CMLogger
	Type         tbftpb.VoteType
	Height       uint64
	Round        int32
	Sum          uint64
	Maj23        []byte
	Votes        map[string]*Vote
	VotesByBlock map[string]*BlockVotes
	validators   *validatorSet
}

// NewVoteSet creates a new VoteSet instance
func NewVoteSet(logger *logger.CMLogger, voteType tbftpb.VoteType, height uint64, round int32,
	validators *validatorSet) *VoteSet {
	return &VoteSet{
		logger:       logger,
		Type:         voteType,
		Height:       height,
		Round:        round,
		Votes:        make(map[string]*Vote),
		VotesByBlock: make(map[string]*BlockVotes),
		validators:   validators,
	}
}

// NewVoteSetFromProto creates a new VoteSet instance from pb
func NewVoteSetFromProto(logger *logger.CMLogger, vsProto *tbftpb.VoteSet, validators *validatorSet) *VoteSet {
	vs := NewVoteSet(logger, vsProto.Type, vsProto.Height, vsProto.Round, validators)

	for _, v := range vsProto.Votes {
		vote := NewVoteFromProto(v)
		added, err := vs.AddVote(vote)
		if !added || err != nil {
			logger.Errorf("validators: %s, vote: %s", validators, vote)
		}
	}

	return vs
}

// ToProto serializes the VoteSet instance
func (vs *VoteSet) ToProto() *tbftpb.VoteSet {
	if vs == nil {
		return nil
	}

	vsProto := &tbftpb.VoteSet{
		Type:         vs.Type,
		Height:       vs.Height,
		Round:        vs.Round,
		Sum:          vs.Sum,
		Maj23:        vs.Maj23,
		Votes:        make(map[string]*tbftpb.Vote),
		VotesByBlock: make(map[string]*tbftpb.BlockVotes),
	}

	for k, v := range vs.Votes {
		vsProto.Votes[k] = v.ToProto()
	}

	for k, v := range vs.VotesByBlock {
		vsProto.VotesByBlock[k] = v.ToProto()
	}

	return vsProto
}

func (vs *VoteSet) String() string {
	if vs == nil {
		return ""
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "{Type: %s, Height: %d, Round: %d, Votes: [",
		vs.Type, vs.Height, vs.Round)
	for k := range vs.Votes {
		fmt.Fprintf(&builder, " %s,", k)
	}
	fmt.Fprintf(&builder, "]}")
	return builder.String()
}

// Size returns the size of the VoteSet
func (vs *VoteSet) Size() int32 {
	if vs == nil {
		return 0
	}
	return vs.validators.Size()
}

func (vs *VoteSet) checkVoteMatched(vote *Vote) bool {
	if vote.Type != vs.Type ||
		vote.Height != vs.Height ||
		vote.Round != vs.Round {
		return false
	}
	return true
}

// AddVote adds a vote to the VoteSet
func (vs *VoteSet) AddVote(vote *Vote) (added bool, err error) {
	if vs == nil {
		// This should not happen
		panic("AddVote on nil VoteSet")
	}

	if vote == nil {
		vs.logger.Errorf("%v add nil vote error", vs)
		return false, fmt.Errorf("%w %v", ErrVoteNil, vote)
	}

	if !vs.checkVoteMatched(vote) {
		vs.logger.Infof("expect %s/%d/%d, got %s/%d/%d",
			vs.Type, vs.Height, vs.Round, vote.Type, vote.Height, vote.Round)
		return false, fmt.Errorf("%w expect %s/%d/%d, got %s/%d/%d",
			ErrUnexceptedStep, vs.Type, vs.Height, vs.Round,
			vote.Type, vote.Height, vote.Round)
	}

	if !vs.validators.HasValidator(vote.Voter) {
		return false, fmt.Errorf("%w %s", ErrInvalidValidator, vote.Voter)
	}

	if v, ok := vs.Votes[vote.Voter]; ok {
		if bytes.Equal(vote.Hash, v.Hash) {
			return false, nil
		}
		return false, fmt.Errorf("%w existing: %v, new: %v",
			ErrVoteForDifferentHash, v.Hash, vote.Hash)
	}

	vs.Votes[vote.Voter] = vote
	vs.Sum++

	hashStr := base64.StdEncoding.EncodeToString(vote.Hash)
	votesByBlock, ok := vs.VotesByBlock[hashStr]
	if !ok {
		votesByBlock = NewBlockVotes()
		vs.VotesByBlock[hashStr] = votesByBlock
	}

	oldSum := votesByBlock.Sum
	quorum := uint64(vs.validators.Size()*2/3 + 1)

	votesByBlock.addVote(vote)
	vs.logger.Debugf("VoteSet(%s/%d/%d) AddVote %s(%s/%d/%d/%x) "+
		"oldSum: %d, quorum: %d, sum: %d",
		vs.Type, vs.Height, vs.Round, vote.Voter, vote.Type, vote.Height, vote.Round,
		vote.Hash, oldSum, quorum, votesByBlock.Sum)

	if oldSum < quorum && quorum <= votesByBlock.Sum && vs.Maj23 == nil {
		vs.logger.Infof("VoteSet(%s/%d/%d) AddVote reach majority %x",
			vs.Type, vs.Height, vs.Round, vote.Hash)
		vs.Maj23 = vote.Hash

		for k, v := range votesByBlock.Votes {
			vs.Votes[k] = v
		}
	}

	return true, nil
}

func (vs *VoteSet) twoThirdsMajority() (hash []byte, ok bool) {
	if vs == nil {
		return nil, false
	}

	if vs.Maj23 != nil {
		vs.logger.Debugf("VoteSet(%s/%d/%d) TwoThirdsMajority (%x/%v)",
			vs.Type, vs.Height, vs.Round, vs.Maj23, true)
		return vs.Maj23, true
	}
	vs.logger.Debugf("VoteSet(%s/%d/%d) TwoThirdsMajority (%x/%v)",
		vs.Type, vs.Height, vs.Round, vs.Maj23, false)
	return nil, false
}

// HasTwoThirdsMajority shoule used when the mutex has been lock
func (vs *VoteSet) HasTwoThirdsMajority() (majority bool) {
	if vs == nil {
		return false
	}

	return vs.Maj23 != nil
}

func (vs *VoteSet) hasTwoThirdsAny() bool {
	if vs == nil {
		return false
	}

	ret := true
	leftSum := uint64(vs.validators.Size()) - vs.Sum
	for _, v := range vs.VotesByBlock {
		if (v.Sum + leftSum) >= uint64(vs.validators.Size()*2/3+1) {
			ret = false
			break
		}
	}
	vs.logger.Debugf("VoteSet(%s/%d/%d) sum: %v, HasTwoThirdsAny return %v",
		vs.Type, vs.Height, vs.Round, vs.Sum, ret)
	return ret
}

// 2f+1 any votes received
func (vs *VoteSet) hasTwoThirdsNoMajority() bool {
	if vs == nil {
		return false
	}

	return vs.Sum >= uint64(vs.validators.Size()*2/3+1)
}

type roundVoteSet struct {
	Height     uint64
	Round      int32
	Prevotes   *VoteSet
	Precommits *VoteSet
}

func newRoundVoteSet(height uint64, round int32, prevotes *VoteSet, precommits *VoteSet) *roundVoteSet {
	return &roundVoteSet{
		Height:     height,
		Round:      round,
		Prevotes:   prevotes,
		Precommits: precommits,
	}
}

func newRoundVoteSetFromProto(logger *logger.CMLogger, rvs *tbftpb.RoundVoteSet,
	validators *validatorSet) *roundVoteSet {
	if rvs == nil {
		return nil
	}
	prevotes := NewVoteSetFromProto(logger, rvs.Prevotes, validators)
	precommits := NewVoteSetFromProto(logger, rvs.Precommits, validators)
	return newRoundVoteSet(rvs.Height, rvs.Round, prevotes, precommits)
}

func (rvs *roundVoteSet) ToProto() *tbftpb.RoundVoteSet {
	if rvs == nil {
		return nil
	}

	rvsProto := &tbftpb.RoundVoteSet{
		Height:     rvs.Height,
		Round:      rvs.Round,
		Prevotes:   rvs.Prevotes.ToProto(),
		Precommits: rvs.Precommits.ToProto(),
	}
	return rvsProto
}

func (rvs *roundVoteSet) String() string {
	if rvs == nil {
		return ""
	}
	return fmt.Sprintf("Height: %d, Round: %d, Prevotes: %s, Precommits: %s",
		rvs.Height, rvs.Round, rvs.Prevotes, rvs.Precommits)
}

type heightRoundVoteSet struct {
	logger        *logger.CMLogger
	Height        uint64
	Round         int32
	RoundVoteSets map[int32]*roundVoteSet

	validators *validatorSet
}

func newHeightRoundVoteSet(logger *logger.CMLogger, height uint64, round int32,
	validators *validatorSet) *heightRoundVoteSet {
	hvs := &heightRoundVoteSet{
		logger:        logger,
		Height:        height,
		Round:         round,
		RoundVoteSets: make(map[int32]*roundVoteSet),

		validators: validators,
	}
	return hvs
}

//func newHeightRoundVoteSetFromProto(logger *logger.CMLogger, hvsProto *tbftpb.HeightRoundVoteSet,
//	validators *validatorSet) *heightRoundVoteSet {
//	hvs := newHeightRoundVoteSet(logger, hvsProto.Height, hvsProto.Round, validators)
//
//	for k, v := range hvsProto.RoundVoteSets {
//		rvs := NewRoundVoteSetFromProto(logger, v, validators)
//		hvs.RoundVoteSets[k] = rvs
//	}
//
//	return hvs
//}

func (hvs *heightRoundVoteSet) ToProto() *tbftpb.HeightRoundVoteSet {
	if hvs == nil {
		return nil
	}

	hvsProto := &tbftpb.HeightRoundVoteSet{
		Height:        hvs.Height,
		Round:         hvs.Round,
		RoundVoteSets: make(map[int32]*tbftpb.RoundVoteSet),
	}

	for k, v := range hvs.RoundVoteSets {
		hvsProto.RoundVoteSets[k] = v.ToProto()
	}

	return hvsProto
}

func (hvs *heightRoundVoteSet) addRound(round int32) {
	if _, ok := hvs.RoundVoteSets[round]; ok {
		// This should not happen
		panic(fmt.Errorf("round %d alread exists", round))
	}

	prevotes := NewVoteSet(hvs.logger, tbftpb.VoteType_VOTE_PREVOTE, hvs.Height, round, hvs.validators)
	precommits := NewVoteSet(hvs.logger, tbftpb.VoteType_VOTE_PRECOMMIT, hvs.Height, round, hvs.validators)
	hvs.RoundVoteSets[round] = newRoundVoteSet(hvs.Height, round, prevotes, precommits)
}

func (hvs *heightRoundVoteSet) getRoundVoteSet(round int32) *roundVoteSet {
	rvs, ok := hvs.RoundVoteSets[round]
	if !ok {
		return nil
	}
	return rvs
}

func (hvs *heightRoundVoteSet) getVoteSet(round int32, voteType tbftpb.VoteType) *VoteSet {
	rvs, ok := hvs.RoundVoteSets[round]
	if !ok {
		return nil
	}

	switch voteType {
	case tbftpb.VoteType_VOTE_PREVOTE:
		return rvs.Prevotes
	case tbftpb.VoteType_VOTE_PRECOMMIT:
		return rvs.Precommits
	default:
		// This should not happen
		panic(fmt.Errorf("invalid VoteType %s", voteType))
	}
}

func (hvs *heightRoundVoteSet) prevotes(round int32) *VoteSet {
	return hvs.getVoteSet(round, tbftpb.VoteType_VOTE_PREVOTE)
}

func (hvs *heightRoundVoteSet) precommits(round int32) *VoteSet {
	return hvs.getVoteSet(round, tbftpb.VoteType_VOTE_PRECOMMIT)
}

func (hvs *heightRoundVoteSet) addVote(vote *Vote) (added bool, err error) {
	voteSet := hvs.getVoteSet(vote.Round, vote.Type)
	if voteSet == nil {
		hvs.addRound(vote.Round)
		voteSet = hvs.getVoteSet(vote.Round, vote.Type)
	}

	added, err = voteSet.AddVote(vote)
	return
}

func createProposalMsg(proposal *Proposal) *tbftpb.TBFTMsg {
	proposalProto := proposal.ToProto()
	data := mustMarshal(proposalProto)

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_MSG_PROPOSE,
		Msg:  data,
	}

	return tbftMsg
}

func createPrevoteMsg(prevote *Vote) *tbftpb.TBFTMsg {
	prevoteProto := prevote.ToProto()
	data := mustMarshal(prevoteProto)

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_MSG_PREVOTE,
		Msg:  data,
	}

	return tbftMsg
}

func createPrecommitMsg(precommit *Vote) *tbftpb.TBFTMsg {
	precommitProto := precommit.ToProto()
	data := mustMarshal(precommitProto)

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_MSG_PRECOMMIT,
		Msg:  data,
	}

	return tbftMsg
}
