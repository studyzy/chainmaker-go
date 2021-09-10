/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/governance"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"github.com/gogo/protobuf/proto"
)

//constructBlock generates a block at height and level
func (cbi *ConsensusChainedBftImpl) constructBlock(block *common.Block, level uint64) *common.Block {
	if block == nil {
		cbi.logger.Debugf(`constructBlock block is nil`)
		return nil
	}
	var (
		err     error
		txRWSet *common.TxRWSet
	)
	if txRWSet, err = governance.CheckAndCreateGovernmentArgs(block, cbi.store, cbi.proposalCache,
		cbi.ledgerCache); err != nil {
		cbi.logger.Errorf(`CheckAndCreateGovernmentArgs err!`)
		return nil
	}
	if err = utils.AddConsensusArgstoBlock(block, level, txRWSet); err != nil {
		cbi.logger.Errorf(`add consensus args to block err, %v`, err)
		return nil
	}
	if err = utils.SignBlock(block, cbi.chainConf.ChainConfig().Crypto.Hash, cbi.singer); err != nil {
		cbi.logger.Errorf(`sign block err, %v`, err)
		return nil
	}
	return block
}

//generateProposal generates a proposal at height and level
func (cbi *ConsensusChainedBftImpl) constructProposal(
	block *common.Block, height uint64, level uint64, epochId uint64) *chainedbftpb.ConsensusPayload {
	toProposalBlock := cbi.constructBlock(block, level)
	if toProposalBlock == nil {
		return nil
	}
	qc := cbi.chainStore.getCurrentQC()
	proposalData := &chainedbftpb.ProposalData{
		Level:       level,
		Height:      height,
		EpochId:     epochId,
		Block:       toProposalBlock,
		Proposer:    []byte(cbi.id),
		ProposerIdx: cbi.selfIndexInEpoch,
		JustifyQc:   qc,
	}
	syncInfo := &chainedbftpb.SyncInfo{
		HighestQc:      qc,
		HighestTc:      cbi.smr.getTC(),
		HighestTcLevel: cbi.smr.getHighestTCLevel(),
	}
	proposalMsg := &chainedbftpb.ProposalMsg{
		SyncInfo:     syncInfo,
		ProposalData: proposalData,
	}
	cbi.logger.Debugf("service selfIndexInEpoch [%v] constructProposal, proposal: [%v:%v:%v],"+
		"JustifyQc: %v, HighestTc: %v",
		cbi.selfIndexInEpoch, proposalData.ProposerIdx, proposalData.Height,
		proposalData.Level, qc.String(), syncInfo.HighestTc.String())

	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_PROPOSAL_MESSAGE,
		Data: &chainedbftpb.ConsensusPayload_ProposalMsg{
			ProposalMsg: proposalMsg},
	}
	return consensusPayload
}

//constructVote builds a vote msg with given params
func (cbi *ConsensusChainedBftImpl) constructVote(height uint64, level uint64, epochId uint64,
	block *common.Block) (*chainedbftpb.ConsensusPayload, error) {
	voteData := &chainedbftpb.VoteData{
		Level:     level,
		Height:    height,
		EpochId:   epochId,
		Author:    []byte(cbi.id),
		NewView:   false,
		AuthorIdx: cbi.selfIndexInEpoch,
	}
	if block == nil {
		voteData.NewView = true
	} else {
		voteData.BlockId = block.Header.BlockHash
	}
	var (
		err  error
		data []byte
		sign []byte
	)
	if data, err = proto.Marshal(voteData); err != nil {
		return nil, err
	}
	if sign, err = cbi.singer.Sign(cbi.chainConf.ChainConfig().Crypto.Hash, data); err != nil {
		cbi.logger.Errorf("sign data failed, err %v data %v", err, data)
		return nil, err
	}

	voteData.Signature = &common.EndorsementEntry{
		Signer:    nil,
		Signature: sign,
	}
	syncInfo := &chainedbftpb.SyncInfo{
		HighestTc:      cbi.smr.getTC(),
		HighestQc:      cbi.chainStore.getCurrentQC(),
		HighestTcLevel: cbi.smr.getHighestTCLevel(),
	}
	vote := &chainedbftpb.VoteMsg{
		VoteData: voteData,
		SyncInfo: syncInfo,
	}
	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_VOTE_MESSAGE,
		Data: &chainedbftpb.ConsensusPayload_VoteMsg{
			VoteMsg: vote},
	}
	return consensusPayload, nil
}

//constructBlockFetchMsg builds a block fetch request msg at given height
func (cbi *ConsensusChainedBftImpl) constructBlockFetchMsg(reqID uint64, endBlockId []byte,
	endHeight uint64, num uint64, commitBlock, lockedBlock []byte) *chainedbftpb.ConsensusPayload {
	msg := &chainedbft.BlockFetchMsg{
		ReqId:       reqID,
		Height:      endHeight,
		BlockId:     endBlockId,
		NumBlocks:   num,
		AuthorIdx:   cbi.selfIndexInEpoch,
		CommitBlock: commitBlock,
		LockedBLock: lockedBlock,
	}
	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_BLOCK_FETCH_MESSAGE,
		Data: &chainedbftpb.ConsensusPayload_BlockFetchMsg{
			BlockFetchMsg: msg},
	}
	return consensusPayload
}

//constructBlockFetchRespMsg builds a blßock fetch response with given params
func (cbi *ConsensusChainedBftImpl) constructBlockFetchRespMsg(blocks []*chainedbft.BlockPair,
	status chainedbft.BlockFetchStatus, respId uint64) *chainedbftpb.ConsensusPayload {
	msg := &chainedbft.BlockFetchRespMsg{
		RespId:    respId,
		Status:    status,
		Blocks:    blocks,
		AuthorIdx: cbi.selfIndexInEpoch,
	}
	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_BLOCK_FETCH_RESP_MESSAGE,
		Data: &chainedbftpb.ConsensusPayload_BlockFetchRespMsg{
			BlockFetchRespMsg: msg},
	}
	return consensusPayload
}
