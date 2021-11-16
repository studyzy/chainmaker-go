/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"strconv"
	"sync"
	"time"

	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/common/v2/wal"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	tbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/tbft"
	netpb "chainmaker.org/chainmaker/pb-go/v2/net"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/zap"
)

var clog = zap.S()

var (
	defaultChanCap = 1000
	nilHash        = []byte("NilHash")
	//consensusStateKey              = []byte("ConsensusStateKey")
	walDir                         = "tbftwal"
	defaultConsensusStateCacheSize = uint64(10)
)

const (
	DefaultTimeoutPropose      = 30 * time.Second // Timeout of waitting for a proposal before prevoting nil
	DefaultTimeoutProposeDelta = 1 * time.Second  // Increased time delta of TimeoutPropose between rounds
	DefaultBlocksPerProposer   = uint64(1)        // The number of blocks each proposer can propose
	TimeoutPrevote             = 30 * time.Second // Timeout of waitting for >2/3 prevote
	TimeoutPrevoteDelta        = 1 * time.Second  // Increased time delta of TimeoutPrevote between round
	TimeoutPrecommit           = 30 * time.Second // Timeout of waitting for >2/3 precommit
	TimeoutPrecommitDelta      = 1 * time.Second  // Increased time delta of TimeoutPrecommit between round
	TimeoutCommit              = 30 * time.Second
)

// mustMarshal marshals protobuf message to byte slice or panic
func mustMarshal(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
}

// mustUnmarshal unmarshals from byte slice to protobuf message or panic
func mustUnmarshal(b []byte, msg proto.Message) {
	if err := proto.Unmarshal(b, msg); err != nil {
		panic(err)
	}
}

// ConsensusTBFTImpl is the implementation of TBFT algorithm
// and it implements the ConsensusEngine interface.
type ConsensusTBFTImpl struct {
	sync.RWMutex
	logger             *logger.CMLogger
	chainID            string
	Id                 string
	dpos               protocol.DPoS
	singer             protocol.SigningMember
	ac                 protocol.AccessControlProvider
	dbHandle           protocol.DBHandle
	ledgerCache        protocol.LedgerCache
	chainConf          protocol.ChainConf
	netService         protocol.NetService
	msgbus             msgbus.MessageBus
	closeC             chan struct{}
	internalMsgCCloseC chan struct{}
	waldir             string
	wal                *wal.Log
	heightFirstIndex   uint64

	validatorSet *validatorSet

	*ConsensusState
	consensusStateCache *consensusStateCache

	gossip        *gossipService
	timeScheduler *timeScheduler

	proposedBlockC chan *consensuspb.ProposalBlock
	verifyResultC  chan *consensuspb.VerifyResult
	blockHeightC   chan uint64
	externalMsgC   chan *tbftpb.TBFTMsg
	internalMsgC   chan *tbftpb.TBFTMsg

	TimeoutPropose      time.Duration
	TimeoutProposeDelta time.Duration

	// time metrics
	metrics *heightMetrics
}

// ConsensusTBFTImplConfig contains initialization config for ConsensusTBFTImpl
type ConsensusTBFTImplConfig struct {
	ChainID     string
	Id          string
	Dpos        protocol.DPoS
	Signer      protocol.SigningMember
	Ac          protocol.AccessControlProvider
	DbHandle    protocol.DBHandle
	LedgerCache protocol.LedgerCache
	ChainConf   protocol.ChainConf
	NetService  protocol.NetService
	MsgBus      msgbus.MessageBus
}

// New creates a tbft consensus instance
func New(config ConsensusTBFTImplConfig) (*ConsensusTBFTImpl, error) {
	var err error
	consensus := &ConsensusTBFTImpl{}
	consensus.logger = logger.GetLoggerByChain(logger.MODULE_CONSENSUS, config.ChainID)
	consensus.logger.Infof("New ConsensusTBFTImpl[%s]", config.Id)
	consensus.chainID = config.ChainID
	consensus.Id = config.Id
	consensus.singer = config.Signer
	consensus.ac = config.Ac
	consensus.dbHandle = config.DbHandle
	consensus.ledgerCache = config.LedgerCache
	consensus.chainConf = config.ChainConf
	consensus.netService = config.NetService
	consensus.msgbus = config.MsgBus
	consensus.closeC = make(chan struct{})
	consensus.internalMsgCCloseC = make(chan struct{})

	if config.ChainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_DPOS {
		consensus.dpos = config.Dpos
	}
	consensus.waldir = path.Join(localconf.ChainMakerConfig.GetStorePath(), consensus.chainID, walDir)
	consensus.wal, err = wal.Open(consensus.waldir, nil)
	if err != nil {
		return nil, err
	}
	consensus.heightFirstIndex = 0

	consensus.proposedBlockC = make(chan *consensuspb.ProposalBlock, defaultChanCap)
	consensus.verifyResultC = make(chan *consensuspb.VerifyResult, defaultChanCap)
	consensus.blockHeightC = make(chan uint64, defaultChanCap)
	consensus.externalMsgC = make(chan *tbftpb.TBFTMsg, defaultChanCap)
	consensus.internalMsgC = make(chan *tbftpb.TBFTMsg, defaultChanCap)

	validators, err := GetValidatorListFromConfig(consensus.chainConf.ChainConfig())
	if err != nil {
		return nil, err
	}
	consensus.validatorSet = newValidatorSet(consensus.logger, validators, DefaultBlocksPerProposer)
	consensus.ConsensusState = NewConsensusState(consensus.logger, consensus.Id)
	consensus.consensusStateCache = newConsensusStateCache(defaultConsensusStateCacheSize)
	consensus.timeScheduler = newTimeSheduler(consensus.logger, config.Id)
	consensus.gossip = newGossipService(consensus.logger, consensus)

	return consensus, nil
}

// Start starts the tbft instance with:
// 1. Register to message bus for subscribing topics
// 2. Start background goroutinues for processing events
// 3. Start timeScheduler for processing timeout shedule
func (consensus *ConsensusTBFTImpl) Start() error {
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.VerifyResult, consensus)
	consensus.msgbus.Register(msgbus.RecvConsensusMsg, consensus)
	consensus.msgbus.Register(msgbus.BlockInfo, consensus)
	_ = chainconf.RegisterVerifier(consensus.chainID, consensuspb.ConsensusType_TBFT, consensus)

	consensus.logger.Infof("start ConsensusTBFTImpl[%s]", consensus.Id)
	consensus.timeScheduler.Start()
	err := consensus.replayWal()
	if err != nil {
		return err
	}

	consensus.gossip.start()
	go consensus.handle()
	return nil
}

func (consensus *ConsensusTBFTImpl) sendProposeState(isProposer bool) {
	consensus.logger.Debugf("[%s](%d/%d/%s) sendProposeState isProposer: %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, isProposer)
	consensus.msgbus.PublishSafe(msgbus.ProposeState, isProposer)
}

// Stop implements the Stop method of ConsensusEngine interface.
func (consensus *ConsensusTBFTImpl) Stop() error {
	consensus.Lock()
	defer consensus.Unlock()

	consensus.logger.Infof("[%s](%d/%d/%s) stopped", consensus.Id, consensus.Height, consensus.Round,
		consensus.Step)
	err := consensus.wal.Sync()
	if err != nil {
		return err
	}
	consensus.gossip.stop()
	close(consensus.closeC)
	return nil
}

// 1. when leadership transfer, change consensus state and send singal
// atomic.StoreInt32()
// proposable <- atomic.LoadInt32(consensus.isLeader)

// 2. when receive pre-prepare block, send block to verifyBlockC
// verifyBlockC <- block

// 3. when receive commit block, send block to commitBlockC
// commitBlockC <- block
func (consensus *ConsensusTBFTImpl) OnMessage(message *msgbus.Message) {
	consensus.logger.Debugf("[%s] OnMessage receive topic: %s", consensus.Id, message.Topic)

	switch message.Topic {
	case msgbus.ProposedBlock:
		if proposedBlock, ok := message.Payload.(*consensuspb.ProposalBlock); ok {
			consensus.proposedBlockC <- proposedBlock
			consensus.logger.Debugf("len of proposedBlockC: %d", len(consensus.proposedBlockC))
		}
	case msgbus.VerifyResult:
		if verifyResult, ok := message.Payload.(*consensuspb.VerifyResult); ok {
			consensus.logger.Debugf("[%s] verify result: %s", consensus.Id, verifyResult.Code)
			consensus.verifyResultC <- verifyResult
		}
	case msgbus.RecvConsensusMsg:
		if msg, ok := message.Payload.(*netpb.NetMsg); ok {
			tbftMsg := new(tbftpb.TBFTMsg)
			mustUnmarshal(msg.Payload, tbftMsg)
			consensus.externalMsgC <- tbftMsg
		} else {
			panic(fmt.Errorf("receive message failed, error message type"))
		}
	case msgbus.BlockInfo:
		if blockInfo, ok := message.Payload.(*common.BlockInfo); ok {
			if blockInfo == nil || blockInfo.Block == nil {
				consensus.logger.Errorf("receive message failed, error message BlockInfo = nil")
				return
			}
			consensus.blockHeightC <- blockInfo.Block.Header.BlockHeight
		} else {
			panic(fmt.Errorf("error message type"))
		}
	}
}

func (consensus *ConsensusTBFTImpl) OnQuit() {
	// do nothing
	//panic("implement me")
}

// Verify implements interface of struct Verifier,
// This interface is used to verify the validity of parameters,
// it executes before consensus.
func (consensus *ConsensusTBFTImpl) Verify(consensusType consensuspb.ConsensusType,
	chainConfig *config.ChainConfig) error {
	consensus.logger.Infof("[%s](%d/%d/%v) verify chain config",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step)
	if consensusType != consensuspb.ConsensusType_TBFT {
		errMsg := fmt.Sprintf("consensus type is not TBFT: %s", consensusType)
		return errors.New(errMsg)
	}
	config := chainConfig.Consensus
	_, _, _, _, err := consensus.extractConsensusConfig(config)
	return err
}

func (consensus *ConsensusTBFTImpl) updateChainConfig() (addedValidators []string, removedValidators []string,
	err error) {
	consensus.logger.Debugf("[%s](%d/%d/%v) update chain config",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step)

	config := consensus.chainConf.ChainConfig().Consensus
	validators, timeoutPropose, timeoutProposeDelta, tbftBlocksPerProposer,
		err := consensus.extractConsensusConfig(config)
	if err != nil {
		return nil, nil, err
	}

	consensus.logger.Debugf("[%s](%d/%d/%v) update chain config, config: %v,"+
		" TimeoutPropose: %v, TimeoutProposeDelta: %v, validators: %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, config,
		consensus.TimeoutPropose, consensus.TimeoutProposeDelta, validators)

	consensus.TimeoutPropose = timeoutPropose
	consensus.TimeoutProposeDelta = timeoutProposeDelta
	if consensus.chainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_DPOS {
		consensus.logger.Debugf("enter dpos to get proposers ...")
		if validators, err = consensus.dpos.GetValidators(); err != nil {
			return nil, nil, err
		}
	} else {
		consensus.logger.Debugf("enter tbft to get proposers ...")
		err := consensus.validatorSet.updateBlocksPerProposer(tbftBlocksPerProposer)
		if err != nil {
			consensus.logger.Errorf("update Proposer per Blocks failed err: %s", err)
		}
	}
	return consensus.validatorSet.updateValidators(validators)
}

func (consensus *ConsensusTBFTImpl) extractConsensusConfig(config *config.ConsensusConfig) (validators []string,
	timeoutPropose time.Duration, timeoutProposeDelta time.Duration, tbftBlocksPerProposer uint64, err error) {
	timeoutPropose = DefaultTimeoutPropose
	timeoutProposeDelta = DefaultTimeoutProposeDelta
	tbftBlocksPerProposer = uint64(1)

	validators, err = GetValidatorListFromConfig(consensus.chainConf.ChainConfig())
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%v) get validator list from config failed: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, validators)
		return
	}

	for _, v := range config.ExtConfig {
		switch v.Key {
		case protocol.TBFT_propose_timeout_key:
			timeoutPropose, err = consensus.extractProposeTimeout(string(v.Value))
		case protocol.TBFT_propose_delta_timeout_key:
			timeoutProposeDelta, err = consensus.extractProposeTimeoutDelta(string(v.Value))
		case protocol.TBFT_blocks_per_proposer:
			tbftBlocksPerProposer, err = consensus.extractBlocksPerProposer(string(v.Value))
		}

		if err != nil {
			return
		}
	}

	return
}

func (consensus *ConsensusTBFTImpl) extractProposeTimeout(value string) (timeoutPropose time.Duration, err error) {
	if timeoutPropose, err = time.ParseDuration(value); err != nil {
		consensus.logger.Infof("[%s](%d/%d/%v) update chain config, TimeoutPropose: %v,"+
			" TimeoutProposeDelta: %v,"+" parse TimeoutPropose error: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			consensus.TimeoutPropose, consensus.TimeoutProposeDelta, err)
	}
	return
}

func (consensus *ConsensusTBFTImpl) extractProposeTimeoutDelta(value string) (timeoutProposeDelta time.Duration,
	err error) {
	if timeoutProposeDelta, err = time.ParseDuration(value); err != nil {
		consensus.logger.Infof("[%s](%d/%d/%v) update chain config, TimeoutPropose: %v,"+
			" TimeoutProposeDelta: %v,"+" parse TimeoutProposeDelta error: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			consensus.TimeoutPropose, consensus.TimeoutProposeDelta, err)
	}
	return
}

func (consensus *ConsensusTBFTImpl) extractBlocksPerProposer(value string) (tbftBlocksPerProposer uint64, err error) {
	if tbftBlocksPerProposer, err = strconv.ParseUint(value, 10, 32); err != nil {
		consensus.logger.Infof("[%s](%d/%d/%v) update chain config, parse BlocksPerProposer error: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return
	}
	if tbftBlocksPerProposer <= 0 {
		err = fmt.Errorf("invalid TBFT_blocks_per_proposer: %d", tbftBlocksPerProposer)
		return
	}
	return
}

func (consensus *ConsensusTBFTImpl) handle() {
	consensus.logger.Infof("[%s] handle start", consensus.Id)
	defer consensus.logger.Infof("[%s] handle end", consensus.Id)

	loop := true
	for loop {
		select {
		case proposedBlock := <-consensus.proposedBlockC:
			consensus.handleProposedBlock(proposedBlock, false)
		case result := <-consensus.verifyResultC:
			consensus.handleVerifyResult(result, false)
		case height := <-consensus.blockHeightC:
			consensus.handleBlockHeight(height)
		case msg := <-consensus.externalMsgC:
			consensus.logger.Debugf("[%s] receive from externalMsgC %s, size: %d", consensus.Id, msg.Type,
				proto.Size(msg))
			consensus.handleConsensusMsg(msg)
		case msg := <-consensus.internalMsgC:
			consensus.logger.Debugf("[%s] receive from internalMsgC %s, size: %d", consensus.Id, msg.Type,
				proto.Size(msg))
			consensus.handleConsensusMsg(msg)
		case ti := <-consensus.timeScheduler.GetTimeoutC():
			consensus.handleTimeout(ti, false)
		case <-consensus.closeC:
			loop = false
		}
	}
}

func (consensus *ConsensusTBFTImpl) handleProposedBlock(proposedBlock *consensuspb.ProposalBlock, replayMode bool) {
	consensus.Lock()
	defer consensus.Unlock()

	block := proposedBlock.Block
	consensus.logger.Debugf("[%s](%d/%d/%s) receive proposal from core engine (%d/%x/%d), isProposer: %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step,
		block.Header.BlockHeight, block.Header.BlockHash, proto.Size(block), consensus.isProposer(consensus.Height,
			consensus.Round),
	)

	if block.Header.BlockHeight != consensus.Height {
		consensus.logger.Errorf("[%s](%d/%d/%v) handle proposed block failed,"+
			" receive block from invalid height: %d",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, block.Header.BlockHeight)
		return
	}

	if !consensus.isProposer(consensus.Height, consensus.Round) {
		consensus.logger.Warnf("[%s](%d/%d/%s) receive proposal from core engine (%d/%x), but isProposer: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			block.Header.BlockHeight, block.Header.BlockHash, consensus.isProposer(consensus.Height, consensus.Round),
		)
		return
	}

	if consensus.Step != tbftpb.Step_PROPOSE {
		consensus.logger.Warnf("[%s](%d/%d/%s) receive proposal from core engine (%d/%x), step error",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			block.Header.BlockHeight, block.Header.BlockHash,
		)
		return
	}

	// add DPoS consensus args in block
	if consensus.dpos != nil {
		if err := consensus.dpos.CreateDPoSRWSet(block.Header.PreBlockHash, proposedBlock); err != nil {
			consensus.logger.Errorf("[%s](%d/%d/%s) Create DPoS RWSets failed, reason: %s",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
			return
		}
	}

	// Add hash and signature to block
	hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
	if err != nil {
		consensus.logger.Errorf("[%s]sign block failed, %s", consensus.Id, err)
		return
	}
	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig
	consensus.logger.Infof("[%s]create proposal block[%d:%x] success",
		consensus.Id, block.Header.BlockHeight, block.Header.BlockHash)

	// Add proposal
	proposal := NewProposal(consensus.Id, consensus.Height, consensus.Round, -1, block)
	err = consensus.signProposal(proposal)
	if err != nil {
		consensus.logger.Errorf("sign proposal err %s", err)
		return
	}
	consensus.Proposal = proposal

	if !replayMode {
		consensus.saveWalEntry(consensus.Proposal)
	}
	// prevote
	consensus.enterPrevote(consensus.Height, consensus.Round)
}

func (consensus *ConsensusTBFTImpl) handleVerifyResult(verifyResult *consensuspb.VerifyResult, replayMode bool) {
	consensus.Lock()
	defer consensus.Unlock()

	height := verifyResult.VerifiedBlock.Header.BlockHeight
	hash := verifyResult.VerifiedBlock.Header.BlockHash

	consensus.logger.Infof("[%s](%d/%d/%s) receive verify result (%d/%x) %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step,
		height, hash, verifyResult.Code)

	if consensus.VerifingProposal == nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) receive verify result failed, (%d/%x) %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			height, hash, verifyResult.Code,
		)
		return
	}

	if consensus.Height == height &&
		consensus.Round == consensus.VerifingProposal.Round &&
		verifyResult.Code == consensuspb.VerifyResult_FAIL {
		consensus.logger.Warnf("[%s](%d/%d/%s) %x receive verify result (%d/%x) %v failed",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, consensus.VerifingProposal.Block.Header.BlockHash,
			height, hash, verifyResult.Code,
		)
		consensus.VerifingProposal = nil
		go consensus.gossip.triggerEvent()
		return
	}

	// if there are quorum pre_commit vote, then commit block
	if bytes.Equal(consensus.VerifingProposal.Block.Header.BlockHash, hash) {
		if consensus.heightRoundVoteSet != nil && consensus.heightRoundVoteSet.precommits(consensus.Round) != nil {
			voteSet := consensus.heightRoundVoteSet.precommits(consensus.Round)
			quorumHash, ok := voteSet.twoThirdsMajority()
			//if ok && bytes.Compare(quorumHash, consensus.VerifingProposal.Block.Header.BlockHash) == 0 {
			if ok && bytes.Equal(quorumHash, consensus.VerifingProposal.Block.Header.BlockHash) {
				consensus.Proposal = consensus.VerifingProposal
				if !replayMode {
					consensus.saveWalEntry(consensus.Proposal)
				}

				qc := mustMarshal(voteSet.ToProto())
				if consensus.Proposal.Block.AdditionalData == nil {
					consensus.Proposal.Block.AdditionalData = &common.AdditionalData{
						ExtraData: make(map[string][]byte),
					}
				}
				consensus.Proposal.Block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc
				// Commit block to core engine
				consensus.commitBlock(consensus.Proposal.Block)
				return
			}
		}
	}

	if consensus.Step != tbftpb.Step_PROPOSE ||
		!bytes.Equal(consensus.VerifingProposal.Block.Header.BlockHash, hash) {
		consensus.logger.Warnf("[%s](%d/%d/%s) %x receive verify result (%d/%x) error",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, consensus.VerifingProposal.Block.Header.BlockHash,
			height, hash,
		)
		consensus.VerifingProposal = nil
		go consensus.gossip.triggerEvent()
		return
	}

	if consensus.chainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_DPOS {
		if err := consensus.dpos.VerifyConsensusArgs(verifyResult.VerifiedBlock, verifyResult.TxsRwSet); err != nil {
			consensus.logger.Warnf("verify block DPoS consensus failed, reason: %s", err)
			return
		}
	}

	consensus.Proposal = consensus.VerifingProposal
	if !replayMode {
		consensus.saveWalEntry(consensus.Proposal)
	}
	// Prevote
	consensus.enterPrevote(consensus.Height, consensus.Round)
}

func (consensus *ConsensusTBFTImpl) handleBlockHeight(height uint64) {
	consensus.Lock()
	defer consensus.Unlock()

	consensus.logger.Infof("[%s](%d/%d/%s) receive block height %d",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, height)

	// Outdated block height event
	if consensus.Height > height {
		return
	}

	consensus.logger.Infof("[%s](%d/%d/%s) enterNewHeight because receiving block height %d",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, height)
	consensus.enterNewHeight(height+1, false)
}

func (consensus *ConsensusTBFTImpl) procPropose(msg *tbftpb.TBFTMsg) {
	proposalProto := new(tbftpb.Proposal)
	mustUnmarshal(msg.Msg, proposalProto)

	consensus.logger.Debugf("[%s](%d/%d/%s) receive proposal from %s(%d/%d) (%d/%x/%d)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step,
		proposalProto.Voter, proposalProto.Height, proposalProto.Round,
		proposalProto.Block.Header.BlockHeight, proposalProto.Block.Header.BlockHash, proto.Size(proposalProto.Block),
	)

	if err := consensus.verifyProposal(proposalProto); err != nil {
		consensus.logger.Debugf("[%s](%d/%d/%s) verify proposal error: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			err,
		)
		return
	}

	proposal := NewProposalFromProto(proposalProto)
	if proposal == nil || proposal.Block == nil {
		consensus.logger.Debugf("[%s](%d/%d/%s) receive invalid proposal because nil proposal",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step)
		return
	}

	height := proposal.Block.Header.BlockHeight
	hash := proposal.Block.Header.BlockHash
	consensus.logger.Debugf("[%s](%d/%d/%s) receive propose %s(%d/%d) hash: %x",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step,
		proposal.Voter, proposal.Height, proposal.Round, hash,
	)

	if !consensus.canReceiveProposal(height, proposal.Round) {
		consensus.logger.Debugf("[%s](%d/%d/%s) receive invalid proposal: %s(%d/%d)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			proposal.Voter, proposal.Height, proposal.Round,
		)
		return
	}

	proposer, _ := consensus.validatorSet.GetProposer(proposal.Height, proposal.Round)
	if proposer != proposal.Voter {
		consensus.logger.Infof("[%s](%d/%d/%s) proposer: %s, receive proposal from incorrect proposal: %s",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, proposer, proposal.Voter)
		return
	}

	if consensus.Proposal != nil {
		if bytes.Equal(consensus.Proposal.Block.Header.BlockHash, proposal.Block.Header.BlockHash) {
			consensus.logger.Infof("[%s](%d/%d/%s) receive duplicate proposal from proposer: %s(%x)",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, proposal.Voter, proposal.Block.Header.BlockHash)
		} else {
			consensus.logger.Infof("[%s](%d/%d/%s) receive unequal proposal from proposer: %s(%x)",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, consensus.Proposal.Block.Header.BlockHash,
				proposal.Voter, proposal.Block.Header.BlockHash)
		}
		return
	}

	if consensus.VerifingProposal != nil {
		if bytes.Equal(consensus.VerifingProposal.Block.Header.BlockHash, proposal.Block.Header.BlockHash) {
			consensus.logger.Infof("[%s](%d/%d/%s) receive proposal which is verifying from proposer: %s(%x)",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, proposal.Voter, proposal.Block.Header.BlockHash)
		} else {
			consensus.logger.Infof("[%s](%d/%d/%s) receive unequal proposal with verifying proposal from proposer: %s(%x)",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, consensus.VerifingProposal.Block.Header.BlockHash,
				proposal.Voter, proposal.Block.Header.BlockHash)
		}
		return
	}

	consensus.logger.Debugf("[%s](%d/%d/%s) send for verifying block: (%d-%x)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, proposal.Block.Header.BlockHeight,
		proposal.Block.Header.BlockHash)
	consensus.VerifingProposal = proposal
	consensus.msgbus.PublishSafe(msgbus.VerifyBlock, proposal.Block)
}

func (consensus *ConsensusTBFTImpl) canReceiveProposal(height uint64, round int32) bool {
	if consensus.Height != height || consensus.Round != round || consensus.Step != tbftpb.Step_PROPOSE {
		consensus.logger.Debugf("[%s](%d/%d/%s) receive invalid proposal: (%d/%d)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)
		return false
	}
	return true
}

func (consensus *ConsensusTBFTImpl) procPrevote(msg *tbftpb.TBFTMsg) {
	prevote := new(tbftpb.Vote)
	mustUnmarshal(msg.Msg, prevote)

	consensus.logger.Debugf("[%s](%d/%d/%s) receive prevote %s(%d/%d/%x)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step,
		prevote.Voter, prevote.Height, prevote.Round, prevote.Hash,
	)

	if consensus.Height != prevote.Height ||
		consensus.Round > prevote.Round ||
		(consensus.Round == prevote.Round && consensus.Step > tbftpb.Step_PREVOTE) {
		errMsg := fmt.Sprintf("[%s](%d/%d/%s) receive invalid vote %s(%d/%d/%s)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			prevote.Voter, prevote.Height, prevote.Round, prevote.Type)
		consensus.logger.Debugf(errMsg)
		return
	}

	if prevote.Voter != consensus.Id {
		err := consensus.verifyVote(prevote)
		if err != nil {
			consensus.logger.Errorf("[%s](%d/%d/%s) receive prevote %s(%d/%d/%x), verifyVote failed: %v",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step,
				prevote.Voter, prevote.Height, prevote.Round, prevote.Hash, err,
			)
			return
		}
	}

	vote := NewVoteFromProto(prevote)
	err := consensus.addVote(vote, false)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) addVote %s(%d/%d/%s) failed, %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			prevote.Voter, prevote.Height, prevote.Round, prevote.Type, err,
		)
		return
	}
}

func (consensus *ConsensusTBFTImpl) procPrecommit(msg *tbftpb.TBFTMsg) {
	precommit := new(tbftpb.Vote)
	mustUnmarshal(msg.Msg, precommit)

	consensus.logger.Debugf("[%s](%d/%d/%s) receive precommit %s(%d/%d)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step,
		precommit.Voter, precommit.Height, precommit.Round,
	)

	if consensus.Height != precommit.Height ||
		consensus.Round > precommit.Round ||
		(consensus.Round == precommit.Round && consensus.Step > tbftpb.Step_PRECOMMIT) {
		consensus.logger.Debugf("[%s](%d/%d/%s) receive invalid precommit %s(%d/%d)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			precommit.Voter, precommit.Height, precommit.Round,
		)
		return
	}

	if precommit.Voter != consensus.Id {
		err := consensus.verifyVote(precommit)
		if err != nil {
			consensus.logger.Errorf("[%s](%d/%d/%s) receive precommit %s(%d/%d/%x), verifyVote failed, %v",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step,
				precommit.Voter, precommit.Height, precommit.Round, precommit.Hash, err,
			)
			return
		}
	}

	vote := NewVoteFromProto(precommit)
	err := consensus.addVote(vote, false)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) addVote %s(%d/%d/%s) failed, %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			precommit.Voter, precommit.Height, precommit.Round, precommit.Type, err,
		)
		return
	}
}

func (consensus *ConsensusTBFTImpl) handleConsensusMsg(msg *tbftpb.TBFTMsg) {
	consensus.Lock()
	defer consensus.Unlock()

	switch msg.Type {
	case tbftpb.TBFTMsgType_MSG_PROPOSE:
		consensus.procPropose(msg)
	case tbftpb.TBFTMsgType_MSG_PREVOTE:
		consensus.procPrevote(msg)
	case tbftpb.TBFTMsgType_MSG_PRECOMMIT:
		consensus.procPrecommit(msg)
	case tbftpb.TBFTMsgType_MSG_STATE:
		// Async is ok
		go consensus.gossip.onRecvState(msg)
	}
}

// handleTimeout handles timeout event
func (consensus *ConsensusTBFTImpl) handleTimeout(ti timeoutInfo, replayMode bool) {
	consensus.Lock()
	defer consensus.Unlock()

	consensus.logger.Infof("[%s](%d/%d/%s) handleTimeout ti: %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, ti)
	if !replayMode {
		consensus.saveWalEntry(ti)
	}
	switch ti.Step {
	case tbftpb.Step_PREVOTE:
		consensus.enterPrevote(ti.Height, ti.Round)
	case tbftpb.Step_PRECOMMIT:
		consensus.enterPrecommit(ti.Height, ti.Round)
	case tbftpb.Step_COMMIT:
		consensus.enterCommit(ti.Height, ti.Round)
	}
}

func (consensus *ConsensusTBFTImpl) commitBlock(block *common.Block) {
	consensus.logger.Debugf("[%s] commitBlock to %d-%x", consensus.Id, block.Header.BlockHeight, block.Header.BlockHash)
	//Simulate a malicious node which commit block without notification
	if localconf.ChainMakerConfig.DebugConfig.IsCommitWithoutPublish {
		consensus.logger.Debugf("[%s](%d/%d/%s) switch IsCommitWithoutPublish: %v, commitBlock block(%d/%x)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsCommitWithoutPublish,
			block.Header.BlockHeight, block.Header.BlockHash,
		)
	} else {
		consensus.logger.Debugf("[%s](%d/%d/%s) commitBlock block(%d/%x)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			block.Header.BlockHeight, block.Header.BlockHash,
		)
		consensus.msgbus.Publish(msgbus.CommitBlock, block)
	}
}

// ProposeTimeout returns timeout to wait for proposing at `round`
func (consensus *ConsensusTBFTImpl) ProposeTimeout(round int32) time.Duration {
	return time.Duration(
		consensus.TimeoutPropose.Nanoseconds()+consensus.TimeoutProposeDelta.Nanoseconds()*int64(round),
	) * time.Nanosecond
}

// PrevoteTimeout returns timeout to wait for prevoting at `round`
func (consensus *ConsensusTBFTImpl) PrevoteTimeout(round int32) time.Duration {
	return time.Duration(
		TimeoutPrevote.Nanoseconds()+TimeoutPrevoteDelta.Nanoseconds()*int64(round),
	) * time.Nanosecond
}

// PrecommitTimeout returns timeout to wait for precommiting at `round`
func (consensus *ConsensusTBFTImpl) PrecommitTimeout(round int32) time.Duration {
	return time.Duration(
		TimeoutPrecommit.Nanoseconds()+TimeoutPrecommitDelta.Nanoseconds()*int64(round),
	) * time.Nanosecond
}

// CommitTimeout returns timeout to wait for precommiting at `round`
func (consensus *ConsensusTBFTImpl) CommitTimeout(round int32) time.Duration {
	return time.Duration(TimeoutCommit.Nanoseconds()*int64(round)) * time.Nanosecond
}

// AddTimeout adds timeout event to timeScheduler
func (consensus *ConsensusTBFTImpl) AddTimeout(duration time.Duration, height uint64, round int32,
	step tbftpb.Step) {
	consensus.timeScheduler.AddTimeoutInfo(timeoutInfo{duration, height, round, step})
}

// addVote adds `vote` to heightVoteSet
func (consensus *ConsensusTBFTImpl) addVote(vote *Vote, replayMode bool) error {
	consensus.logger.Debugf("[%s](%d/%d/%s) addVote %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)

	added, err := consensus.heightRoundVoteSet.addVote(vote)
	if !added || err != nil {
		consensus.logger.Infof("[%s](%d/%d/%s) addVote %v, added: %v, err: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote, added, err)
		return err
	}

	if !replayMode {
		consensus.saveWalEntry(vote)
	}

	switch vote.Type {
	case tbftpb.VoteType_VOTE_PREVOTE:
		consensus.addPrevoteVote(vote)
	case tbftpb.VoteType_VOTE_PRECOMMIT:
		consensus.addPrecommitVote(vote)
	}

	// Trigger gossip when receive self vote
	if consensus.Id == vote.Voter {
		go consensus.gossip.triggerEvent()
	}
	return nil
}

func (consensus *ConsensusTBFTImpl) addPrevoteVote(vote *Vote) {
	if consensus.Step != tbftpb.Step_PREVOTE {
		consensus.logger.Infof("[%s](%d/%d/%s) addVote prevote %v at inappropriate step",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)
		return
	}
	voteSet := consensus.heightRoundVoteSet.prevotes(vote.Round)
	hash, ok := voteSet.twoThirdsMajority()
	if !ok {
		consensus.logger.Debugf("[%s](%d/%d/%s) addVote %v without majority",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)

		if consensus.Round == vote.Round && voteSet.hasTwoThirdsAny() {
			consensus.logger.Infof("[%s](%d/%d/%s) addVote %v with hasTwoThirdsAny",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)
			consensus.enterPrecommit(consensus.Height, consensus.Round)
		} else if consensus.Round == vote.Round && voteSet.hasTwoThirdsNoMajority() && !consensus.TriggeredTimeoutPrevote {
			// add the prevote timeout event
			consensus.logger.Infof("[%s](%d/%d/%s) addVote %v with hasTwoThirdsAny, PrevoteTimeout is igniting",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)
			consensus.AddTimeout(consensus.PrevoteTimeout(consensus.Round), consensus.Height,
				consensus.Round, tbftpb.Step_PRECOMMIT)
			consensus.TriggeredTimeoutPrevote = true
		}
		return
	}
	// Upon >2/3 prevotes, Step into StepPrecommit
	if consensus.Proposal != nil {
		if !bytes.Equal(hash, consensus.Proposal.Block.Header.BlockHash) {
			consensus.logger.Warnf("[%s](%d/%d/%s) block matched failed, receive valid block: %x,"+
				"but unmatched with proposal: %x",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, hash, consensus.Proposal.Block.Header.BlockHash)
		}
		consensus.enterPrecommit(consensus.Height, consensus.Round)
	} else {
		if isNilHash(hash) {
			consensus.enterPrecommit(consensus.Height, consensus.Round)
		} else {
			consensus.logger.Warnf("[%s](%d/%d/%s) add vote failed, receive valid block: %x, but proposal is nil",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, hash)
		}
	}
}

func (consensus *ConsensusTBFTImpl) addPrecommitVote(vote *Vote) {
	if consensus.Step != tbftpb.Step_PRECOMMIT {
		consensus.logger.Infof("[%s](%d/%d/%s) addVote precommit %v at inappropriate step",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)
		return
	}

	voteSet := consensus.heightRoundVoteSet.precommits(vote.Round)
	hash, ok := voteSet.twoThirdsMajority()
	if !ok {
		consensus.logger.Debugf("[%s](%d/%d/%s) addVote %v without majority",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)

		if consensus.Round == vote.Round && voteSet.hasTwoThirdsAny() {
			consensus.logger.Infof("[%s](%d/%d/%s) addVote %v with hasTwoThirdsAny",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)
			consensus.enterCommit(consensus.Height, consensus.Round)
		} else if consensus.Round == vote.Round && voteSet.hasTwoThirdsNoMajority() && !consensus.TriggeredTimeoutPrecommit {
			// add the precommit timeout event
			consensus.logger.Infof("[%s](%d/%d/%s) addVote %v with hasTwoThirdsNoMajority, PrecommitTimeout is igniting",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, vote)
			consensus.AddTimeout(consensus.PrecommitTimeout(consensus.Round), consensus.Height,
				consensus.Round, tbftpb.Step_COMMIT)
			consensus.TriggeredTimeoutPrecommit = true
		}
		return
	}
	// Upon >2/3 precommits, Step into StepCommit
	if consensus.Proposal != nil {
		if isNilHash(hash) || bytes.Equal(hash, consensus.Proposal.Block.Header.BlockHash) {
			consensus.enterCommit(consensus.Height, consensus.Round)
		} else {
			consensus.logger.Errorf("[%s](%d/%d/%s) block matched failed, receive valid block: %x,"+
				" but unmatched with proposal: %x",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, hash, consensus.Proposal.Block.Header.BlockHash)
		}
	} else {
		if !isNilHash(hash) {
			consensus.logger.Errorf("[%s](%d/%d/%s) receive valid block: %x, but proposal is nil",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, hash)
			return
		}
		consensus.enterCommit(consensus.Height, consensus.Round)
	}
}

// enterNewHeight enter `height`
func (consensus *ConsensusTBFTImpl) enterNewHeight(height uint64, replayMode bool) {
	consensus.logger.Infof("[%s]attempt enter new height to (%d)", consensus.Id, height)
	if consensus.Height >= height {
		consensus.logger.Errorf("[%s](%v/%v/%v) invalid enter invalid new height to (%v)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height)
		return
	}
	addedValidators, removedValidators, err := consensus.updateChainConfig()
	if err != nil {
		consensus.logger.Errorf("[%s](%v/%v/%v) update chain config failed: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
	}

	if !replayMode {
		lastIndex, err1 := consensus.wal.LastIndex()
		if err1 != nil {
			consensus.logger.Fatalf("[%s](%d/%d/%s) get last index error: %v",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		}

		consensus.logger.Infof("consensus.Id:[%s] consensus.Height:%d walLastIndex:%d ",
			consensus.Id, consensus.Height, lastIndex)

		consensus.heightFirstIndex = lastIndex + 1

		err = consensus.deleteWalEntry(height, lastIndex)
		if err != nil {
			consensus.logger.Infof("[%s](%d/%d/%s) failed to delete wal log %v",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		}

	}

	err = consensus.gossip.addValidators(addedValidators)
	if err != nil {
		consensus.logger.Infof("[%s](%d/%d/%s) failed to addValidators %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
	}
	err = consensus.gossip.removeValidators(removedValidators)
	if err != nil {
		consensus.logger.Infof("[%s](%d/%d/%s) failed to removeValidators %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
	}

	consensus.consensusStateCache.addConsensusState(consensus.ConsensusState)
	consensus.ConsensusState = NewConsensusState(consensus.logger, consensus.Id)
	consensus.Height = height
	consensus.Round = 0
	consensus.Step = tbftpb.Step_NEW_HEIGHT
	consensus.heightRoundVoteSet = newHeightRoundVoteSet(
		consensus.logger, consensus.Height, consensus.Round, consensus.validatorSet)
	consensus.metrics = newHeightMetrics(consensus.Height)
	consensus.metrics.SetEnterNewHeightTime()
	consensus.enterNewRound(height, 0)
}

// enterNewRound enter `round` at `height`
func (consensus *ConsensusTBFTImpl) enterNewRound(height uint64, round int32) {
	consensus.logger.Debugf("[%s] attempt enterNewRound to (%d/%d)", consensus.Id, height, round)
	if consensus.Height > height ||
		consensus.Round > round ||
		(consensus.Round == round && consensus.Step != tbftpb.Step_NEW_HEIGHT) {
		consensus.logger.Infof("[%s](%v/%v/%v) enter new round invalid(%v/%v)",

			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)
		return
	}
	consensus.Height = height
	consensus.Round = round
	consensus.Step = tbftpb.Step_NEW_ROUND
	consensus.TriggeredTimeoutPrevote = false
	consensus.TriggeredTimeoutPrecommit = false
	consensus.Proposal = nil
	consensus.VerifingProposal = nil
	consensus.metrics.SetEnterNewRoundTime(consensus.Round)
	consensus.enterPropose(height, round)
}

func (consensus *ConsensusTBFTImpl) enterPropose(height uint64, round int32) {
	consensus.logger.Debugf("[%s] attempt enterPropose to (%d/%d)", consensus.Id, height, round)
	if consensus.Height != height ||
		consensus.Round > round ||
		(consensus.Round == round && consensus.Step != tbftpb.Step_NEW_ROUND) {
		consensus.logger.Infof("[%s](%v/%v/%v) enter invalid propose(%v/%v)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)
		return
	}

	// Step into Propose
	consensus.Step = tbftpb.Step_PROPOSE
	consensus.metrics.SetEnterProposalTime(consensus.Round)
	consensus.AddTimeout(consensus.ProposeTimeout(round), height, round, tbftpb.Step_PREVOTE)

	//Simulate a node which delay when Propose
	if localconf.ChainMakerConfig.DebugConfig.IsProposeDelay {
		consensus.logger.Infof("[%s](%v/%v/%v) switch IsProposeDelay: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsProposeDelay)
		time.Sleep(2 * time.Second)
	}

	//Simulate a malicious node which think itself a proposal
	if localconf.ChainMakerConfig.DebugConfig.IsProposeMultiNodeDuplicately {
		consensus.logger.Infof("[%s](%v/%v/%v) switch IsProposeMultiNodeDuplicately: %v, it always propose",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsProposeMultiNodeDuplicately)
		consensus.sendProposeState(true)
	}

	if consensus.isProposer(height, round) {
		consensus.sendProposeState(true)
	}

	go consensus.gossip.triggerEvent()
}

// enterPrevote enter `prevote` phase
func (consensus *ConsensusTBFTImpl) enterPrevote(height uint64, round int32) {
	if consensus.Height != height ||
		consensus.Round > round ||
		(consensus.Round == round && consensus.Step != tbftpb.Step_PROPOSE) {
		consensus.logger.Infof("[%s](%v/%v/%v) enter invalid prevote(%v/%v)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)
		return
	}

	consensus.logger.Infof("[%s](%v/%v/%v) enter prevote(%v/%v)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)

	// Enter StepPrevote
	consensus.Step = tbftpb.Step_PREVOTE
	consensus.metrics.SetEnterPrevoteTime(consensus.Round)

	// Disable propose
	consensus.sendProposeState(false)

	var hash = nilHash
	if consensus.Proposal != nil {
		hash = consensus.Proposal.Block.Header.BlockHash
	}

	//Simulate a node which send an invalid(hash=NIL) Prevote
	if localconf.ChainMakerConfig.DebugConfig.IsPrevoteInvalid {
		consensus.logger.Infof("[%s](%v/%v/%v) switch IsPrevoteInvalid: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsPrevoteInvalid)
		hash = nil
	}

	//Simulate a node which delay when Propose
	if localconf.ChainMakerConfig.DebugConfig.IsPrevoteDelay {
		consensus.logger.Infof("[%s](%v/%v/%v) switch PrevoteDelay: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsPrevoteDelay)
		time.Sleep(2 * time.Second)
	}

	// Broadcast prevote
	// prevote := createPrevoteMsg(consensus.Id, consensus.Height, consensus.Round, hash)
	prevote := NewVote(tbftpb.VoteType_VOTE_PREVOTE, consensus.Id, consensus.Height, consensus.Round, hash)
	if localconf.ChainMakerConfig.DebugConfig.IsPrevoteOldHeight {
		consensus.logger.Infof("[%s](%v/%v/%v) switch IsPrevoteOldHeight: %v, prevote old height: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsPrevoteOldHeight, consensus.Height-1)
		prevote = NewVote(tbftpb.VoteType_VOTE_PREVOTE, consensus.Id, consensus.Height-1, consensus.Round, hash)
	}
	err := consensus.signVote(prevote)
	if err != nil {
		consensus.logger.Errorf("enter Prevote sign Vote error: %s", err)
	}
	prevoteProto := createPrevoteMsg(prevote)

	consensus.logger.Debugf("internalMsgC len: %d", len(consensus.internalMsgC))
	consensus.internalMsgC <- prevoteProto
}

// enterPrecommit enter `precommit` phase
func (consensus *ConsensusTBFTImpl) enterPrecommit(height uint64, round int32) {
	if consensus.Height != height ||
		consensus.Round > round ||
		(consensus.Round == round && consensus.Step != tbftpb.Step_PREVOTE) {
		consensus.logger.Infof("[%s](%v/%v/%v) enter precommit invalid(%v/%v)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)
		return
	}

	consensus.logger.Infof("[%s](%v/%v/%v) enter precommit(%v/%v)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)

	// Enter StepPrecommit
	consensus.Step = tbftpb.Step_PRECOMMIT
	consensus.metrics.SetEnterPrecommitTime(consensus.Round)

	voteSet := consensus.heightRoundVoteSet.prevotes(consensus.Round)
	hash, ok := voteSet.twoThirdsMajority()
	if !ok {
		if voteSet.hasTwoThirdsAny() || voteSet.hasTwoThirdsNoMajority() {
			hash = nilHash
			consensus.logger.Infof("[%s](%v/%v/%v) enter precommit to nil because hasTwoThirdsAny "+
				"or hasTwoThirdsNoMajority", consensus.Id, consensus.Height, consensus.Round, consensus.Step)
		} else {
			panic("this should not happen")
		}
	}

	//Simulate a node which send an invalid(hash=NIL) Precommit
	if localconf.ChainMakerConfig.DebugConfig.IsPrecommitInvalid {
		consensus.logger.Infof("[%s](%v/%v/%v) switch IsPrecommitInvalid: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsPrecommitInvalid)
		hash = nil
	}

	// Broadcast precommit
	precommit := NewVote(tbftpb.VoteType_VOTE_PRECOMMIT, consensus.Id, consensus.Height, consensus.Round, hash)
	if localconf.ChainMakerConfig.DebugConfig.IsPrecommitOldHeight {
		consensus.logger.Infof("[%s](%d/%d/%v) switch IsPrecommitOldHeight: %v, precommit old height: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsPrecommitOldHeight, consensus.Height-1)
		precommit = NewVote(tbftpb.VoteType_VOTE_PRECOMMIT, consensus.Id, consensus.Height-1, consensus.Round, hash)
	}
	err := consensus.signVote(precommit)
	if err != nil {
		consensus.logger.Errorf("enter Precommit sign Vote error: %s", err)
	}
	precommitProto := createPrecommitMsg(precommit)

	//Simulate a node which delay when Precommit
	if localconf.ChainMakerConfig.DebugConfig.IsPrecommitDelay {
		consensus.logger.Infof("[%s](%v/%v/%v) switch IsPrecommitDelay: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			localconf.ChainMakerConfig.DebugConfig.IsPrecommitDelay)
		time.Sleep(2 * time.Second)
	}

	consensus.internalMsgC <- precommitProto
}

// enterCommit enter `Commit` phase
func (consensus *ConsensusTBFTImpl) enterCommit(height uint64, round int32) {
	if consensus.Height != height ||
		consensus.Round > round ||
		(consensus.Round == round && consensus.Step != tbftpb.Step_PRECOMMIT) {
		consensus.logger.Infof("[%s](%d/%d/%s) enterCommit invalid(%v/%v)",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)
		return
	}

	consensus.logger.Infof("[%s](%d/%d/%s) enterCommit(%v/%v)",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, height, round)

	// Enter StepCommit
	consensus.Step = tbftpb.Step_COMMIT
	consensus.metrics.SetEnterCommitTime(consensus.Round)
	consensus.logger.Infof("[%s] consensus cost: %s", consensus.Id, consensus.metrics.roundString(consensus.Round))

	voteSet := consensus.heightRoundVoteSet.precommits(consensus.Round)
	hash, ok := voteSet.twoThirdsMajority()
	if !isNilHash(hash) && !ok {
		// This should not happen
		panic(fmt.Errorf("[%s]-%x, enter commit failed, without majority", consensus.Id, hash))
	}

	if isNilHash(hash) {
		// consensus.AddTimeout(consensus.CommitTimeout(round), consensus.Height, round+1, tbftpb.Step_NEW_ROUND)
		consensus.enterNewRound(consensus.Height, round+1)
	} else {
		// Proposal block hash must be match with precommited block hash
		if !bytes.Equal(hash, consensus.Proposal.Block.Header.BlockHash) {
			// This should not happen
			panic(fmt.Errorf("[%s] block match failed, unmatch precommit hash: %x with proposal hash: %x",
				consensus.Id, hash, consensus.Proposal.Block.Header.BlockHash))
		}

		qc := mustMarshal(voteSet.ToProto())
		if consensus.Proposal.Block.AdditionalData == nil {
			consensus.Proposal.Block.AdditionalData = &common.AdditionalData{
				ExtraData: make(map[string][]byte),
			}
		}
		consensus.Proposal.Block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey] = qc

		// Commit block to core engine
		consensus.commitBlock(consensus.Proposal.Block)
	}
}

func isNilHash(hash []byte) bool {
	return len(hash) == 0 || bytes.Equal(hash, nilHash)
}

// isProposer returns true if this node is proposer at `height` and `round`,
// and returns false otherwise
func (consensus *ConsensusTBFTImpl) isProposer(height uint64, round int32) bool {
	proposer, _ := consensus.validatorSet.GetProposer(height, round)

	return proposer == consensus.Id
}

func (consensus *ConsensusTBFTImpl) ToProto() *tbftpb.ConsensusState {
	consensus.RLock()
	defer consensus.RUnlock()
	msg := proto.Clone(consensus.toProto())
	return msg.(*tbftpb.ConsensusState)
}

func (consensus *ConsensusTBFTImpl) ToGossipStateProto() *tbftpb.GossipState {
	consensus.RLock()
	defer consensus.RUnlock()

	var proposal []byte
	if consensus.Proposal != nil {
		proposal = consensus.Proposal.Block.Header.BlockHash
	}

	var verifingProposal []byte
	if consensus.Proposal != nil {
		verifingProposal = consensus.Proposal.Block.Header.BlockHash
	}

	gossipProto := &tbftpb.GossipState{
		Id:               consensus.Id,
		Height:           consensus.Height,
		Round:            consensus.Round,
		Step:             consensus.Step,
		Proposal:         proposal,
		VerifingProposal: verifingProposal,
		RoundVoteSet:     consensus.heightRoundVoteSet.getRoundVoteSet(consensus.Round).ToProto(),
	}
	msg := proto.Clone(gossipProto)
	return msg.(*tbftpb.GossipState)
}

func (consensus *ConsensusTBFTImpl) signProposal(proposal *Proposal) error {
	proposalBytes := mustMarshal(proposal.ToProto())
	sig, err := consensus.singer.Sign(consensus.chainConf.ChainConfig().Crypto.Hash, proposalBytes)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%v) sign proposal %s(%d/%d)-%x failed: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			proposal.Voter, proposal.Height, proposal.Round, proposal.Block.Header.BlockHash, err)
		return err
	}

	serializeMember, err := consensus.singer.GetMember()
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%v) get serialize member failed: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return err
	}
	proposal.Endorsement = &common.EndorsementEntry{
		Signer:    serializeMember,
		Signature: sig,
	}
	return nil
}

func (consensus *ConsensusTBFTImpl) signVote(vote *Vote) error {
	voteBytes := mustMarshal(vote.ToProto())
	sig, err := consensus.singer.Sign(consensus.chainConf.ChainConfig().Crypto.Hash, voteBytes)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%v) sign vote %s(%d/%d)-%x failed: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			vote.Voter, vote.Height, vote.Round, vote.Hash, err)
		return err
	}

	serializeMember, err := consensus.singer.GetMember()
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%v) get serialize member failed: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return err
	}
	vote.Endorsement = &common.EndorsementEntry{
		Signer:    serializeMember,
		Signature: sig,
	}
	return nil
}

func (consensus *ConsensusTBFTImpl) verifyProposal(proposal *tbftpb.Proposal) error {
	// Verified by idmgmt
	proposalCopy := proto.Clone(proposal)
	proposalCopy.(*tbftpb.Proposal).Endorsement = nil
	proposalCopy.(*tbftpb.Proposal).Block.AdditionalData = nil
	message := mustMarshal(proposalCopy)
	principal, err := consensus.ac.CreatePrincipal(
		protocol.ResourceNameConsensusNode,
		[]*common.EndorsementEntry{proposal.Endorsement},
		message,
	)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) receive proposal new principal failed, %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return err
	}

	result, err := consensus.ac.VerifyPrincipal(principal)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) receive proposal VerifyPolicy result: %v, error %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, result, err)
		return err
	}

	if !result {
		consensus.logger.Errorf("[%s](%d/%d/%s) receive proposal VerifyPolicy result: %v, error %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, result, err)
		return fmt.Errorf("VerifyPolicy result: %v", result)
	}

	return nil
}

func (consensus *ConsensusTBFTImpl) verifyVote(voteProto *tbftpb.Vote) error {
	voteProtoCopy := proto.Clone(voteProto)
	vote, ok := voteProtoCopy.(*tbftpb.Vote)
	if !ok {
		return fmt.Errorf("interface to *tbftpb.Vote failed")
	}
	vote.Endorsement = nil
	message := mustMarshal(vote)

	principal, err := consensus.ac.CreatePrincipal(
		protocol.ResourceNameConsensusNode,
		[]*common.EndorsementEntry{voteProto.Endorsement},
		message,
	)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) verifyVote new policy failed %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return err
	}

	result, err := consensus.ac.VerifyPrincipal(principal)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) verifyVote verify policy failed %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return err
	}

	if !result {
		consensus.logger.Errorf("[%s](%d/%d/%s) verifyVote verify policy result: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, result)
		return fmt.Errorf("verifyVote result: %v", result)
	}

	member, err := consensus.ac.NewMember(voteProto.Endorsement.Signer)
	if err != nil {
		consensus.logger.Errorf("[%s](%d/%d/%s) verifyVote new member failed %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, err)
		return err
	}

	var uid string
	chainConf := consensus.chainConf.ChainConfig()
	for _, v := range chainConf.TrustMembers {
		if v.MemberInfo == string(voteProto.Endorsement.Signer.MemberInfo) {
			uid = v.NodeId
			break
		}
	}

	if uid == "" {
		certId := member.GetMemberId()
		uid, err = consensus.netService.GetNodeUidByCertId(certId)
		if err != nil {
			consensus.logger.Errorf("[%s](%d/%d/%s) verifyVote certId: %v, GetNodeUidByCertId failed %v",
				consensus.Id, consensus.Height, consensus.Round, consensus.Step, certId, err)
			return err
		}
	}

	if uid != voteProto.Voter {
		consensus.logger.Errorf("[%s](%d/%d/%s) verifyVote failed, uid %s is not equal with voter %s",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step,
			uid, voteProto.Voter)
		return fmt.Errorf("verifyVote failed, unmatch uid: %v with vote: %v", uid, voteProto.Voter)
	}

	return nil
}

// nolint: deadcode, unused
func publicKeyValidateVoter(vote *tbftpb.Vote) error {
	signer := vote.Endorsement.Signer
	pk, err := asym.PublicKeyFromPEM(signer.MemberInfo)
	if err != nil {
		return fmt.Errorf("get pk failed from signer, reason: %s", err)
	}
	uid, err := helper.CreateLibp2pPeerIdWithPublicKey(pk)
	if err != nil {
		return fmt.Errorf("createLibp2pId with pubkey failed, reason: %s", err)
	}
	if uid != vote.Voter {
		return fmt.Errorf("verifyVote failed, unmatch uid: %v with vote: %v", uid, vote.Voter)
	}
	return nil
}

func (consensus *ConsensusTBFTImpl) getValidatorSet() *validatorSet {
	consensus.Lock()
	defer consensus.Unlock()
	return consensus.validatorSet
}

// saveWalEntry saves entry to Wal
func (consensus *ConsensusTBFTImpl) saveWalEntry(entry interface{}) {
	var walType tbftpb.WalEntryType
	var data []byte
	switch m := entry.(type) {
	case *Proposal:
		walType = tbftpb.WalEntryType_PROPOSAL_ENTRY
		data = mustMarshal(m.ToProto())
	case *Vote:
		walType = tbftpb.WalEntryType_VOTE_ENTRY
		data = mustMarshal(m.ToProto())
	case timeoutInfo:
		walType = tbftpb.WalEntryType_TIMEOUT_ENTRY
		data = mustMarshal(m.ToProto())
		consensus.logger.Debugf("[%s](%d/%d/%s) save wal timeout data length: %v, timeout: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, len(data), m)
	default:
		consensus.logger.Fatalf("[%s](%d/%d/%s) save wal of unknown type",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step)
	}
	lastIndex, err := consensus.wal.LastIndex()
	if err != nil {
		consensus.logger.Fatalf("[%s](%d/%d/%s) save wal type: %s get last index error: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, walType, err)
	}

	walEntry := tbftpb.WalEntry{
		Height:           consensus.Height,
		HeightFirstIndex: consensus.heightFirstIndex,
		Type:             walType,
		Data:             data,
	}

	log := mustMarshal(&walEntry)
	err = consensus.wal.Write(lastIndex+1, log)
	if err != nil {
		consensus.logger.Fatalf("[%s](%d/%d/%s) save wal type: %s write error: %v",
			consensus.Id, consensus.Height, consensus.Round, consensus.Step, walType, err)
	}
	consensus.logger.Debugf("[%s](%d/%d/%s) save wal type: %s data length: %v",
		consensus.Id, consensus.Height, consensus.Round, consensus.Step, walType, len(data))
}

// replayWal replays the wal when the node starting
func (consensus *ConsensusTBFTImpl) replayWal() error {
	currentHeight, err := consensus.ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}

	lastIndex, err := consensus.wal.LastIndex()
	if err != nil {
		return err
	}

	lastData, err := consensus.wal.Read(lastIndex)
	if err == wal.ErrNotFound {
		consensus.logger.Infof("[%s] replayWal can't found log entry in wal", consensus.Id)
		consensus.enterNewHeight(currentHeight+1, false)
		return nil
	}
	if err != nil {
		return err
	}

	lastEntry := &tbftpb.WalEntry{}
	mustUnmarshal(lastData, lastEntry)

	consensus.heightFirstIndex = lastEntry.HeightFirstIndex
	consensus.logger.Infof("[%s] replayWal lastIndex of wal: %d and HeightFirstIndex of wal: %d",
		consensus.Id, lastIndex, lastEntry.HeightFirstIndex)

	height := lastEntry.Height
	consensus.logger.Infof("[%s] replayWal chainHeight: %d and walHeight: %d",
		consensus.Id, currentHeight, height)

	if currentHeight < height-1 {
		consensus.logger.Fatalf("[%s] replay currentHeight: %v < height-1: %v, this should not happen",
			consensus.Id, currentHeight, height-1)
	}

	if currentHeight >= height {
		// consensus is slower than ledger
		consensus.enterNewHeight(currentHeight+1, false)
		return nil
	}
	go consensus.deleteInternalMsgC()

	// replay wal log, currentHeight=height-1
	consensus.enterNewHeight(height, true)
	for i := lastEntry.HeightFirstIndex; i <= lastIndex; i++ {
		if i == lastIndex {
			close(consensus.internalMsgCCloseC)
		}
		data, err := consensus.wal.Read(i)
		if err != nil {
			return err
		}
		entry := &tbftpb.WalEntry{}
		mustUnmarshal(data, entry)
		consensus.logger.Debugf("[%s] replay entry index: %d, entry height：%d，"+
			"HeightFirstIndex: %d，type: %s, Data.len: %d",
			consensus.Id, i, entry.Height, entry.HeightFirstIndex, entry.Type, len(entry.Data))
		switch entry.Type {
		case tbftpb.WalEntryType_TIMEOUT_ENTRY:
			timeoutInfoProto := new(tbftpb.TimeoutInfo)
			mustUnmarshal(entry.Data, timeoutInfoProto)
			timeoutInfo1 := newTimeoutInfoFromProto(timeoutInfoProto)
			consensus.handleTimeout(timeoutInfo1, true)
		case tbftpb.WalEntryType_PROPOSAL_ENTRY:
			proposalProto := new(tbftpb.Proposal)
			mustUnmarshal(entry.Data, proposalProto)
			proposal := NewProposalFromProto(proposalProto)

			if consensus.isProposer(consensus.Height, consensus.Round) {
				proposedBlock := &consensuspb.ProposalBlock{
					Block: proposal.Block,
				}
				consensus.handleProposedBlock(proposedBlock, true)
			} else {
				consensus.VerifingProposal = proposal
				verifyResult := &consensuspb.VerifyResult{
					VerifiedBlock: proposal.Block,
					Code:          consensuspb.VerifyResult_SUCCESS,
				}
				consensus.handleVerifyResult(verifyResult, true)
			}
		case tbftpb.WalEntryType_VOTE_ENTRY:
			voteProto := new(tbftpb.Vote)
			mustUnmarshal(entry.Data, voteProto)
			vote := NewVoteFromProto(voteProto)
			err := consensus.addVote(vote, true)
			if err != nil {
				errMsg := fmt.Sprintf("[%s](%d/%d/%s) addVote %s(%d/%d) failed, %v",
					consensus.Id, consensus.Height, consensus.Round, consensus.Step,
					vote.Voter, vote.Height, vote.Round, err)
				consensus.logger.Warnf(errMsg)
				//return errors.New(errMsg)
				// for jenkins-arm
			}
		}
	}

	return nil
}

func (consensus *ConsensusTBFTImpl) deleteInternalMsgC() {
	consensus.logger.Infof("[%s] delete internalMsgC start", consensus.Id)
	defer consensus.logger.Infof("[%s] delete internalMsgC end", consensus.Id)

	loop := true
	for loop {
		select {
		case msg := <-consensus.internalMsgC:
			consensus.logger.Debugf("[%s] delete internalMsgC msg Type: %s, size: %d", consensus.Id, msg.Type, proto.Size(msg))

		case <-consensus.internalMsgCCloseC:
			loop = false
		}
	}
}

func (consensus *ConsensusTBFTImpl) deleteWalEntry(num uint64, index uint64) error {

	//Block height is begin from zero,Delete the block data every 10 blocks. If the block height is 10,
	//there are 11 blocks in total and delete the consensus state data of the first 10 blocks
	i := num % 10
	if i != 0 {
		return nil
	}

	err := consensus.wal.TruncateFront(index)
	if err != nil {
		return err
	}

	consensus.logger.Infof("deleteWalEntry success! walLastIndex:%d consensus.height:%d",
		index, consensus.Height)

	return nil
}
