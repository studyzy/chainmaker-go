/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package message

import (
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
)

//ValidateMessageBasicInfo is an external api to check a msg's basic info
func ValidateMessageBasicInfo(payload *chainedbft.ConsensusPayload) error {
	if payload == nil {
		return fmt.Errorf("nil consensus payload")
	}

	switch payload.Type {
	case chainedbft.MessageType_PROPOSAL_MESSAGE:
		return validateProposalBasicInfo(payload)
	case chainedbft.MessageType_VOTE_MESSAGE:
		return validateVoteBasicInfo(payload)
	case chainedbft.MessageType_BLOCK_FETCH_MESSAGE:
		return validateBlockFetchBasicInfo(payload)
	case chainedbft.MessageType_BLOCK_FETCH_RESP_MESSAGE:
		return validateBlockFetchRespBasicInfo(payload)
	}
	return nil
}

//validateProposalBasicInfo checks whether the received proposal msg's basic info is valid
func validateProposalBasicInfo(payload *chainedbft.ConsensusPayload) error {
	proposalMsg := payload.GetProposalMsg()
	if proposalMsg == nil {
		return fmt.Errorf("nil proposal msg")
	}
	if proposalMsg.SyncInfo == nil {
		return fmt.Errorf("nil sync info in proposal msg")
	}
	if proposalMsg.SyncInfo.HighestQc == nil {
		return fmt.Errorf("nil highest qc in sync info within proposal msg")
	}

	proposal := proposalMsg.ProposalData
	if proposal == nil {
		return fmt.Errorf("nil proposal Data")
	}
	if proposal.Block == nil {
		return fmt.Errorf("nil block in proposal msg")
	}
	if proposal.Block.Header == nil {
		return fmt.Errorf("nil block header in proposal msg")
	}
	if proposal.Block.Header.PreBlockHash == nil {
		return fmt.Errorf("nil previous block hash in block %v", proposal.Block)
	}
	if proposal.Block.Header.Signature == nil {
		return fmt.Errorf("nil signature in block %v", proposal.Block)
	}
	if proposal.Proposer == nil {
		return fmt.Errorf("nil proposer address in proposal msg")
	}
	if proposal.JustifyQc == nil {
		return fmt.Errorf("nil justify qc in proposal msg")
	}
	return nil
}

//validateVoteBasicInfo checks whether the received vote msg's basic info is valid
func validateVoteBasicInfo(payload *chainedbft.ConsensusPayload) error {
	voteMsg := payload.GetVoteMsg()
	if voteMsg == nil {
		return fmt.Errorf("nil vote msg")
	}
	if voteMsg.SyncInfo == nil {
		return fmt.Errorf("nil sync info in vote msg")
	}
	if voteMsg.SyncInfo.HighestQc == nil {
		return fmt.Errorf("nil highest qc in sync info within vote msg")
	}

	vote := voteMsg.VoteData
	if vote == nil {
		return fmt.Errorf("nil vote data")
	}
	if !vote.NewView && vote.BlockId == nil {
		return fmt.Errorf("not voted for newView and block")
	}
	if vote.Author == nil {
		return fmt.Errorf("nil author in vote msg")
	}
	return nil
}

//validateBlockFetchBasicInfo checks whether the received block fetch msg's basic info is valid
func validateBlockFetchBasicInfo(payload *chainedbft.ConsensusPayload) error {
	blockFetchMsg := payload.GetBlockFetchMsg()
	if blockFetchMsg == nil {
		return fmt.Errorf("nil block fetch msg")
	}
	if blockFetchMsg.BlockId == nil {
		return fmt.Errorf("nil block id")
	}
	return nil
}

//validateBlockFetchRespBasicInfo checks whether the received block fetch resp msg's basic info is valid
func validateBlockFetchRespBasicInfo(payload *chainedbft.ConsensusPayload) error {
	blockFetchResp := payload.GetBlockFetchRespMsg()
	if blockFetchResp == nil {
		return fmt.Errorf("nil block fetch rsp msg")
	}
	if blockFetchResp.Status == chainedbft.BlockFetchStatus_SUCCEEDED &&
		blockFetchResp.Blocks == nil {
		return fmt.Errorf("empty blocks from block fetch rsp msg")
	}

	for _, pair := range blockFetchResp.Blocks {
		if pair.Block == nil || pair.Qc == nil {
			return fmt.Errorf("nil block or nil qc in block pair from block fetch rsp msg")
		}
		if pair.Block.Header == nil {
			return fmt.Errorf("nil block header from block fetch rsp msg")
		}
		if pair.Block.Header.PreBlockHash == nil {
			return fmt.Errorf("nil previous block hash in block %v from block fetch rsp", pair)
		}
		if pair.Block.Header.Signature == nil {
			return fmt.Errorf("nil signature in block %v from block fetch rsp", pair)
		}
	}

	return nil
}
