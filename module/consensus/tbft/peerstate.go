/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"fmt"
	"strings"
	"sync"

	netpb "chainmaker.org/chainmaker/pb-go/v2/net"

	"chainmaker.org/chainmaker/logger/v2"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	tbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/tbft"
	"github.com/gogo/protobuf/proto"
)

// PeerStateService represents the consensus state of peer node
type PeerStateService struct {
	sync.Mutex
	logger *logger.CMLogger
	Id     string
	Height uint64
	Round  int32
	Step   tbftpb.Step

	Proposal         []byte // proposal
	VerifingProposal []byte
	LockedRound      int32
	LockedProposal   *Proposal // locked proposal
	ValidRound       int32
	ValidProposal    *Proposal // valid proposal
	RoundVoteSet     *roundVoteSet

	stateC   chan *tbftpb.GossipState
	tbftImpl *ConsensusTBFTImpl
	msgbus   msgbus.MessageBus
	closeC   chan struct{}
}

// NewPeerStateService create a PeerStateService instance
func NewPeerStateService(logger *logger.CMLogger, id string, tbftImpl *ConsensusTBFTImpl) *PeerStateService {
	pcs := &PeerStateService{
		logger:   logger,
		Id:       id,
		tbftImpl: tbftImpl,
		msgbus:   tbftImpl.msgbus,
	}
	pcs.stateC = make(chan *tbftpb.GossipState, defaultChanCap)
	pcs.closeC = make(chan struct{})
	return pcs
}

func (pcs *PeerStateService) updateWithProto(pcsProto *tbftpb.GossipState) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "[%s] update with proto to (%d/%d/%s)",
		pcs.Id, pcsProto.Height, pcsProto.Round, pcsProto.Step)

	if pcsProto.RoundVoteSet != nil &&
		pcsProto.RoundVoteSet.Prevotes != nil &&
		pcsProto.RoundVoteSet.Prevotes.Votes != nil {
		fmt.Fprintf(&builder, " prevote: [")
		for k := range pcsProto.RoundVoteSet.Prevotes.Votes {
			fmt.Fprintf(&builder, "%s, ", k)
		}
		fmt.Fprintf(&builder, "]")
	}

	if pcsProto.RoundVoteSet != nil &&
		pcsProto.RoundVoteSet.Precommits != nil &&
		pcsProto.RoundVoteSet.Precommits.Votes != nil {
		fmt.Fprintf(&builder, " precommit: [")
		for k := range pcsProto.RoundVoteSet.Precommits.Votes {
			fmt.Fprintf(&builder, "%s, ", k)
		}
		fmt.Fprintf(&builder, "]")
	}

	pcs.logger.Debugf(builder.String())

	pcs.Lock()
	defer pcs.Unlock()

	pcs.Height = pcsProto.Height
	pcs.Round = pcsProto.Round
	pcs.Step = pcsProto.Step
	pcs.Proposal = pcsProto.Proposal
	pcs.VerifingProposal = pcsProto.VerifingProposal
	validatorSet := pcs.tbftImpl.getValidatorSet()
	pcs.RoundVoteSet = newRoundVoteSetFromProto(pcs.logger, pcsProto.RoundVoteSet, validatorSet)
	// fetch votes from this node state
	if pcs.Height == pcs.tbftImpl.Height && pcs.Round == pcs.tbftImpl.Round &&
		pcs.RoundVoteSet != nil {
		pcs.logger.Debugf("[%s] updateVoteWithProto: [%d/%d]", pcs.Id, pcs.Height, pcs.Round)
		pcs.updateVoteWithProto(pcs.RoundVoteSet)
	}
	pcs.logger.Debugf("[%s] RoundVoteSet: %s", pcs.Id, pcs.RoundVoteSet)
}

// get the votes for tbft Engine based on the peer node state
func (pcs *PeerStateService) updateVoteWithProto(voteSet *roundVoteSet) {
	for _, voter := range pcs.tbftImpl.getValidatorSet().Validators {
		pcs.logger.Debugf("%s updateVoteWithProto : %v,%v", voter, voteSet.Prevotes, voteSet.Precommits)
		// prevote Vote
		vote := voteSet.Prevotes.Votes[voter]
		if vote != nil && pcs.tbftImpl.Step < tbftpb.Step_PRECOMMIT {
			pcs.logger.Debugf("updateVoteWithProto prevote : %s", voter)
			tbftMsg := createPrevoteMsg(vote)
			pcs.tbftImpl.internalMsgC <- tbftMsg
		}
		// precommit Vote
		vote = voteSet.Precommits.Votes[voter]
		if vote != nil && pcs.tbftImpl.Step < tbftpb.Step_COMMIT {
			pcs.logger.Debugf("updateVoteWithProto precommit : %s", voter)
			tbftMsg := createPrevoteMsg(vote)
			pcs.tbftImpl.internalMsgC <- tbftMsg
		}
	}
}
func (pcs *PeerStateService) start() {
	go pcs.procStateChange()
}

func (pcs *PeerStateService) stop() {
	pcs.logger.Infof("[%s] stop PeerStateService", pcs.Id)
	close(pcs.closeC)
}

// GetStateC return the stateC channel
func (pcs *PeerStateService) GetStateC() chan<- *tbftpb.GossipState {
	return pcs.stateC
}

func (pcs *PeerStateService) procStateChange() {
	pcs.logger.Infof("PeerStateService[%s] start procStateChange", pcs.Id)
	defer pcs.logger.Infof("PeerStateService[%s] exit procStateChange", pcs.Id)

	loop := true
	for loop {
		select {
		case stateProto := <-pcs.stateC:
			pcs.updateWithProto(stateProto)

			pcs.sendStateChange()
		case <-pcs.closeC:
			loop = false
		}
	}
}

func (pcs *PeerStateService) gossipState(state *tbftpb.GossipState) {
	pcs.Lock()
	defer pcs.Unlock()

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_MSG_STATE,
		Msg:  mustMarshal(state),
	}

	pcs.logger.Debugf("Proposal: %d, verifingProposal: %d, HeightRoundVoteSet: %d",
		len(state.Proposal),
		len(state.VerifingProposal),
		proto.Size(state.RoundVoteSet),
	)
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(tbftMsg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.logger.Debugf("%s gossip (%d/%d/%s) to %s", state.Id, state.Height, state.Round, state.Step, pcs.Id)
	pcs.publishToMsgbus(netMsg)

	go pcs.sendStateChange()
}

func (pcs *PeerStateService) sendStateChange() {
	pcs.Lock()
	defer pcs.Unlock()

	pcs.tbftImpl.RLock()
	defer pcs.tbftImpl.RUnlock()

	pcs.logger.Debugf("[%s](%d/%d/%s) sendStateChange to [%s](%d/%d/%s)",
		pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
		pcs.Id, pcs.Height, pcs.Round, pcs.Step,
	)
	if pcs.tbftImpl.Height < pcs.Height {
		return
	} else if pcs.tbftImpl.Height == pcs.Height {
		pcs.logger.Debugf("[%s](%d) sendStateOfRound to [%s](%d/%d/%s)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.Id, pcs.Height, pcs.Round, pcs.Step)
		pcs.sendStateOfRound()
	} else {
		pcs.logger.Debugf("[%s](%d) sendStateOfHeight to [%s](%d/%d/%s)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.Id, pcs.Height, pcs.Round, pcs.Step)
		go pcs.sendStateOfHeight(pcs.Height)
	}
}

func (pcs *PeerStateService) sendStateOfRound() {
	pcs.sendProposalOfRound(pcs.Height, pcs.Round)
	pcs.sendPrevoteOfRound(pcs.Round)
	pcs.sendPrecommitOfRound(pcs.Round)
}

func (pcs *PeerStateService) sendProposalOfRound(height uint64, round int32) {
	// Send proposal (only proposer can send proposal)
	if pcs.tbftImpl.isProposer(height, round) &&
		pcs.tbftImpl.Proposal != nil &&
		pcs.VerifingProposal == nil &&
		pcs.Step >= tbftpb.Step_PROPOSE {
		pcs.logger.Debugf("[%s] sendProposalOfRound: [%d,%d]",
			pcs.Id, pcs.tbftImpl.Proposal.Height, pcs.tbftImpl.Proposal.Round)
		pcs.sendProposal(pcs.tbftImpl.Proposal)
	}
}

func (pcs *PeerStateService) sendPrevoteOfRound(round int32) {
	pcs.logger.Debugf("[%s] RoundVoteSet: %s", pcs.Id, pcs.RoundVoteSet)
	// Send prevote
	prevoteVs := pcs.tbftImpl.heightRoundVoteSet.prevotes(round)
	if prevoteVs != nil {
		vote, ok := prevoteVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Prevotes != nil {
			var builder strings.Builder
			fmt.Fprintf(&builder, " prevote: [")
			for k := range pcs.RoundVoteSet.Prevotes.Votes {
				fmt.Fprintf(&builder, "%s, ", k)
			}
			fmt.Fprintf(&builder, "]")
			pcs.logger.Debugf(builder.String())

			if _, pOk := pcs.RoundVoteSet.Prevotes.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrevote(vote)
			}
		}
	}
}

func (pcs *PeerStateService) sendPrecommitOfRound(round int32) {
	pcs.logger.Debugf("[%s] RoundVoteSet: %s", pcs.Id, pcs.RoundVoteSet)
	// Send precommit
	precommitVs := pcs.tbftImpl.heightRoundVoteSet.precommits(round)
	if precommitVs != nil {
		vote, ok := precommitVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Precommits != nil {
			var builder strings.Builder
			fmt.Fprintf(&builder, " precommit: [")
			for k := range pcs.RoundVoteSet.Precommits.Votes {
				fmt.Fprintf(&builder, "%s, ", k)
			}
			fmt.Fprintf(&builder, "]")
			pcs.logger.Debugf(builder.String())

			if _, pOk := pcs.RoundVoteSet.Precommits.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrecommit(vote)
			}
		}
	}
}

func (pcs *PeerStateService) publishToMsgbus(msg *netpb.NetMsg) {
	pcs.logger.Debugf("[%s] publishToMsgbus size: %d", pcs.tbftImpl.Id, proto.Size(msg))
	pcs.msgbus.Publish(msgbus.SendConsensusMsg, msg)
}

func (pcs *PeerStateService) sendProposal(proposal *Proposal) {
	pcs.logger.Infof("[%s](%d/%d/%s) sendProposal [%s](%d/%d/%x) to %v",
		pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
		proposal.Voter, proposal.Height, proposal.Round, proposal.Block.Header.BlockHash, pcs.Id)

	// Send proposal
	msg := createProposalMsg(proposal)
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(msg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.publishToMsgbus(netMsg)

	pcs.logger.Debugf("send proposal(%d/%x) to %s(%d/%d/%s)",
		proposal.Block.Header.BlockHeight, proposal.Block.Header.BlockHash,
		pcs.Id, pcs.Height, pcs.Round, pcs.Step)

}

func (pcs *PeerStateService) sendPrevote(prevote *Vote) {
	// Send prevote
	msg := createPrevoteMsg(prevote)
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(msg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.publishToMsgbus(netMsg)

	pcs.logger.Debugf("send prevote(%d/%d/%s/%x) to %s",
		pcs.Height, pcs.Round, pcs.Step, prevote.Hash, pcs.Id)
}

func (pcs *PeerStateService) sendPrecommit(precommit *Vote) {
	// Send precommit
	msg := createPrecommitMsg(precommit)

	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(msg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.publishToMsgbus(netMsg)
	pcs.logger.Debugf("send precommit(%d/%d/%s/%x) to %s",
		pcs.Height, pcs.Round, pcs.Step, precommit.Hash, pcs.Id)

}

func (pcs *PeerStateService) sendStateOfHeight(height uint64) {
	state := pcs.tbftImpl.consensusStateCache.getConsensusState(pcs.Height)
	if state == nil {
		pcs.logger.Debugf("[%s] no caching consensusState, height:%d", pcs.Id, pcs.Height)
		return
	}
	pcs.sendProposalInState(state)
	pcs.sendPrevoteInState(state)
	pcs.sendPrecommitInState(state)
}

func (pcs *PeerStateService) sendProposalInState(state *ConsensusState) {
	// Send Proposal (only proposer can send proposal)
	if pcs.tbftImpl.isProposer(state.Height, state.Round) &&
		state.Proposal != nil &&
		pcs.VerifingProposal == nil &&
		pcs.Step >= tbftpb.Step_PROPOSE {
		pcs.logger.Debugf("[%s] sendProposalInState: [%d,%d]",
			pcs.Id, state.Proposal.Height, state.Proposal.Round)
		pcs.sendProposal(state.Proposal)
	}
}

func (pcs *PeerStateService) sendPrevoteInState(state *ConsensusState) {
	// Send Prevote
	prevoteVs := state.heightRoundVoteSet.prevotes(pcs.Round)
	if prevoteVs != nil {
		vote, ok := prevoteVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Prevotes != nil {
			if _, pOk := pcs.RoundVoteSet.Prevotes.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrevote(vote)
			}
		}
	}
}

func (pcs *PeerStateService) sendPrecommitInState(state *ConsensusState) {
	// Send precommit
	precommitVs := state.heightRoundVoteSet.precommits(pcs.Round)
	if precommitVs != nil {
		vote, ok := precommitVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Precommits != nil {
			if _, pOk := pcs.RoundVoteSet.Precommits.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrecommit(vote)
			}
		}
	}
}
