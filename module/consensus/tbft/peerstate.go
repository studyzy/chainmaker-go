/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	netpb "chainmaker.org/chainmaker-go/pb/protogo/net"
	"sync"

	"chainmaker.org/chainmaker-go/logger"

	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
	"github.com/gogo/protobuf/proto"
)

// PeerStateService represents the consensus state of peer node
type PeerStateService struct {
	sync.Mutex
	logger *logger.CMLogger
	Id     string
	Height int64
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
	pcs.Lock()
	defer pcs.Unlock()

	pcs.logger.Debugf("[%s] update with proto to (%d/%d/%s)",
		pcs.Id, pcsProto.Height, pcsProto.Round, pcsProto.Step)
	pcs.Height = pcsProto.Height
	pcs.Round = pcsProto.Round
	pcs.Step = pcsProto.Step
	pcs.Proposal = pcsProto.Proposal
	pcs.VerifingProposal = pcsProto.VerifingProposal
	pcs.RoundVoteSet = NewRoundVoteSetFromProto(pcs.logger, pcsProto.RoundVoteSet, nil)
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
			break
		}
	}
}

func (pcs *PeerStateService) gossipState(state *tbftpb.GossipState) {
	pcs.Lock()
	defer pcs.Unlock()

	if state.Height < pcs.Height {
		pcs.logger.Debugf("[%s](%d/%d/%s) skip send state to %s(%d/%d/%s)",
			state.Id, state.Height, state.Round, state.Step,
			pcs.Id, pcs.Height, pcs.Round, pcs.Step,
		)
		return
	}

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_state,
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

	pcs.logger.Debugf("begin sendStateChange to %s", pcs.Id)
	defer pcs.logger.Debugf("end sendStateChange to %s", pcs.Id)

	pcs.tbftImpl.RLock()
	defer pcs.tbftImpl.RUnlock()

	if pcs.tbftImpl.Height != pcs.Height || pcs.tbftImpl.Round < pcs.Round {
		pcs.logger.Debugf("[%s](%d/%d/%s) receive invalid state (%d/%d/%s)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
			pcs.Height, pcs.Round, pcs.Step,
		)
		return
	}

	if pcs.tbftImpl.Round == pcs.Round {
		pcs.sendStateChangeInSameRound()
	} else if pcs.tbftImpl.Round > pcs.Round {
		pcs.sendStateChangeInDifferentRound()
	} else {
		panic("this should not happen")
	}
}

func (pcs *PeerStateService) sendProposalInSameRound() {
	// Send proposal
	if pcs.tbftImpl.isProposer(pcs.tbftImpl.Height, pcs.tbftImpl.Round) &&
		pcs.tbftImpl.Proposal != nil &&
		pcs.VerifingProposal == nil &&
		pcs.Step >= tbftpb.Step_Propose {
		msg, err := createProposalMsg(pcs.tbftImpl.Proposal)

		if localconf.ChainMakerConfig.DebugConfig.IsProposalOldHeight {
			pcs.logger.Infof("[%s](%d/%d/%v) switch IsPrevoteOldHeight: %v, prevote old height: %v",
				pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
				localconf.ChainMakerConfig.DebugConfig.IsProposalOldHeight, pcs.tbftImpl.Height-1)
			msgClone := proto.Clone(msg)
			proposalProto := new(tbftpb.Proposal)
			mustUnmarshal(msgClone.(*tbftpb.TBFTMsg).Msg, proposalProto)
			proposalProto.Height -= 1
			proposal := NewProposal(proposalProto.Voter, proposalProto.Height, proposalProto.Round, proposalProto.PolRound, proposalProto.Block)
			proposal.Endorsement = proposalProto.Endorsement
			msg, err = createProposalMsg(proposal)
		}

		if err != nil {
			pcs.logger.Errorf("[%s](%d/%d/%s) create proposal msg falied, %v",
				pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step, err)
			return
		}
		netMsg := &netpb.NetMsg{
			Payload: mustMarshal(msg),
			Type:    netpb.NetMsg_CONSENSUS_MSG,
			To:      pcs.Id,
		}

		//Simulate a node which lost the Proposal
		if localconf.ChainMakerConfig.DebugConfig.IsProposeLost {
			pcs.logger.Infof("[%s](%v/%v/%v) switch IsProposeLost: %v",
				pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
				localconf.ChainMakerConfig.DebugConfig.IsProposeLost)
			return
		}

		//Simulate a malicious node propose duplicate proposal during a round
		if localconf.ChainMakerConfig.DebugConfig.IsProposeDuplicately {
			pcs.logger.Infof("[%s](%v/%v/%v) switch IsProposeDuplicately: %v, propose duplicately to %s",
				pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
				localconf.ChainMakerConfig.DebugConfig.IsProposeDuplicately, pcs.Id)
			pcs.publishToMsgbus(netMsg)
		}

		//normal broadcast
		pcs.publishToMsgbus(netMsg)

		pcs.logger.Debugf("send proposal(%d/%x) to %s(%d/%d/%s)",
			pcs.tbftImpl.Proposal.Block.Header.BlockHeight, pcs.tbftImpl.Proposal.Block.Header.BlockHash,
			pcs.Id, pcs.Height, pcs.Round, pcs.Step)
	}
}

func (pcs *PeerStateService) sendPrevoteInSameRound() {
	// Send prevote
	prevoteVs := pcs.tbftImpl.heightRoundVoteSet.prevotes(pcs.tbftImpl.Round)
	if prevoteVs == nil {
		return
	}
	vote, ok := prevoteVs.Votes[pcs.tbftImpl.Id]
	if !ok {
		return
	}
	if pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Prevotes != nil {
		peerPrevoteVs := pcs.RoundVoteSet.Prevotes
		if _, pOk := peerPrevoteVs.Votes[pcs.tbftImpl.Id]; !pOk {
			msg := createPrevoteMsg(vote)
			if localconf.ChainMakerConfig.DebugConfig.IsPrevoteOldHeight {
				pcs.logger.Infof("[%s](%d/%d/%v) switch IsPrevoteOldHeight: %v, prevote old height: %v",
					pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
					localconf.ChainMakerConfig.DebugConfig.IsPrevoteOldHeight, pcs.tbftImpl.Height-1)
				msgClone := proto.Clone(msg)
				prevoteProto := new(tbftpb.Vote)
				mustUnmarshal(msgClone.(*tbftpb.TBFTMsg).Msg, prevoteProto)
				prevoteProto.Height -= 1
				prevote := NewVoteFromProto(prevoteProto)
				msg = createPrevoteMsg(prevote)
			}
			netMsg := &netpb.NetMsg{
				Payload: mustMarshal(msg),
				Type:    netpb.NetMsg_CONSENSUS_MSG,
				To:      pcs.Id,
			}
			//Simulate a node which lost the Proposal
			if localconf.ChainMakerConfig.DebugConfig.IsPrevoteLost {
				pcs.logger.Infof("[%s](%v/%v/%v) switch IsPrevoteLost: %v",
					pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
					localconf.ChainMakerConfig.DebugConfig.IsPrevoteLost)
				return
			}

			pcs.publishToMsgbus(netMsg)

			//Simulate a node which Prevote duplicately
			if localconf.ChainMakerConfig.DebugConfig.IsPrevoteDuplicately {
				pcs.logger.Infof("[%s](%v/%v/%v) switch IsPrevoteDuplicately: %v, prevote duplicately",
					pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
					localconf.ChainMakerConfig.DebugConfig.IsPrevoteDuplicately)
				pcs.publishToMsgbus(netMsg)
			}

			pcs.logger.Debugf("send prevote(%d/%d/%s/%x) to %s",
				pcs.Height, pcs.Round, pcs.Step, vote.Hash, pcs.Id)
		}
	}
}

func (pcs *PeerStateService) sendPrecommitInSameRound() {
	// Send precommit
	precommitVs := pcs.tbftImpl.heightRoundVoteSet.precommits(pcs.tbftImpl.Round)
	if precommitVs == nil {
		return
	}
	vote, ok := precommitVs.Votes[pcs.tbftImpl.Id]
	if !ok {
		return
	}
	if pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Precommits != nil {
		peerPrecommitVs := pcs.RoundVoteSet.Precommits
		if _, pOk := peerPrecommitVs.Votes[pcs.tbftImpl.Id]; !pOk {
			msg := createPrecommitMsg(vote)
			if localconf.ChainMakerConfig.DebugConfig.IsPrecommitOldHeight {
				pcs.logger.Infof("[%s](%v/%v/%v) switch IsPrecommitOldHeight: %v, precommit old height: %v",
					pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
					localconf.ChainMakerConfig.DebugConfig.IsPrecommitOldHeight, pcs.tbftImpl.Height-1)
				msgClone := proto.Clone(msg)
				precommitProto := new(tbftpb.Vote)
				mustUnmarshal(msgClone.(*tbftpb.TBFTMsg).Msg, precommitProto)
				precommitProto.Height -= 1
				precommit := NewVoteFromProto(precommitProto)
				msg = createPrevoteMsg(precommit)
			}

			netMsg := &netpb.NetMsg{
				Payload: mustMarshal(msg),
				Type:    netpb.NetMsg_CONSENSUS_MSG,
				To:      pcs.Id,
			}
			//Simulate a node which lost its Precommit
			if localconf.ChainMakerConfig.DebugConfig.IsPrecommitLost {
				pcs.logger.Infof("[%s](%v/%v/%v) switch IsPrecommitLost: %v",
					pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
					localconf.ChainMakerConfig.DebugConfig.IsPrecommitLost)
				return
			}
			pcs.publishToMsgbus(netMsg)
			pcs.logger.Debugf("send precommit(%d/%d/%s/%x) to %s",
				pcs.Height, pcs.Round, pcs.Step, vote.Hash, pcs.Id)

			//Simulate a node which Precommit duplicately
			if localconf.ChainMakerConfig.DebugConfig.IsPrecommitDuplicately {
				pcs.logger.Infof("[%s](%v/%v/%v) switch IsPrecommitDuplicately: %v, precommit duplicately",
					pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
					localconf.ChainMakerConfig.DebugConfig.IsPrecommitDuplicately)
				pcs.publishToMsgbus(netMsg)
			}
		}
	}
}

func (pcs *PeerStateService) sendStateChangeInSameRound() {
	pcs.sendProposalInSameRound()
	pcs.sendPrevoteInSameRound()
	pcs.sendPrecommitInSameRound()
}

func (pcs *PeerStateService) sendPrevoteInDifferentRound() {
	// Send prevote
	prevoteVs := pcs.tbftImpl.heightRoundVoteSet.prevotes(pcs.Round)
	if prevoteVs != nil {
		vote, ok := prevoteVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Prevotes != nil {
			peerPrevoteVs := pcs.RoundVoteSet.Prevotes
			if _, pOk := peerPrevoteVs.Votes[pcs.tbftImpl.Id]; !pOk {
				msg := createPrevoteMsg(vote)
				netMsg := &netpb.NetMsg{
					Payload: mustMarshal(msg),
					Type:    netpb.NetMsg_CONSENSUS_MSG,
					To:      pcs.Id,
				}
				pcs.publishToMsgbus(netMsg)
			}
		}
	}
}

func (pcs *PeerStateService) sendPrecommitInDifferentRound() {
	// Send precommit
	precommitVs := pcs.tbftImpl.heightRoundVoteSet.precommits(pcs.Round)
	if precommitVs != nil {
		vote, ok := precommitVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Precommits != nil {
			peerPrecommitVs := pcs.RoundVoteSet.Precommits
			if _, pOk := peerPrecommitVs.Votes[pcs.tbftImpl.Id]; !pOk {
				msg := createPrecommitMsg(vote)
				netMsg := &netpb.NetMsg{
					Payload: mustMarshal(msg),
					Type:    netpb.NetMsg_CONSENSUS_MSG,
					To:      pcs.Id,
				}
				pcs.publishToMsgbus(netMsg)
			}
		}
	}
}

func (pcs *PeerStateService) sendStateChangeInDifferentRound() {
	pcs.sendPrevoteInDifferentRound()
	pcs.sendPrecommitInDifferentRound()
}

func (pcs *PeerStateService) publishToMsgbus(msg *netpb.NetMsg) {
	pcs.logger.Debugf("[%s] publishToMsgbus size: %d", pcs.tbftImpl.Id, proto.Size(msg))
	pcs.msgbus.Publish(msgbus.SendConsensusMsg, msg)
}
