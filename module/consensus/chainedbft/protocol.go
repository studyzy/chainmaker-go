/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"bytes"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker-go/common/msgbus"
	timeservice "chainmaker.org/chainmaker-go/consensus/chainedbft/time_service"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/governance"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	"chainmaker.org/chainmaker-go/protocol"
	chainUtils "chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
)

var (
	InvalidPeerErr        = errors.New("invalid peer")
	ValidateSignErr       = errors.New("validate sign error")
	VerifySignerFailedErr = errors.New("verify signer failed")
)

//processNewHeight If the local node is one of the validators in current epoch, update SMR state to ConsStateType_NewLevel
//and prepare to generate a new block if local node is proposer in the current level
func (cbi *ConsensusChainedBftImpl) processNewHeight(height uint64, level uint64) {
	if cbi.smr.state != chainedbftpb.ConsStateType_NewHeight {
		cbi.logger.Debugf("service selfIndexInEpoch [%v] processNewHeight: "+
			"height [%v] level [%v] state %v", cbi.selfIndexInEpoch, height, level, cbi.smr.state.String())
		return
	}
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processNewHeight at "+
		"height [%v] level [%v],  state %v epoch %v", cbi.selfIndexInEpoch, height, level, cbi.smr.state, cbi.smr.getEpochId())
	if !cbi.smr.isValidIdx(cbi.selfIndexInEpoch) {
		cbi.logger.Infof("self selfIndexInEpoch [%v] is not in current consensus epoch", cbi.selfIndexInEpoch)
		return
	}
	cbi.smr.updateState(chainedbftpb.ConsStateType_NewLevel)
	cbi.processNewLevel(height, level)
}

//processNewLevel update state to ConsStateType_Propose and prepare to generate a new block if local node is proposer in the current level
func (cbi *ConsensusChainedBftImpl) processNewLevel(height uint64, level uint64) {
	if cbi.smr.getHeight() != height || cbi.smr.getCurrentLevel() > level ||
		cbi.smr.state != chainedbftpb.ConsStateType_NewLevel {
		cbi.logger.Debugf("service selfIndexInEpoch [%v] processNewLevel: "+
			"invalid input [%v:%v], smr height [%v] level [%v] state %v", cbi.selfIndexInEpoch,
			height, level, cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), cbi.smr.state)
		return
	}

	cbi.logger.Debugf("service selfIndexInEpoch [%v] processNewLevel at height [%v] level [%v], smr height [%v] "+
		"level [%v] state %v", cbi.selfIndexInEpoch, height, level, cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), cbi.smr.state)
	hqcBlock := cbi.chainStore.getCurrentCertifiedBlock()
	hqcLevel, err := utils.GetLevelFromBlock(hqcBlock)
	if err != nil {
		cbi.logger.Errorf("get level from block failed, error %v, height [%v]", err, hqcBlock.Header.BlockHeight)
		return
	}
	if hqcLevel >= level {
		cbi.logger.Errorf("given level [%v] too low than certified %v", level, hqcLevel)
		return
	}
}

func (cbi *ConsensusChainedBftImpl) processNewPropose(height, level uint64, preBlkHash []byte) {
	cbi.logger.Debugf("begin processNewPropose block:[%d:%d], preHash:%x, "+
		"nodeStatus: %s", height, level, preBlkHash, cbi.smr.state.String())
	if cbi.smr.state != chainedbftpb.ConsStateType_Propose {
		return
	}
	nextProposerIndex := cbi.getProposer(level)
	if cbi.isValidProposer(level, cbi.selfIndexInEpoch) {
		event := &timeservice.TimerEvent{
			Level:      level,
			Height:     height,
			State:      cbi.smr.state,
			Index:      cbi.selfIndexInEpoch,
			EpochId:    cbi.smr.getEpochId(),
			PreBlkHash: preBlkHash,
			Duration:   timeservice.GetEventTimeout(timeservice.PROPOSAL_BLOCK_TIMEOUT, 0),
		}
		cbi.addTimerEvent(event)
		cbi.logger.Infof("service selfIndexInEpoch [%v], build proposal, height: [%v], level [%v]", cbi.selfIndexInEpoch, height, level)
		cbi.msgbus.Publish(msgbus.BuildProposal, &chainedbftpb.BuildProposal{
			Height:     height,
			IsProposer: true,
			PreHash:    preBlkHash,
		})
	}
	cbi.logger.Infof("service selfIndexInEpoch [%v], waiting proposal, "+
		"height: [%v], level [%v], nextProposerIndex [%d]", cbi.selfIndexInEpoch, height, level, nextProposerIndex)
}

//processProposedBlock receive proposed block form core module, then go to new level
func (cbi *ConsensusChainedBftImpl) processProposedBlock(block *common.Block) {
	cbi.mtx.Lock()
	defer cbi.mtx.Unlock()

	height := cbi.smr.getHeight()
	level := cbi.smr.getCurrentLevel()
	cbi.logger.Debugf(`processProposedBlock start, block height: [%v], level: [%v]`, block.Header.BlockHeight, level)
	if !cbi.isValidProposer(level, cbi.selfIndexInEpoch) {
		return
	}
	if int64(height) != block.Header.BlockHeight {
		cbi.logger.Warnf(`service id [%v] selfIndexInEpoch [%v] receive proposed block height [%v]
		 not equal to smr.height [%v]`, cbi.id, cbi.selfIndexInEpoch, block.Header.BlockHeight, height)
		return
	}

	beginConstruct := chainUtils.CurrentTimeMillisSeconds()
	proposal := cbi.constructProposal(block, height, level, cbi.smr.getEpochId())
	endConstruct := chainUtils.CurrentTimeMillisSeconds()
	cbi.signAndBroadcast(proposal)
	endSignAndBroad := chainUtils.CurrentTimeMillisSeconds()
	cbi.logger.Debugf("time cost in processProposedBlock, constructProposalTime: %d, "+
		"signAndBroadTime: %d", endConstruct-beginConstruct, endSignAndBroad-endConstruct)
}

func (cbi *ConsensusChainedBftImpl) processLocalTimeout(height uint64, level uint64) {
	if !cbi.smr.processLocalTimeout(level) {
		return
	}
	var vote *chainedbftpb.ConsensusPayload
	if lastVotedLevel, lastVote := cbi.smr.getLastVote(); lastVotedLevel == level {
		// retry send last vote
		vote = cbi.retryVote(lastVote)
	} else {
		vote = cbi.constructVote(height, level, cbi.smr.getEpochId(), nil)
	}
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processLocalTimeout: broadcasts timeout "+
		"vote [%v:%v] to other validators", cbi.selfIndexInEpoch, height, level)
	cbi.smr.setLastVote(vote, level)
	cbi.signAndBroadcast(vote)
}

func (cbi *ConsensusChainedBftImpl) retryVote(lastVote *chainedbftpb.ConsensusPayload) *chainedbftpb.ConsensusPayload {
	var (
		err      error
		data     []byte
		sign     []byte
		voteMsg  = lastVote.GetVoteMsg()
		voteData = voteMsg.VoteData
	)
	cbi.logger.Debugf("service index [%v] processLocalTimeout: "+
		"get last vote [%v:%v] to other validators, blockId [%x]", cbi.selfIndexInEpoch, voteData.Height, voteData.Level, voteData.BlockID)
	// when a node timeouts at the same consensus level, it needs to change the vote type for the current level to a timeout.
	tempVoteData := &chainedbftpb.VoteData{
		NewView:   true,
		Level:     voteData.Level,
		Author:    voteData.Author,
		Height:    voteData.Height,
		BlockID:   voteData.BlockID,
		EpochId:   cbi.smr.getEpochId(),
		AuthorIdx: voteData.AuthorIdx,
	}
	if data, err = proto.Marshal(tempVoteData); err != nil {
		cbi.logger.Errorf("marshal vote failed: %s", err)
		return nil
	}
	if sign, err = cbi.singer.Sign(cbi.chainConf.ChainConfig().Crypto.Hash, data); err != nil {
		cbi.logger.Errorf("failed to sign data failed, err %v data %v", err, data)
		return nil
	}
	serializeMember, err := cbi.singer.GetSerializedMember(true)
	if err != nil {
		cbi.logger.Errorf("failed to get signer serializeMember failed, err %v", err)
		return nil
	}

	tempVoteData.Signature = &common.EndorsementEntry{
		Signer:    serializeMember,
		Signature: sign,
	}
	return &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_VoteMessage,
		Data: &chainedbftpb.ConsensusPayload_VoteMsg{&chainedbftpb.VoteMsg{
			VoteData: tempVoteData,
			SyncInfo: voteMsg.SyncInfo,
		}},
	}
}

func (cbi *ConsensusChainedBftImpl) verifyJustifyQC(qc *chainedbftpb.QuorumCert) error {
	if !qc.NewView && qc.BlockID == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validate qc failed, nil block id", cbi.selfIndexInEpoch)
		return fmt.Errorf(fmt.Sprintf("nil block id in qc"))
	}

	if qc.NewView && qc.BlockID != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validate qc failed, invalid block id", cbi.selfIndexInEpoch)
		return fmt.Errorf(fmt.Sprintf("invalid block id in qc"))
	}
	if cbi.smr.getEpochId() == qc.EpochId+1 {
		return nil
	}
	if qc.EpochId != cbi.smr.getEpochId() && (cbi.smr.getEpochId() != qc.EpochId+1) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validate qc failed, invalid "+
			"epoch id [%v],need [%v]", cbi.selfIndexInEpoch, qc.EpochId, cbi.smr.getEpochId())
		return fmt.Errorf(fmt.Sprintf("invalid epoch id in qc"))
	}

	newViewNum, votedBlockNum, err := cbi.countNumFromVotes(qc)
	if err != nil {
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

func (cbi *ConsensusChainedBftImpl) needFetch(syncInfo *chainedbftpb.SyncInfo) (bool, error) {
	var (
		err       error
		rootLevel uint64
		qc        = syncInfo.HighestQC
	)
	if rootLevel, err = cbi.chainStore.getRootLevel(); err != nil {
		return false, fmt.Errorf("get root level fail")
	}
	if qc.Level < rootLevel {
		cbi.logger.Debugf("service selfIndexInEpoch [%v] needFetch: syncinfo has an older qc [%v:%v] than root level [%v]",
			cbi.selfIndexInEpoch, qc.Height, qc.Level, rootLevel)
		return false, fmt.Errorf("sync info has a highest quorum certificate with level older than root level")
	}
	hasQc, _ := cbi.chainStore.getQC(string(qc.BlockID), qc.Height)
	hasBlk, _ := cbi.chainStore.getBlock(string(qc.BlockID), qc.Height)
	if hasQc != nil && hasBlk != nil {
		cbi.logger.Debugf("service selfIndexInEpoch [%v] needFetch: local already has a qc and block [%v:%v %x]",
			cbi.selfIndexInEpoch, qc.Height, qc.Level, qc.BlockID)
		return false, nil
	}
	return true, nil

	//var (
	//	currentQC      = cbi.chainStore.getCurrentQC()
	//	currentTCLevel = cbi.smr.getHighestTCLevel()
	//	level          = currentQC.Level
	//)
	//if currentQC.Level < currentTCLevel {
	//	level = currentTCLevel
	//}
	//if qc.Level >= level+3 {
	//	cbi.logger.Debugf("service selfIndexInEpoch [%v] needFetch: local received sync info from a future level [%v], local qc [%v] tc [%v]",
	//		cbi.selfIndexInEpoch, qc.Level, currentQC.Level, currentTCLevel)
	//	return false, fmt.Errorf("received sync info from a future level")
	//}
}

func (cbi *ConsensusChainedBftImpl) validateProposalMsg(msg *chainedbftpb.ConsensusMsg) error {
	proposal := msg.Payload.GetProposalMsg().ProposalData
	if proposal.Level < cbi.smr.getCurrentLevel() {
		return fmt.Errorf("old proposal, ignore it")
	}
	if proposal.EpochId != cbi.smr.getEpochId() {
		return fmt.Errorf("err epochId, ignore it")
	}
	if hasMsg := cbi.msgPool.GetProposal(proposal.Height, proposal.Level); hasMsg != nil {
		return fmt.Errorf("specify consensus"+
			" rounds that already have proposals, has proposal: %s", hasMsg.Payload.String())
	}

	if !cbi.validateProposer(msg) {
		return fmt.Errorf("invalid proposer")
	}

	if err := cbi.verifyJustifyQC(proposal.JustifyQC); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposal: block [%v:%v] "+
			"verifyJustifyQC failed, err %v", cbi.selfIndexInEpoch, proposal.Height, proposal.Level, err)
		return fmt.Errorf("failed to verify JustifyQC")
	}
	if !bytes.Equal(proposal.JustifyQC.BlockID, proposal.Block.GetHeader().PreBlockHash) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposal: mismatch pre hash [%x] in block, justifyQC %x",
			cbi.selfIndexInEpoch, proposal.Block.GetHeader().PreBlockHash, proposal.JustifyQC.BlockID)
		return fmt.Errorf("mismatch pre hash in block header and justify qc")
	}
	return nil
}

func (cbi *ConsensusChainedBftImpl) validateProposer(msg *chainedbftpb.ConsensusMsg) bool {
	proposal := msg.Payload.GetProposalMsg().ProposalData
	if !cbi.smr.isValidIdx(proposal.ProposerIdx) || !cbi.isValidProposer(proposal.Level, proposal.ProposerIdx) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposal: received a proposal "+
			"at height [%v] level [%v] from invalid selfIndexInEpoch [%v] addr [%v]",
			cbi.selfIndexInEpoch, proposal.Height, proposal.Level, proposal.ProposerIdx, proposal.Proposer)
		return false
	}
	if err := cbi.validateSignerAndSignature(msg, cbi.smr.getPeerByIndex(proposal.ProposerIdx)); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposer failed, err %v"+
			" proposal %v, err %v", cbi.selfIndexInEpoch, proposal, err)
		return false
	}
	return true
}

func (cbi *ConsensusChainedBftImpl) processProposal(msg *chainedbftpb.ConsensusMsg) {
	cbi.mtx.Lock()
	defer cbi.mtx.Unlock()

	var (
		proposalMsg = msg.Payload.GetProposalMsg()
		proposal    = proposalMsg.ProposalData
	)

	cbi.logger.Infof("service selfIndexInEpoch [%v] processProposal step0. proposal.ProposerIdx [%v] ,proposal.Height[%v],"+
		" proposal.Level[%v],proposal.EpochId [%v],expected [%v:%v:%v]", cbi.selfIndexInEpoch, proposal.ProposerIdx, proposal.Height,
		proposal.Level, proposal.EpochId, cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), cbi.smr.getEpochId())
	//step0: validate proposal
	if err := cbi.validateProposalMsg(msg); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal validate proposal failed, err %v",
			cbi.selfIndexInEpoch, err)
		return
	}
	cbi.logger.Debugf("validate proposal msg success [%d:%d:%d]", proposal.ProposerIdx, proposal.Height, proposal.Level)

	//step1: fetch data
	if err := cbi.fetchDataIfRequire(proposalMsg); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal fetch data failed, err %v",
			cbi.selfIndexInEpoch, err)
		return
	}

	//step2: validate and process new qc from proposal
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal step1 process qc start", cbi.selfIndexInEpoch)
	if ok := cbi.processQC(msg); !ok {
		return
	}

	//step3: validate new block from proposal
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal step2 validate new block from proposal start", cbi.selfIndexInEpoch)
	if ok := cbi.validateBlock(proposal); !ok {
		return
	}

	//step4: validate consensus args
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal step3 validate consensus args", cbi.selfIndexInEpoch)
	cbi.validateConsensusArg(proposal)

	//step5: add proposal to msg pool and add block to chainStore
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal step4 add proposal to msg pool and "+
		"add proposal block to chainStore start", cbi.selfIndexInEpoch)
	if inserted, err := cbi.msgPool.InsertProposal(proposal.Height, proposal.Level, msg); err != nil || !inserted {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal insert "+
			"a proposal failed, err %v, insert %v", cbi.selfIndexInEpoch, err, inserted)
		return
	}
	if executorErr := cbi.chainStore.insertBlock(proposal.GetBlock()); executorErr != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal add proposal block %v to chainStore failed, err: %s",
			cbi.selfIndexInEpoch, proposal.GetBlock().GetHeader().BlockHeight, executorErr)
		return
	}

	//step6: vote it and send vote to next proposer in the epoch
	// 当提案level小于节点的最新投票level时，表示当前节点已经在前述level上进行过投票，可能为赞成票或者超时票。
	if lastVoteLevel, _ := cbi.smr.getLastVote(); lastVoteLevel < proposal.Level {
		cbi.generateVoteAndSend(proposal)
	}
}

func (cbi *ConsensusChainedBftImpl) fetchDataIfRequire(proposalMsg *chainedbftpb.ProposalMsg) error {
	var (
		err     error
		isFetch bool
	)
	if isFetch, err = cbi.needFetch(proposalMsg.SyncInfo); err != nil {
		return err
	}
	if isFetch {
		cbi.fetchData(proposalMsg.ProposalData)
	}
	return nil
}

func (cbi *ConsensusChainedBftImpl) generateVoteAndSend(proposal *chainedbftpb.ProposalData) bool {
	cbi.smr.updateState(chainedbftpb.ConsStateType_Vote)
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal step5 construct vote and "+
		"send vote to next proposer start", cbi.selfIndexInEpoch)
	level, err := utils.GetLevelFromBlock(proposal.GetBlock())
	if err != nil {
		cbi.logger.Errorf("GetLevelFromBlock failed, reason: %s, block: %d:%x", err, proposal.Height, proposal.GetBlock().Header.BlockHash)
		return false
	}

	vote := cbi.constructVote(uint64(proposal.GetBlock().GetHeader().GetBlockHeight()), level, cbi.smr.getEpochId(), proposal.GetBlock())
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal step6 send vote msg to next leader start", cbi.selfIndexInEpoch)
	cbi.sendVote2Next(proposal, vote)
	return true
}

func (cbi *ConsensusChainedBftImpl) fetchData(proposal *chainedbftpb.ProposalData) {
	cbi.logger.Infof("service selfIndexInEpoch [%v] validateProposal need sync up to [%v:%v]",
		cbi.selfIndexInEpoch, proposal.JustifyQC.Height, proposal.JustifyQC.Level)

	//fetch block and qc from proposer
	req := &blockSyncReq{
		targetPeer:  proposal.ProposerIdx,
		blockID:     proposal.JustifyQC.BlockID,
		height:      proposal.JustifyQC.Height,
		startLevel:  cbi.chainStore.getCurrentQC().Level + 1,
		targetLevel: proposal.JustifyQC.Level,
	}

	//note: WaitGroup is used here to provide the blocking function, waiting for SyncManager to synchronize the data,
	//and when SyncManager does not synchronize to the block in the timeout, WaitGroup.Done is called to resolve the blocking
	cbi.syncer.blockSyncReqC <- req
	<-cbi.syncer.reqDone
	cbi.logger.Infof("service selfIndexInEpoch [%v] onReceivedProposal finish sync startlevel [%v] targetlevel [%v]",
		cbi.selfIndexInEpoch, req.startLevel, req.targetLevel)
}

// processQC insert qc and process qc from proposal msg
func (cbi *ConsensusChainedBftImpl) processQC(msg *chainedbftpb.ConsensusMsg) bool {
	proposal := msg.Payload.GetProposalMsg().ProposalData
	syncInfo := msg.Payload.GetProposalMsg().SyncInfo
	cbi.logger.Debugf("processQC start. height: [%d], level: [%d], blockHash: [%x], "+
		"JustifyQC.NewView: [%v]", proposal.Height, proposal.Level, proposal.Block.Header.BlockHash, proposal.JustifyQC.NewView)
	if !cbi.smr.voteRules(proposal.Level, proposal.JustifyQC) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposal block [%v:%v] pass "+
			"safety rules check failed", cbi.selfIndexInEpoch, proposal.Height, proposal.Level)
		return false
	}
	if !proposal.JustifyQC.NewView {
		if err := cbi.chainStore.insertQC(proposal.JustifyQC); err != nil {
			cbi.logger.Errorf("insert qc to chainStore failed: %s, qc info: %s", err, proposal.JustifyQC.String())
			return false
		}
	}

	//local already handle it when aggregating qc
	cbi.smr.updateLockedQC(proposal.JustifyQC)
	cbi.commitBlocksByQC(proposal.JustifyQC)
	cbi.processCertificates(proposal.JustifyQC, syncInfo.HighestTC)
	if proposal.Level != cbi.smr.getCurrentLevel() {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal proposal [%v:%v] does not match the "+
			"smr level [%v:%v], ignore proposal", cbi.selfIndexInEpoch, proposal.Height, proposal.Level,
			cbi.smr.getHeight(), cbi.smr.getCurrentLevel())
		return false
	}
	return true
}

func (cbi *ConsensusChainedBftImpl) validateBlock(proposal *chainedbftpb.ProposalData) bool {
	var (
		err      error
		preBlock *common.Block
	)
	if preBlock, err = cbi.chainStore.getBlock(string(
		proposal.Block.Header.PreBlockHash), proposal.Height-1); err != nil || preBlock == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposal failed to get preBlock [%v], err %v",
			cbi.selfIndexInEpoch, proposal.Height-1, err)
		return false
	}
	if !bytes.Equal(preBlock.Header.BlockHash, proposal.JustifyQC.BlockID) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateProposal failed, qc'block not equal to block's preHash",
			cbi.selfIndexInEpoch)
		return false
	}

	if err = cbi.blockVerifier.VerifyBlock(proposal.Block, protocol.CONSENSUS_VERIFY); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal invalid proposal block "+
			"from proposer %v at height [%v] level [%v], err %v", cbi.selfIndexInEpoch,
			proposal.ProposerIdx, proposal.Height, proposal.Level, err)
		return false
	}
	return true
}

func (cbi *ConsensusChainedBftImpl) validateConsensusArg(proposal *chainedbftpb.ProposalData) bool {
	var (
		err           error
		txRWSet       *common.TxRWSet
		consensusArgs *consensus.BlockHeaderConsensusArgs
	)

	if consensusArgs, err = utils.GetConsensusArgsFromBlock(proposal.Block); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal: GetConsensusArgsFromBlock err from proposer %v"+
			" at height [%v] level [%v], err %v", cbi.selfIndexInEpoch, proposal.ProposerIdx, proposal.Height, proposal.Level, err)
		return false
	}
	if txRWSet, err = governance.CheckAndCreateGovernmentArgs(proposal.Block, cbi.store, cbi.proposalCache, cbi.ledgerCache); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal: CheckAndCreateGovernmentArgs err from proposer"+
			" %v at height [%v] level [%v], err %v", cbi.selfIndexInEpoch, proposal.ProposerIdx, proposal.Height, proposal.Level, err)
		return false
	}

	txRWSetBytes, _ := proto.Marshal(txRWSet)
	ConsensusDataBytes, _ := proto.Marshal(consensusArgs.ConsensusData)
	if !bytes.Equal(txRWSetBytes, ConsensusDataBytes) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processProposal: invalid Consensus Args "+
			"from proposer %v at height [%v] level [%v], proposal data:[%v] local data:[%v]", cbi.selfIndexInEpoch,
			proposal.ProposerIdx, proposal.Height, proposal.Level, txRWSet, consensusArgs.ConsensusData)
		return false
	}
	return true
}

func (cbi *ConsensusChainedBftImpl) sendVote2Next(proposal *chainedbftpb.ProposalData, vote *chainedbftpb.ConsensusPayload) {
	nextLeaderIndex := cbi.getProposer(proposal.Level + 1)

	cbi.logger.Debugf("service selfIndexInEpoch [%v] processProposal send vote to next leader [%v]",
		cbi.selfIndexInEpoch, nextLeaderIndex)

	cbi.smr.setLastVote(vote, proposal.Level)
	if nextLeaderIndex == cbi.selfIndexInEpoch {
		consensusMessage := &chainedbftpb.ConsensusMsg{Payload: vote}
		if err := utils.SignConsensusMsg(consensusMessage, cbi.chainConf.ChainConfig().Crypto.Hash, cbi.singer); err != nil {
			cbi.logger.Errorf("sign consensus message failed, err %v", err)
			return
		}
		cbi.logger.Debugf("send vote msg to self[%d], voteHeight:[%d], voteLevel:[%d], voteBlockID:[%x]", cbi.selfIndexInEpoch,
			proposal.Height, proposal.Level, proposal.Block.Header.BlockHash)
		cbi.internalMsgCh <- consensusMessage
	} else {
		cbi.logger.Debugf("send vote msg to other peer [%d], voteHeight:[%d], voteLevel:[%d], voteBlockID:[%x]", nextLeaderIndex,
			proposal.Height, proposal.Level, proposal.Block.Header.BlockHash)
		cbi.signAndSendToPeer(vote, nextLeaderIndex)
	}
}

func (cbi *ConsensusChainedBftImpl) validateVoteData(voteData *chainedbftpb.VoteData) error {
	var (
		err       error
		data      []byte
		author    = voteData.GetAuthor()
		authorIdx = voteData.GetAuthorIdx()
	)
	if author == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteData received a "+
			"vote data with nil author", cbi.selfIndexInEpoch)
		return fmt.Errorf("nil author")
	}

	if peer := cbi.smr.getPeerByIndex(authorIdx); peer == nil || peer.id != string(author) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteData received a "+
			"vote data from invalid peer,vote authorIdx [%v]", cbi.selfIndexInEpoch, authorIdx)
		return InvalidPeerErr
	}

	cbi.logger.Debugf("service selfIndexInEpoch [%v] validateVoteData, voteData %v", cbi.selfIndexInEpoch, voteData)
	sign := voteData.Signature
	voteData.Signature = nil
	defer func() {
		voteData.Signature = sign
	}()
	if data, err = proto.Marshal(voteData); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteData "+
			"marshal vote failed, data %v , err %v", cbi.selfIndexInEpoch, voteData, err)
		return fmt.Errorf("failed to marshal payload")
	}
	if err = utils.VerifyDataSign(data, sign, cbi.accessControlProvider); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteData "+
			"verify vote failed, data signature, err %v", cbi.selfIndexInEpoch, err)
		return fmt.Errorf("failed to verify voteData signature")
	}
	return nil
}

func (cbi *ConsensusChainedBftImpl) validateVoteMsg(msg *chainedbftpb.ConsensusMsg) error {
	var (
		peer      *peer
		voteMsg   = msg.Payload.GetVoteMsg()
		author    = voteMsg.VoteData.GetAuthor()
		authorIdx = voteMsg.VoteData.GetAuthorIdx()
	)
	if author == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteMsg: received a "+
			"vote msg with nil author", cbi.selfIndexInEpoch)
		return fmt.Errorf("nil author")
	}

	if peer = cbi.smr.getPeerByIndex(authorIdx); peer == nil || peer.id != string(author) {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteMsg: received a vote msg from invalid peer", cbi.selfIndexInEpoch)
		return InvalidPeerErr
	}
	if err := cbi.validateSignerAndSignature(msg, peer); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateVoteMsg failed, vote %v, err %v", cbi.selfIndexInEpoch, voteMsg, err)
		return ValidateSignErr
	}

	vote := voteMsg.VoteData
	vote.Signature.Signer = msg.SignEntry.Signer
	if err := cbi.validateVoteData(vote); err != nil {
		return fmt.Errorf("verify vote data failed, err %v", err)
	}
	return nil
}

func (cbi *ConsensusChainedBftImpl) processVote(msg *chainedbftpb.ConsensusMsg) {
	// 1. base check vote msg
	var (
		voteMsg   = msg.Payload.GetVoteMsg()
		vote      = voteMsg.VoteData
		authorIdx = vote.GetAuthorIdx()
	)
	cbi.mtx.Lock()
	defer cbi.mtx.Unlock()

	cbi.logger.Debugf("service selfIndexInEpoch [%v] processVote: authorIdx:[%v] voteBaseInfo:[%d:%d:%d], expected:[%v:%v:%v]",
		cbi.selfIndexInEpoch, authorIdx, vote.Height, vote.Level, vote.EpochId, cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), cbi.smr.getEpochId())
	if vote.Height < cbi.smr.getHeight() || vote.Level < cbi.smr.getCurrentLevel() || vote.EpochId != cbi.smr.getEpochId() {
		cbi.logger.Warnf("service selfIndexInEpoch [%v] processVote: received vote at wrong height or level or epoch")
		return
	}
	// 1. validate vote msg info
	if err := cbi.validateVoteMsg(msg); err != nil {
		return
	}

	// 2. only proposer could handle proposal’s vote
	cbi.logger.Debugf("process vote step 1 only proposer will process vote with Proposal type or all peer can process vote with NewView type")
	if !vote.NewView {
		//regular votes are sent to the leaders of the next round only.
		if nextLeaderIndex := cbi.getProposer(vote.Level + 1); nextLeaderIndex != cbi.selfIndexInEpoch {
			cbi.logger.Errorf("service selfIndexInEpoch [%v] processVote: self is not next "+
				"leader[%d] for level [%v]", cbi.selfIndexInEpoch, nextLeaderIndex, vote.Level+1)
			return
		}
	}

	var (
		err      error
		need     bool
		inserted bool
	)
	// 3. Add vote to msgPool
	cbi.logger.Debugf("process vote step 2 check whether need sync from other peer")
	if need, err = cbi.needFetch(voteMsg.SyncInfo); err != nil {
		cbi.logger.Errorf("processVote: needFetch failed, reason: %s", err)
		return
	}
	cbi.logger.Debugf("process vote step 3 inserted the vote ")
	if inserted, err = cbi.msgPool.InsertVote(vote.Height, vote.Level, msg); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] processVote: insert vote msg failed, err %v ",
			cbi.selfIndexInEpoch, err)
		return
	} else if !inserted {
		cbi.logger.Warnf("service selfIndexInEpoch [%v] processVote: inserted"+
			" vote msg from validator failed, %v", cbi.selfIndexInEpoch, authorIdx)
		return
	}

	// 4. fetch data from peers and handle qc in voteInfo
	cbi.logger.Debugf("process vote step 4 whether need fetch info: [%v] and process", need)
	if need {
		cbi.fetchAndHandleQc(authorIdx, voteMsg)
		return
	}

	// 5. generate QC if majority are voted and process the new QC if don't need sync data from peers
	cbi.logger.Debugf("process vote step 5 no need fetch info and process vote")
	cbi.processVotes(vote)
}

//fetchAndHandleQc Fetch the missing block data and the  process the received QC until the data is all fetched.
func (cbi *ConsensusChainedBftImpl) fetchAndHandleQc(authorIdx uint64, voteMsg *chainedbftpb.VoteMsg) {
	cbi.logger.Infof("service selfIndexInEpoch [%v] processVote: need sync up to [%v:%v]",
		cbi.selfIndexInEpoch, voteMsg.SyncInfo.HighestQC.Height, voteMsg.SyncInfo.HighestQC.Level)
	req := &blockSyncReq{
		targetPeer:  authorIdx,
		height:      voteMsg.SyncInfo.HighestQC.Height,
		blockID:     voteMsg.SyncInfo.HighestQC.BlockID,
		targetLevel: voteMsg.SyncInfo.HighestQC.Level,
		startLevel:  cbi.chainStore.getCurrentQC().Level + 1,
	}
	cbi.syncer.blockSyncReqC <- req
	<-cbi.syncer.reqDone
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processVote: finish sync startlevel [%v] targetlevel [%v]",
		cbi.selfIndexInEpoch, req.startLevel, req.targetLevel)
	if cbi.smr.getCurrentLevel() < req.targetLevel {
		cbi.logger.Infof("service index [%v] processVote: sync currentLevel [%v] not catch targetLevel [%v]",
			cbi.selfIndexInEpoch, cbi.smr.getCurrentLevel(), req.targetLevel)
		return
	}
	cbi.processVotes(voteMsg.VoteData)
}

//processVotes QC is generated if a majority are voted for the special Height and Level.
func (cbi *ConsensusChainedBftImpl) processVotes(vote *chainedbftpb.VoteData) {
	blockID, newView, done := cbi.msgPool.CheckVotesDone(vote.Height, vote.Level)
	if !done {
		return
	}
	//aggregate qc
	qc, err := cbi.aggregateQCAndInsert(vote.Height, vote.Level, blockID, newView)
	if err != nil {
		cbi.logger.Errorf("service index [%v] processVote: new qc aggregated for height [%v] "+
			"level [%v] blockId [%x], err=%v", cbi.selfIndexInEpoch, vote.Height, vote.Level, blockID, err)
		return
	}

	cbi.logger.Debugf("service selfIndexInEpoch [%v] processVotes: aggregated for height [%v]"+
		" level [%v], newView: %v, qcInfo: %s", cbi.selfIndexInEpoch, vote.Height, vote.Level, newView, qc.String())

	var tc *chainedbftpb.QuorumCert
	if qc.NewView {
		// If the newly generated QC type is NewView, it means that majority agree on the timeout and assign QC to TC
		tc = qc
	}
	cbi.processCertificates(qc, tc)
	if cbi.isValidProposer(cbi.smr.getCurrentLevel(), cbi.selfIndexInEpoch) {
		cbi.smr.updateState(chainedbftpb.ConsStateType_Propose)
		qcBlk := cbi.chainStore.getCurrentCertifiedBlock()
		cbi.processNewPropose(cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), qcBlk.Header.BlockHash)
	}
}

func (cbi *ConsensusChainedBftImpl) aggregateQCAndInsert(height, level uint64, blockID []byte, isNewView bool) (*chainedbftpb.QuorumCert, error) {
	votes := cbi.msgPool.GetVotes(height, level)
	qc := &chainedbftpb.QuorumCert{
		BlockID: blockID,
		Height:  height,
		Level:   level,
		Votes:   votes,
		NewView: isNewView,
		EpochId: cbi.smr.getEpochId(),
	}
	if blockID != nil {
		if err := cbi.chainStore.insertQC(qc); err != nil {
			return nil, err
		}
	}
	return qc, nil
}

// processCertificates
// qc When processing a proposalMsg or voteMsg, the tc information is contained in the incoming message;
// in other cases, the parameter is currentQC in local node.
// tc When processing a proposalMsg or voteMsg, the tc information is contained in the incoming message;
// in other cases, the parameter is nil.
func (cbi *ConsensusChainedBftImpl) processCertificates(qc *chainedbftpb.QuorumCert, tc *chainedbftpb.QuorumCert) {
	cbi.logger.Debugf("service selfIndexInEpoch [%v] processCertificates start:, height [%v], level [%v],qc.Height "+
		"[%v] qc.Level [%v], qc.epochID [%d]", cbi.selfIndexInEpoch, cbi.smr.getHeight(), cbi.smr.getCurrentLevel(), qc.Height, qc.Level, qc.EpochId)

	var (
		tcLevel        = uint64(0)
		currentQC      = cbi.chainStore.getCurrentQC()
		committedLevel = cbi.smr.getLastCommittedLevel()
	)
	if tc != nil {
		tcLevel = tc.Level
		cbi.smr.updateTC(tc)
	}
	cbi.smr.updateLockedQC(qc)
	if enterNewLevel := cbi.smr.processCertificates(qc.Height, currentQC.Level, tcLevel, committedLevel); enterNewLevel {
		cbi.smr.updateState(chainedbftpb.ConsStateType_NewHeight)
		cbi.processNewHeight(cbi.smr.getHeight(), cbi.smr.getCurrentLevel())
	}
}

func (cbi *ConsensusChainedBftImpl) commitBlocksByQC(qc *chainedbftpb.QuorumCert) {
	commit, block, level := cbi.smr.commitRules(qc)
	if !commit {
		return
	}

	cbi.logger.Debugf("service selfIndexInEpoch [%v] processCertificates: commitRules success, height [%v], level [%v],"+
		" committed level [%v]", cbi.selfIndexInEpoch, block.Header.BlockHeight, level, cbi.chainStore.getCommitLevel())
	if level > cbi.chainStore.getCommitLevel() {
		cbi.logger.Debugf("service selfIndexInEpoch [%v] processCertificates: try committing a block %v on [%x:%v]",
			cbi.selfIndexInEpoch, block.Header.BlockHash, block.Header.BlockHeight, level)
		lastCommittedBlock, lastCommitLevel, err := cbi.chainStore.commitBlock(block)
		if lastCommittedBlock != nil {
			cbi.logger.Debugf("setCommit block status")
			cbi.smr.setLastCommittedBlock(lastCommittedBlock, lastCommitLevel)
			cbi.logger.Debugf("on block sealed, blockHeight: %d", lastCommittedBlock.Header.BlockHeight)
			cbi.msgPool.OnBlockSealed(uint64(lastCommittedBlock.Header.BlockHeight))
		}
		if err != nil {
			cbi.logger.Errorf("commit block to the chain failed, reason: %s", err)
		}
	}
}

func (cbi *ConsensusChainedBftImpl) processBlockCommitted(block *common.Block) {
	cbi.mtx.Lock()
	defer cbi.mtx.Unlock()
	cbi.logger.Debugf("processBlockCommitted received has committed block, height:%d, hash:%x",
		block.Header.BlockHeight, block.Header.BlockHash)
	// 1. check base commit block info
	if int64(cbi.commitHeight) >= block.Header.BlockHeight {
		cbi.logger.Warnf("service selfIndexInEpoch [%v] block:[%d:%x] has been committed",
			cbi.selfIndexInEpoch, block.Header.BlockHeight, block.Header.BlockHash)
		return
	}
	// 2. insert committed block to chainStore
	cbi.logger.Debugf("processBlockCommitted step 1 insert complete block")
	if err := cbi.chainStore.insertCompletedBlock(block); err != nil {
		cbi.logger.Errorf("insert block[%d:%x] to chainStore failed", block.Header.BlockHeight, block.Header.BlockHash)
		return
	}
	// 3. update commit info in the consensus
	cbi.logger.Debugf("processBlockCommitted step 2 update the last committed block info")
	cbi.commitHeight = uint64(block.Header.BlockHeight)
	cbi.msgPool.OnBlockSealed(uint64(block.Header.BlockHeight))
	cbi.smr.setLastCommittedBlock(block, cbi.chainStore.getCommitLevel())
	// 4. create next epoch if meet the condition
	cbi.logger.Debugf("processBlockCommitted step 3 create epoch if meet the condition")
	cbi.createNextEpochIfRequired(cbi.commitHeight)
	// 5. check if need to switch with the epoch
	if cbi.nextEpoch == nil || (cbi.nextEpoch != nil && cbi.nextEpoch.switchHeight > cbi.commitHeight) {
		cbi.logger.Debugf("processBlockCommitted step 4 no switch epoch and process qc")
		return
	}
	// 6. switch epoch and update field state in consensus
	oldIndex := cbi.selfIndexInEpoch
	cbi.logger.Debugf("processBlockCommitted step 5 switch epoch and process qc")
	if err := cbi.switchNextEpoch(cbi.commitHeight); err != nil {
		return
	}
	if cbi.smr.isValidIdx(cbi.selfIndexInEpoch) {
		cbi.logger.Infof("service selfIndexInEpoch [%v] start processCertificates,"+
			"height [%v],level [%v]", cbi.selfIndexInEpoch, cbi.smr.getHeight(), cbi.smr.getCurrentLevel())
	} else if oldIndex != cbi.selfIndexInEpoch {
		if oldIndex == utils.InvalidIndex {
			cbi.logger.Infof("service selfIndexInEpoch [%v] got a chance to join consensus group", cbi.selfIndexInEpoch)
		} else {
			cbi.logger.Infof("service old selfIndexInEpoch [%v] next selfIndexInEpoch [%v] leave consensus group",
				oldIndex, cbi.selfIndexInEpoch)
		}
	}
	//7. process qc: commitHeight+1
	// todo. may be delete later.
	//cbi.processCertificates(cbi.chainStore.getCurrentQC(), nil)
	cbi.logger.Infof("processBlockCommitted end, block: [%d:%x].", cbi.commitHeight, block.Header.BlockHash)
}

func (cbi *ConsensusChainedBftImpl) switchNextEpoch(blockHeight uint64) error {
	cbi.logger.Debugf("service [%v] handle block committed: "+
		"start switching to next epoch at height [%v]", cbi.selfIndexInEpoch, blockHeight)
	chainStore, err := openChainStore(cbi.ledgerCache, cbi.blockCommitter, cbi.store, cbi, cbi.logger)
	if err != nil {
		cbi.logger.Errorf("new consensus service failed, err %v", err)
		return err
	}

	if cbi.timerService != nil {
		cbi.timerService.Stop()
	}
	if cbi.syncer != nil {
		cbi.syncer.stop()
	}
	cbi.chainStore = chainStore
	cbi.syncer = newSyncManager(cbi)
	cbi.msgPool = cbi.nextEpoch.msgPool
	cbi.timerService = timeservice.NewTimerService()
	cbi.selfIndexInEpoch = cbi.nextEpoch.index
	cbi.smr = newChainedBftSMR(cbi.chainID, cbi.nextEpoch, cbi.chainStore, cbi.timerService)
	cbi.nextEpoch = nil
	go cbi.timerService.Start()
	go cbi.syncer.start()
	cbi.helper.DiscardAboveHeight(int64(blockHeight))
	return nil
}

func (cbi *ConsensusChainedBftImpl) validateBlockFetch(msg *chainedbftpb.ConsensusMsg) error {
	req := msg.Payload.GetBlockFetchMsg()
	authorIdx := req.GetAuthorIdx()
	peer := cbi.smr.getPeerByIndex(authorIdx)
	if peer == nil {
		return fmt.Errorf("validateBlockFetch: received a vote msg from invalid peer: %d", authorIdx)
	}
	if err := cbi.validateSignerAndSignature(msg, peer); err != nil {
		return fmt.Errorf("validateBlockFetch verify signature failed, err %v", err)
	}
	//if req.NumBlocks > MaxSyncBlockNum {
	//	return fmt.Errorf("validateBlockFetch: fetch too many blocks %v", req.NumBlocks)
	//}
	return nil
}

func (cbi *ConsensusChainedBftImpl) processBlockFetch(msg *chainedbftpb.ConsensusMsg) {
	var (
		req    = msg.Payload.GetBlockFetchMsg()
		blocks = make([]*chainedbftpb.BlockPair, 0, req.NumBlocks)

		id        = string(req.BlockID)
		height    = req.Height
		status    = chainedbftpb.BlockFetchStatus_Succeeded
		authorIdx = req.GetAuthorIdx()
	)

	cbi.logger.Debugf("processBlockFetch receive req msg:%s", req.String())
	if err := cbi.validateBlockFetch(msg); err != nil {
		cbi.logger.Errorf("processBlockFetch verify msg failed: %s", err)
		return
	}

	for i := 0; i < int(req.NumBlocks); i++ {
		block, _ := cbi.chainStore.getBlock(id, height)
		qc, _ := cbi.chainStore.getQC(id, height)
		if block == nil || qc == nil {
			cbi.logger.Debugf("not found block:[%v] or qc info:[%v] in [%d:%x]", block, qc)
			status = chainedbftpb.BlockFetchStatus_NotEnoughBlocks
			break
		}
		//clone for marshall
		newBlock := proto.Clone(block).(*common.Block)
		newQc := proto.Clone(qc).(*chainedbftpb.QuorumCert)
		blockPair := &chainedbftpb.BlockPair{
			Block: newBlock,
			QC:    newQc,
		}
		height = height - 1
		id = string(newBlock.Header.PreBlockHash)
		blocks = append(blocks, blockPair)
	}
	if len(blocks) == 0 {
		return
	}
	cbi.logger.Debugf("response blocks num: %d", len(blocks))
	count := len(blocks) / MaxSyncBlockNum
	if len(blocks)%MaxSyncBlockNum > 0 {
		count++
	}
	for i := 0; i <= count-1; i++ {
		if i == count-1 {
			rsp := cbi.constructBlockFetchRespMsg(blocks[i*MaxSyncBlockNum:], status)
			cbi.signAndSendToPeer(rsp, authorIdx)
		} else {
			rsp := cbi.constructBlockFetchRespMsg(blocks[i*MaxSyncBlockNum:(i+1)*MaxSyncBlockNum], status)
			cbi.signAndSendToPeer(rsp, authorIdx)
		}
	}
}

func (cbi *ConsensusChainedBftImpl) validateBlockFetchRsp(msg *chainedbftpb.ConsensusMsg) error {
	rsp := msg.Payload.GetBlockFetchRespMsg()
	authorIdx := rsp.GetAuthorIdx()
	peer := cbi.smr.getPeerByIndex(authorIdx)
	if peer == nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateBlockFetchRsp: received a vote msg from invalid peer", cbi.selfIndexInEpoch)
		return InvalidPeerErr
	}

	if err := cbi.validateSignerAndSignature(msg, peer); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] from %v validateBlockFetchRsp failed, err %v"+
			" fetch rsp %v, err %v", cbi.selfIndexInEpoch, rsp.AuthorIdx, rsp, err)
		return ValidateSignErr
	}
	cbi.logger.Infof("service selfIndexInEpoch [%v] from %v validateBlockFetchRsp success,"+
		" fetch rsp %v ", cbi.selfIndexInEpoch, rsp.AuthorIdx, rsp)
	return nil
}

func (cbi *ConsensusChainedBftImpl) addTimerEvent(event *timeservice.TimerEvent) {
	cbi.timerService.AddEvent(event)
}

//validateSignerAndSignature validate msg signer and signatures
func (cbi *ConsensusChainedBftImpl) validateSignerAndSignature(msg *chainedbftpb.ConsensusMsg, peer *peer) error {
	// check cert id
	// if msg generated by self, check signer, because netService not mapping self nodeId
	if peer.index == cbi.selfIndexInEpoch {
		member, err := cbi.accessControlProvider.NewMemberFromProto(msg.SignEntry.Signer)
		if err != nil {
			cbi.logger.Errorf("service selfIndexInEpoch [%v] validateSignerAndSignature failed,: new member by msg.SignEntry.Signer err %v",
				cbi.selfIndexInEpoch, err)
			return VerifySignerFailedErr
		}
		if cbi.singer.GetOrgId() != member.GetOrgId() {
			cbi.logger.Errorf("service selfIndexInEpoch [%v] validateSignerAndSignature failed,: match msg.signer and msg.payload.selfIndexInEpoch signer",
				cbi.selfIndexInEpoch)
			return VerifySignerFailedErr
		}
	} else { // otherwise, check nodeId by signer.cert
		nodeId, err := utils.GetUidFromProtoSigner(msg.SignEntry.Signer, cbi.netService, cbi.accessControlProvider)
		if err != nil {
			cbi.logger.Errorf("service selfIndexInEpoch [%v] validateSignerAndSignature failed,: get nodeId by msg.SignEntry.Signer err %v",
				cbi.selfIndexInEpoch, err)
			return VerifySignerFailedErr
		}
		if nodeId != peer.id {
			cbi.logger.Errorf("service selfIndexInEpoch [%v] validateSignerAndSignature failed,: match msg.signer and msg.payload.selfIndexInEpoch signer "+
				"nodeId %v, payload nodeId %v", cbi.selfIndexInEpoch, nodeId, peer.id)
			return VerifySignerFailedErr
		}
	}
	//check sign
	if err := utils.VerifyConsensusMsgSign(msg, cbi.accessControlProvider); err != nil {
		cbi.logger.Errorf("service selfIndexInEpoch [%v] validateSignerAndSignature failed,: verify "+
			" msg, err %v", cbi.selfIndexInEpoch, err)
		return fmt.Errorf("verify signature failed")
	}
	return nil
}
