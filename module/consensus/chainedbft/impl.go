/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/chainconf"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/message"
	timeservice "chainmaker.org/chainmaker-go/consensus/chainedbft/time_service"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/types"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/governance"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	"chainmaker.org/chainmaker-go/pb/protogo/net"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/gogo/protobuf/proto"
)

const (
	CONSENSUSCAPABILITY = 100000
	INTERNALCAPABILITY  = 100000
	ModuleName          = "chainedbft"
)

// ConsensusChainedBftImpl implements chained hotstuff consensus protocol
type ConsensusChainedBftImpl struct {
	id               string // The identity of the local node
	chainID          string // chain ID
	selfIndexInEpoch uint64 // Index of the local node in the validator collection of the current epoch

	msgCh           chan *net.NetMsg                // Receive information from the msgBus
	consBlockCh     chan *common.Block              // Transmit the committed block information
	proposedBlockCh chan *common.Block              // Transmit the block information generated by the local node
	syncMsgCh       chan *chainedbftpb.ConsensusMsg // Transmit request and response information with the block
	internalMsgCh   chan *chainedbftpb.ConsensusMsg // Transmit the own proposals, voting information by the local node
	protocolMsgCh   chan *chainedbftpb.ConsensusMsg // Transmit Hotstuff protocol information: proposal, vote

	mtx                sync.RWMutex
	nextEpoch          *epochManager       // next epoch
	commitHeight       uint64              // The height of the latest committed block
	governanceContract protocol.Government // The management contract on the block chain

	// Services within the module
	smr          *chainedbftSMR            // State machine replication in hotstuff
	syncer       *syncManager              // The information synchronization of the consensus module
	msgPool      *message.MsgPool          // manages all of consensus messages received for protocol
	chainStore   *chainStore               // Cache blocks, status information of QC, and the process of the commit blocks on the chain
	timerService *timeservice.TimerService // Timer service

	// Services of other modules
	logger                *logger.CMLogger
	msgbus                msgbus.MessageBus
	singer                protocol.SigningMember
	helper                protocol.HotStuffHelper
	store                 protocol.BlockchainStore
	chainConf             protocol.ChainConf
	netService            protocol.NetService
	ledgerCache           protocol.LedgerCache
	blockVerifier         protocol.BlockVerifier
	blockCommitter        protocol.BlockCommitter
	proposalCache         protocol.ProposalCache
	accessControlProvider protocol.AccessControlProvider

	// Exit signal
	quitCh         chan struct{}
	quitSyncCh     chan struct{}
	quitProtocolCh chan struct{}
}

//New returns an instance of chainedbft consensus
func New(chainID string, id string, singer protocol.SigningMember, ac protocol.AccessControlProvider,
	ledgerCache protocol.LedgerCache, proposalCache protocol.ProposalCache, blockVerifier protocol.BlockVerifier,
	blockCommitter protocol.BlockCommitter, netService protocol.NetService, store protocol.BlockchainStore,
	msgBus msgbus.MessageBus, chainConf protocol.ChainConf, helper protocol.HotStuffHelper) (*ConsensusChainedBftImpl, error) {

	service := &ConsensusChainedBftImpl{
		id:              id,
		chainID:         chainID,
		msgCh:           make(chan *net.NetMsg, CONSENSUSCAPABILITY),
		syncMsgCh:       make(chan *chainedbftpb.ConsensusMsg, INTERNALCAPABILITY),
		internalMsgCh:   make(chan *chainedbftpb.ConsensusMsg, INTERNALCAPABILITY),
		protocolMsgCh:   make(chan *chainedbftpb.ConsensusMsg, INTERNALCAPABILITY),
		consBlockCh:     make(chan *common.Block, INTERNALCAPABILITY),
		proposedBlockCh: make(chan *common.Block, INTERNALCAPABILITY),

		store:                 store,
		singer:                singer,
		helper:                helper,
		msgbus:                msgBus,
		chainConf:             chainConf,
		netService:            netService,
		ledgerCache:           ledgerCache,
		proposalCache:         proposalCache,
		blockVerifier:         blockVerifier,
		blockCommitter:        blockCommitter,
		accessControlProvider: ac,
		logger:                logger.GetLoggerByChain(logger.MODULE_CONSENSUS, chainConf.ChainConfig().ChainId),

		timerService:       timeservice.NewTimerService(),
		governanceContract: governance.NewGovernanceContract(store),

		quitCh:         make(chan struct{}),
		quitSyncCh:     make(chan struct{}),
		quitProtocolCh: make(chan struct{}),
	}

	chainStore, err := openChainStore(service.ledgerCache, service.blockCommitter, service.store, service, service.logger)
	if err != nil {
		service.logger.Errorf("new consensus service failed, err %v", err)
		return nil, err
	}
	service.chainStore = chainStore
	service.syncer = newSyncManager(service)
	service.commitHeight = service.chainStore.getCommitHeight()
	epoch := service.createEpoch(service.commitHeight)
	service.msgPool = epoch.msgPool
	service.selfIndexInEpoch = epoch.index
	service.smr = newChainedBftSMR(chainID, epoch, chainStore, service.timerService)
	service.logger.Debugf("init epoch, epochID: %d, index: %d, createHeight: %d", epoch.epochId, epoch.index, epoch.createHeight)
	chainConf.AddWatch(service)
	if err := chainconf.RegisterVerifier(chainID, consensus.ConsensusType_HOTSTUFF, service.governanceContract); err != nil {
		return nil, err
	}
	service.initTimeOutConfig(chainConf.(*chainconf.ChainConf).ChainConfig())
	return service, nil
}

func (cbi *ConsensusChainedBftImpl) initTimeOutConfig(chainConfig *config.ChainConfig) {
	for _, kv := range chainConfig.Consensus.ExtConfig {
		switch kv.Key {
		case timeservice.ProposerTimeoutMill:
			if proposerTimeOut, err := parseInt(kv.Key, kv.Value); err == nil {
				timeservice.ProposerTimeout = time.Duration(proposerTimeOut) * time.Millisecond
			}
		case timeservice.ProposerTimeoutIntervalMill:
			if proposerTimeOutInterval, err := parseInt(kv.Key, kv.Value); err == nil {
				timeservice.ProposerTimeoutInterval = time.Duration(proposerTimeOutInterval) * time.Millisecond
			}
		case timeservice.RoundTimeoutMill:
			if roundTimeOut, err := parseInt(kv.Key, kv.Value); err == nil {
				timeservice.RoundTimeout = time.Duration(roundTimeOut) * time.Millisecond
			}
		case timeservice.RoundTimeoutIntervalMill:
			if roundTimeOutInterval, err := parseInt(kv.Key, kv.Value); err == nil {
				timeservice.RoundTimeoutInterval = time.Duration(roundTimeOutInterval) * time.Millisecond
			}
		}
	}
}

func parseInt(key, val string) (int64, error) {
	t, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	if t <= 0 {
		return 0, fmt.Errorf("invalid config[%s] value: %d <= 0", key, t)
	}
	if t > int64(math.MaxInt64)/int64(time.Millisecond) {
		return 0, fmt.Errorf("invalid config[%s] value: %d > maxInt64/time.Millisecond ", key, t)
	}
	return t, nil
}

//Start start consensus
func (cbi *ConsensusChainedBftImpl) Start() error {
	cbi.logger.Infof("consensus.chainedBft service started")
	cbi.msgbus.Register(msgbus.ProposedBlock, cbi)
	cbi.msgbus.Register(msgbus.RecvConsensusMsg, cbi)
	cbi.msgbus.Register(msgbus.BlockInfo, cbi)

	go cbi.syncer.start()
	go cbi.timerService.Start()
	go cbi.loop()
	go cbi.protocolLoop()
	go cbi.syncLoop()
	go cbi.processCertificates(cbi.chainStore.getCurrentQC(), nil)
	return nil
}

//Stop stop consensus
func (cbi *ConsensusChainedBftImpl) Stop() error {
	close(cbi.quitProtocolCh)
	close(cbi.quitSyncCh)
	close(cbi.quitCh)
	if cbi.timerService != nil {
		cbi.timerService.Stop()
	}
	if cbi.msgPool != nil {
		cbi.msgPool.Cleanup()
	}
	return nil
}

//OnMessage MsgBus implement interface, receive message from MsgBus
func (cbi *ConsensusChainedBftImpl) OnMessage(message *msgbus.Message) {
	cbi.logger.Debugf("id [%s] OnMessage receive topic: %s", cbi.id, message.Topic)
	switch message.Topic {
	case msgbus.ProposedBlock:
		if block, ok := message.Payload.(*common.Block); ok {
			cbi.proposedBlockCh <- block
		}
	case msgbus.RecvConsensusMsg:
		if netMsg, ok := message.Payload.(*net.NetMsg); ok {
			cbi.msgCh <- netMsg
		}
	case msgbus.BlockInfo:
		if blockInfo, ok := message.Payload.(*common.BlockInfo); ok {
			if blockInfo == nil || blockInfo.Block == nil {
				cbi.logger.Errorf("error message BlockInfo is nil")
				return
			}
			cbi.consBlockCh <- blockInfo.Block
		}
	}
}

func (cbi *ConsensusChainedBftImpl) loop() {
	for {
		select {
		case msg, ok := <-cbi.msgCh:
			if ok {
				cbi.onReceivedMsg(msg)
			}
		case msg, ok := <-cbi.internalMsgCh:
			if ok {
				cbi.onConsensusMsg(msg)
			}
		case msg, ok := <-cbi.proposedBlockCh:
			if ok {
				cbi.onProposedBlock(msg)
			}
		case block, ok := <-cbi.consBlockCh:
			if ok {
				cbi.onBlockCommitted(block)
			}
		case firedEvent, ok := <-cbi.timerService.GetFiredCh():
			if ok {
				cbi.onFiredEvent(firedEvent)
			}
		case <-cbi.quitCh:
			return
		}
	}
}

func (cbi *ConsensusChainedBftImpl) protocolLoop() {
	for {
		select {
		case msg, ok := <-cbi.protocolMsgCh:
			if !ok {
				continue
			}
			switch msg.Payload.Type {
			case chainedbftpb.MessageType_ProposalMessage:
				cbi.onReceivedProposal(msg)
			case chainedbftpb.MessageType_VoteMessage:
				cbi.onReceivedVote(msg)
			default:
				cbi.logger.Warnf("service selfIndexInEpoch [%v] received non-protocol msg %v", cbi.selfIndexInEpoch, msg.Payload.Type)
			}
		case <-cbi.quitSyncCh:
			return
		}
	}
}

func (cbi *ConsensusChainedBftImpl) syncLoop() {
	for {
		select {
		case msg, ok := <-cbi.syncMsgCh:
			if !ok {
				continue
			}
			switch msg.Payload.Type {
			case chainedbftpb.MessageType_BlockFetchMessage:
				cbi.onReceiveBlockFetch(msg)
			case chainedbftpb.MessageType_BlockFetchRespMessage:
				cbi.onReceiveBlockFetchRsp(msg)
			default:
				cbi.logger.Warnf("service selfIndexInEpoch [%v] received non-sync msg %v", cbi.selfIndexInEpoch, msg.Payload.Type)
			}
		case <-cbi.quitProtocolCh:
			return
		}
	}
}

//OnQuit msgbus quit
func (cbi *ConsensusChainedBftImpl) OnQuit() {
	// do nothing
}

//Module chainedBft
func (cbi *ConsensusChainedBftImpl) Module() string {
	return ModuleName
}

//Watch implement watch interface
func (cbi *ConsensusChainedBftImpl) Watch(chainConfig *config.ChainConfig) error {
	cbi.logger.Debugf("service selfIndexInEpoch [%v] watch chain config updated %v", cbi.selfIndexInEpoch)
	cbi.initTimeOutConfig(chainConfig)
	return nil
}

func (cbi *ConsensusChainedBftImpl) onReceivedMsg(msg *net.NetMsg) {
	if msg == nil {
		cbi.logger.Warnf("service selfIndexInEpoch [%v] received nil message", cbi.selfIndexInEpoch)
		return
	}
	if msg.Type != net.NetMsg_CONSENSUS_MSG {
		cbi.logger.Warnf("service selfIndexInEpoch [%v] received unsubscribed msg %v to %v",
			cbi.selfIndexInEpoch, msg.Type, msg.To)
		return
	}

	cbi.logger.Debugf("service selfIndexInEpoch [%v] received a consensus msg from remote peer "+
		"id %v addr %v", cbi.selfIndexInEpoch, msg.Type, msg.To)
	consensusMsg := new(chainedbftpb.ConsensusMsg)
	if err := proto.Unmarshal(msg.Payload, consensusMsg); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] failed to unmarshal consensus data %v, err %v",
			cbi.selfIndexInEpoch, msg.Payload, err)
		return
	}
	if consensusMsg.Payload == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] received invalid consensus msg with nil payload "+
			"from remote peer id [%v] add %v", cbi.selfIndexInEpoch, msg.Type, msg.To)
		return
	}
	if err := message.ValidateMessageBasicInfo(consensusMsg.Payload); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] failed to validate msg basic info, err %v",
			cbi.selfIndexInEpoch, err)
		return
	}
	cbi.onConsensusMsg(consensusMsg)
}

//onConsensusMsg dispatches consensus msg to handler
func (cbi *ConsensusChainedBftImpl) onConsensusMsg(msg *chainedbftpb.ConsensusMsg) {
	cbi.logger.Debugf("service selfIndexInEpoch [%v] dispatch msg %v to related channel",
		cbi.selfIndexInEpoch, msg.Payload.Type)
	switch msg.Payload.Type {
	case chainedbftpb.MessageType_ProposalMessage:
		cbi.protocolMsgCh <- msg
	case chainedbftpb.MessageType_VoteMessage:
		cbi.protocolMsgCh <- msg
	case chainedbftpb.MessageType_BlockFetchMessage:
		cbi.syncMsgCh <- msg
	case chainedbftpb.MessageType_BlockFetchRespMessage:
		cbi.syncMsgCh <- msg
	}
}

//onFiredEvent dispatches timer event to handler
func (cbi *ConsensusChainedBftImpl) onFiredEvent(te *timeservice.TimerEvent) {
	if te.Height != cbi.smr.getHeight() ||
		te.Level < cbi.smr.getCurrentLevel() || te.EpochId != cbi.smr.getEpochId() ||
		(te.Level == cbi.smr.getCurrentLevel() && te.State < cbi.smr.state) {
		cbi.logger.Debugf("service selfIndexInEpoch [%v] onFiredEvent: older event %v, smr:"+
			" height [%v], level [%v], state [%v], epoch [%v]", cbi.selfIndexInEpoch, te,
			cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), cbi.smr.state, cbi.smr.getEpochId())
		return
	}

	cbi.logger.Infof("receive time out event, state: %s, height: %d, level: %d, duration: %s", te.State.String(), te.Height, te.Level, te.Duration.String())
	switch te.State {
	case chainedbftpb.ConsStateType_NewLevel:
		cbi.processNewLevel(te.Height, te.Level)
	case chainedbftpb.ConsStateType_PaceMaker:
		cbi.processLocalTimeout(te.Height, te.Level)
	default:
		cbi.logger.Errorf("service selfIndexInEpoch [%v] received invalid event %v", cbi.selfIndexInEpoch, te)
	}
}

//onReceiveBlockFetch handles a block fetch request
func (cbi *ConsensusChainedBftImpl) onReceiveBlockFetch(msg *chainedbftpb.ConsensusMsg) {
	cbi.processBlockFetch(msg)
}

//onReceiveBlockFetchRsp handles a block fetch response
func (cbi *ConsensusChainedBftImpl) onReceiveBlockFetchRsp(msg *chainedbftpb.ConsensusMsg) {
	if err := cbi.validateBlockFetchRsp(msg); err != nil {
		return
	}
	authorIdx := msg.Payload.GetBlockFetchRespMsg().GetAuthorIdx()
	cbi.syncer.syncMsgC <- &syncMsg{
		fromPeer: authorIdx,
		msg:      msg.Payload,
	}
}

//onBlockCommitted update the consensus smr to latest
func (cbi *ConsensusChainedBftImpl) onBlockCommitted(block *common.Block) {
	cbi.processBlockCommitted(block)
}

//onProposedBlock
func (cbi *ConsensusChainedBftImpl) onProposedBlock(block *common.Block) {
	cbi.processProposedBlock(block)
}

func (cbi *ConsensusChainedBftImpl) onReceivedVote(msg *chainedbftpb.ConsensusMsg) {
	cbi.processVote(msg)
}

func (cbi *ConsensusChainedBftImpl) onReceivedProposal(msg *chainedbftpb.ConsensusMsg) {
	cbi.processProposal(msg)
}

// VerifyBlockSignatures verify consensus qc at incoming block
func (cbi *ConsensusChainedBftImpl) VerifyBlockSignatures(block *common.Block) error {
	if block == nil || block.AdditionalData == nil ||
		len(block.AdditionalData.ExtraData) <= 0 {
		return errors.New("nil block or nil additionalData or empty extraData")
	}

	var (
		err           error
		quorumCert    []byte
		newViewNum    int
		votedBlockNum int
		blockID       = block.GetHeader().GetBlockHash()
	)
	if quorumCert = utils.GetQCFromBlock(block); len(quorumCert) == 0 {
		return errors.New("qc is nil")
	}
	qc := new(chainedbftpb.QuorumCert)
	if err = proto.Unmarshal(quorumCert, qc); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] unmarshal qc failed, err %v", cbi.selfIndexInEpoch, err)
		return fmt.Errorf("unmarshal qc failed, err %v", err)
	}
	if qc.BlockID == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validate qc failed, nil block id", cbi.selfIndexInEpoch)
		return fmt.Errorf("nil block id in qc")
	}
	if !bytes.Equal(qc.BlockID, blockID) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validate qc failed, wrong qc blockID [%v],"+
			"expected [%v]", cbi.selfIndexInEpoch, qc.BlockID, blockID)
		return fmt.Errorf("wrong qc block id [%v], expected [%v]",
			qc.BlockID, blockID)
	}
	if newViewNum, votedBlockNum, err = cbi.countNumFromVotes(qc); err != nil {
		return err
	}
	if qc.Level > 0 && qc.NewView && newViewNum < cbi.smr.min() {
		return fmt.Errorf(fmt.Sprintf("vote new view num [%v] less than expected [%v]",
			newViewNum, cbi.smr.min()))
	}
	if qc.Level > 0 && !qc.NewView && votedBlockNum < cbi.smr.min() {
		return fmt.Errorf(fmt.Sprintf("vote block num [%v] less than expected [%v]",
			votedBlockNum, cbi.smr.min()))
	}
	return nil
}

func (cbi *ConsensusChainedBftImpl) countNumFromVotes(qc *chainedbftpb.QuorumCert) (int, int, error) {
	var (
		newViewNum    = 0
		votedBlockNum = 0
		voteIdxs      = make(map[uint64]bool, 0)
	)
	//for each vote
	for _, vote := range qc.Votes {
		if vote == nil {
			return 0, 0, fmt.Errorf("vote is nil")
		}
		if err := cbi.validateVoteData(vote); err != nil {
			return 0, 0, fmt.Errorf("invalid commits, err %v", err)
		}
		if vote.Height != qc.Height || vote.Level != qc.Level {
			return 0, 0, fmt.Errorf("vote for wrong height:round:level [%v:%v], expected [%v:%v]",
				vote.Height, vote.Level, qc.Height, qc.Level)
		}
		if ok := voteIdxs[vote.AuthorIdx]; ok {
			return 0, 0, fmt.Errorf("duplicate vote index [%v] at height:round:level [%v:%v]",
				vote.AuthorIdx, vote.Height, vote.Level)
		}
		voteIdxs[vote.AuthorIdx] = true
		if vote.NewView && vote.BlockID == nil {
			newViewNum++
			continue
		}
		if qc.BlockID != nil && (bytes.Compare(vote.BlockID, qc.BlockID) < 0) {
			continue
		}
		votedBlockNum++
	}
	return newViewNum, votedBlockNum, nil
}

//VerifyBlockSignatures verify consensus qc at incoming block and chainconf
//now, only implement check commit in all validator, not in selected committee
func VerifyBlockSignatures(chainConf protocol.ChainConf, ac protocol.AccessControlProvider, store protocol.BlockchainStore, block *common.Block) error {
	if block == nil || block.AdditionalData == nil ||
		len(block.AdditionalData.ExtraData) <= 0 {
		return errors.New("nil block or nil additionalData or empty extraData")
	}

	//1. get qc and validate
	quorumCert := utils.GetQCFromBlock(block)
	if quorumCert == nil {
		return errors.New("nil qc")
	}
	qc := new(chainedbftpb.QuorumCert)
	if err := proto.Unmarshal(quorumCert, qc); err != nil {
		return fmt.Errorf("failed to unmarshal qc, err %v", err)
	}
	if qc.BlockID == nil {
		return fmt.Errorf("nil block id in qc")
	}
	if blockID := block.GetHeader().GetBlockHash(); !bytes.Equal(qc.BlockID, blockID) {
		return fmt.Errorf("wrong qc block id [%v], expected [%v]", qc.BlockID, blockID)
	}

	// because the validator set has changed after the generation switch, so that validate by validators
	// cannot be continue.
	governanceContract := governance.NewGovernanceContract(store)
	if governanceContract.GetEpochId() == qc.EpochId+1 {
		return nil
	}

	//2. get validators from governance contract
	var curValidators []*types.Validator
	validatorsMembersInterface := governanceContract.GetValidators()
	if validatorsMembersInterface == nil {
		return fmt.Errorf("current validators is nil")
	}
	validatorsMembers := validatorsMembersInterface.([]*consensus.GovernanceMember)
	for _, v := range validatorsMembers {
		validator := &types.Validator{
			Index:  uint64(v.Index),
			NodeID: v.NodeID,
		}
		curValidators = append(curValidators, validator)
	}

	newViewNum, votedBlockNum, err := countNumFromVotes(qc, curValidators, ac)
	if err != nil {
		return err
	}
	minQuorumForQc := governanceContract.GetGovMembersValidatorMinCount()
	if qc.Level > 0 && qc.NewView && newViewNum < minQuorumForQc {
		return fmt.Errorf(fmt.Sprintf("vote new view num [%v] less than expected [%v]",
			newViewNum, minQuorumForQc))
	}
	if qc.Level > 0 && !qc.NewView && votedBlockNum < minQuorumForQc {
		return fmt.Errorf(fmt.Sprintf("vote block num [%v] less than expected [%v]",
			votedBlockNum, minQuorumForQc))
	}
	return nil
}

func validateVoteData(voteData *chainedbftpb.VoteData, validators []*types.Validator, ac protocol.AccessControlProvider) error {
	author := voteData.GetAuthor()
	authorIdx := voteData.GetAuthorIdx()
	if author == nil {
		return fmt.Errorf("author is nil")
	}

	// get validator by authorIdx
	var validator *types.Validator = nil
	for _, v := range validators {
		if v.Index == authorIdx {
			validator = v
			break
		}
	}
	if validator == nil {
		return fmt.Errorf("msg index not in validators")
	}
	if validator.NodeID != string(author) {
		return fmt.Errorf("msg author not equal validator nodeid")
	}

	// check cert id
	if voteData.Signature == nil || voteData.Signature.Signer == nil {
		return fmt.Errorf("signer is nil")
	}

	//check sign
	sign := voteData.Signature
	voteData.Signature = nil
	defer func() {
		voteData.Signature = sign
	}()
	data, err := proto.Marshal(voteData)
	if err != nil {
		return fmt.Errorf("marshal payload failed, err %v", err)
	}
	err = utils.VerifyDataSign(data, sign, ac)
	if err != nil {
		return fmt.Errorf("verify signature failed, err %v", err)
	}
	return nil
}

func countNumFromVotes(qc *chainedbftpb.QuorumCert, curvalidators []*types.Validator, ac protocol.AccessControlProvider) (uint64, uint64, error) {
	var newViewNum uint64 = 0
	var votedBlockNum uint64 = 0
	voteIdxes := make(map[uint64]bool, 0)
	//for each vote
	for _, vote := range qc.Votes {
		if vote == nil {
			return 0, 0, fmt.Errorf("nil Commits msg")
		}
		if err := validateVoteData(vote, curvalidators, ac); err != nil {
			return 0, 0, fmt.Errorf("invalid commits, err %v", err)
		}
		// vote := msg.Payload.GetVoteMsg()
		if vote.Height != qc.Height || vote.Level != qc.Level {
			return 0, 0, fmt.Errorf("vote for wrong height:round:level [%v:%v], expected [%v:%v]",
				vote.Height, vote.Level, qc.Height, qc.Level)
		}
		if ok := voteIdxes[vote.AuthorIdx]; ok {
			return 0, 0, fmt.Errorf("duplicate vote index [%v] at height:round:level [%v:%v]",
				vote.AuthorIdx, vote.Height, vote.Level)
		}
		voteIdxes[vote.AuthorIdx] = true
		if vote.NewView && vote.BlockID == nil {
			newViewNum++
			continue
		}

		if qc.BlockID != nil && (bytes.Compare(vote.BlockID, qc.BlockID) < 0) {
			continue
		}
		votedBlockNum++
	}
	return newViewNum, votedBlockNum, nil
}
