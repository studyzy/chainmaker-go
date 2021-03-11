/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"chainmaker.org/chainmaker-go/logger"
	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
)

// ConsensusState represents the consensus state of the node
type ConsensusState struct {
	logger *logger.CMLogger
	Id     string

	Height int64
	Round  int32
	Step   tbftpb.Step

	Proposal           *Proposal // proposal
	VerifingProposal   *Proposal // verifing proposal
	LockedRound        int32
	LockedProposal     *Proposal // locked proposal
	ValidRound         int32
	ValidProposal      *Proposal // valid proposal
	HeightRoundVoteSet *heightRoundVoteSet
}

// NewConsensusState creates a new ConsensusState instance
func NewConsensusState(logger *logger.CMLogger, id string) *ConsensusState {
	cs := &ConsensusState{
		logger: logger,
		Id:     id,
	}
	return cs
}

// NewConsensusStateFromProto creates a new ConsensusState instance from pb
func NewConsensusStateFromProto(logger *logger.CMLogger, csProto *tbftpb.ConsensusState, validators *validatorSet) *ConsensusState {
	cs := NewConsensusState(logger, csProto.Id)
	cs.Height = csProto.Height
	cs.Round = csProto.Round
	cs.Step = csProto.Step
	cs.Proposal = NewProposalFromProto(csProto.Proposal)
	cs.VerifingProposal = NewProposalFromProto(csProto.VerifingProposal)
	cs.HeightRoundVoteSet = newHeightRoundVoteSetFromProto(logger, csProto.HeightRoundVoteSet, validators)

	return cs
}

// ToProto serializes the ConsensusState instance
func (cs *ConsensusState) ToProto() *tbftpb.ConsensusState {
	if cs == nil {
		return nil
	}
	csProto := &tbftpb.ConsensusState{
		Id:                 cs.Id,
		Height:             cs.Height,
		Round:              cs.Round,
		Step:               cs.Step,
		Proposal:           cs.Proposal.ToProto(),
		VerifingProposal:   cs.VerifingProposal.ToProto(),
		HeightRoundVoteSet: cs.HeightRoundVoteSet.ToProto(),
	}
	return csProto
}
