/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/consensus/government"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	chainedbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
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
	if txRWSet, err = government.CheckAndCreateGovernmentArgs(block, cbi.store, cbi.proposalCache); err != nil {
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
		JustifyQC:   qc,
	}
	syncInfo := &chainedbftpb.SyncInfo{
		HighestQC:      qc,
		HighestTC:      cbi.smr.getTC(),
		HighestTCLevel: cbi.smr.getHighestTCLevel(),
	}
	proposalMsg := &chainedbftpb.ProposalMsg{
		SyncInfo:     syncInfo,
		ProposalData: proposalData,
	}
	cbi.logger.Debugf("service selfIndexInEpoch [%v] constructProposal, proposal: [%v:%v:%v], JustifyQC: %v, HighestTC: %v",
		cbi.selfIndexInEpoch, proposalData.ProposerIdx, proposalData.Height, proposalData.Level, qc, syncInfo.HighestTC)

	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_ProposalMessage,
		Data: &chainedbftpb.ConsensusPayload_ProposalMsg{proposalMsg},
	}
	return consensusPayload
}

//constructVote builds a vote msg with given params
func (cbi *ConsensusChainedBftImpl) constructVote(height uint64, level uint64, epochId uint64,
	block *common.Block) *chainedbftpb.ConsensusPayload {
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
		voteData.BlockID = block.Header.BlockHash
	}
	var (
		err  error
		data []byte
		sign []byte
	)
	if data, err = proto.Marshal(voteData); err != nil {
		return nil
	}
	if sign, err = cbi.singer.Sign(cbi.chainConf.ChainConfig().Crypto.Hash, data); err != nil {
		cbi.logger.Errorf("sign data failed, err %v data %v", err, data)
		return nil
	}

	voteData.Signature = &common.EndorsementEntry{
		Signer:    nil,
		Signature: sign,
	}
	syncInfo := &chainedbftpb.SyncInfo{
		HighestTC:      cbi.smr.getTC(),
		HighestQC:      cbi.chainStore.getCurrentQC(),
		HighestTCLevel: cbi.smr.getHighestTCLevel(),
	}
	vote := &chainedbftpb.VoteMsg{
		VoteData: voteData,
		SyncInfo: syncInfo,
	}
	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_VoteMessage,
		Data: &chainedbftpb.ConsensusPayload_VoteMsg{vote},
	}
	return consensusPayload
}

//constructBlockFetchMsg builds a block fetch request msg at given height
func (cbi *ConsensusChainedBftImpl) constructBlockFetchMsg(blockID []byte,
	height uint64, num uint64) *chainedbftpb.ConsensusPayload {
	msg := &chainedbft.BlockFetchMsg{
		Height:    height,
		BlockID:   blockID,
		NumBlocks: num,
		AuthorIdx: cbi.selfIndexInEpoch,
	}
	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_BlockFetchMessage,
		Data: &chainedbftpb.ConsensusPayload_BlockFetchMsg{msg},
	}
	return consensusPayload
}

//constructBlockFetchRespMsg builds a block fetch response with given params
func (cbi *ConsensusChainedBftImpl) constructBlockFetchRespMsg(blocks []*chainedbft.BlockPair,
	status chainedbft.BlockFetchStatus) *chainedbftpb.ConsensusPayload {
	msg := &chainedbft.BlockFetchRespMsg{
		Status:    status,
		Blocks:    blocks,
		AuthorIdx: cbi.selfIndexInEpoch,
	}
	consensusPayload := &chainedbftpb.ConsensusPayload{
		Type: chainedbftpb.MessageType_BlockFetchRespMessage,
		Data: &chainedbftpb.ConsensusPayload_BlockFetchRespMsg{msg},
	}
	return consensusPayload
}
