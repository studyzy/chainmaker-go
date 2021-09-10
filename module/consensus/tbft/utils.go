/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"chainmaker.org/chainmaker-go/consensus/dpos"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	tbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/tbft"
	"chainmaker.org/chainmaker/protocol/v2"

	"github.com/gogo/protobuf/proto"
)

func GetValidatorList(chainConfig *config.ChainConfig, store protocol.BlockchainStore) (validators []string,
	err error) {
	if chainConfig.Consensus.Type == consensus.ConsensusType_TBFT {
		return GetValidatorListFromConfig(chainConfig)
	} else if chainConfig.Consensus.Type == consensus.ConsensusType_DPOS {
		return GetValidatorListFromLedger(store)
	}
	return nil, fmt.Errorf("unknown consensus type: %s", chainConfig.Consensus.Type)
}

func GetValidatorListFromConfig(chainConfig *config.ChainConfig) (validators []string, err error) {
	nodes := chainConfig.Consensus.Nodes
	for _, node := range nodes {
		//for _, nid := range node.NodeId {
		//	validators = append(validators, nid)
		//}
		validators = append(validators, node.NodeId...)
	}
	return validators, nil
}

func GetValidatorListFromLedger(store protocol.BlockchainStore) (validators []string, err error) {
	epoch, err := dpos.GetLatestEpochInfo(store)
	if err != nil {
		return nil, err
	}
	nodeIDs, err := dpos.GetNodeIDsFromValidators(store, epoch.ProposerVector)
	if err != nil {
		return nil, err
	}
	return nodeIDs, nil
}

// VerifyBlockSignatures verifies whether the signatures in block
// is qulified with the consensus algorithm. It should return nil
// error when verify successfully, and return corresponding error
// when failed.
func VerifyBlockSignatures(chainConf protocol.ChainConf,
	ac protocol.AccessControlProvider, block *common.Block, store protocol.BlockchainStore) error {

	if block == nil || block.Header == nil || block.AdditionalData == nil || block.AdditionalData.ExtraData == nil {
		return fmt.Errorf("invalid block")
	}
	blockVoteSet, ok := block.AdditionalData.ExtraData[protocol.TBFTAddtionalDataKey]
	if !ok {
		return fmt.Errorf("block.AdditionalData.ExtraData[TBFTAddtionalDataKey] not exist")
	}

	voteSetProto := new(tbftpb.VoteSet)
	if err := proto.Unmarshal(blockVoteSet, voteSetProto); err != nil {
		return err
	}

	height := block.Header.BlockHeight
	chainConfig, err := chainConf.GetChainConfigFromFuture(height)
	if err != nil {
		return err
	}

	validators, err := GetValidatorList(chainConfig, store)
	if err != nil {
		return err
	}

	logger := logger.GetLoggerByChain(logger.MODULE_CONSENSUS, chainConfig.ChainId)
	validatorSet := newValidatorSet(logger, validators, DefaultBlocksPerProposer)
	voteSet := NewVoteSetFromProto(logger, voteSetProto, validatorSet)
	hash, ok := voteSet.twoThirdsMajority()
	if !ok {
		return fmt.Errorf("voteSet without majority")
	}

	if !bytes.Equal(hash, block.Header.BlockHash) {
		return fmt.Errorf("unmatch QC: %x to block hash: %v", hash, block.Header.BlockHash)
	}

	hashStr := base64.StdEncoding.EncodeToString(hash)
	blockVotes := voteSet.VotesByBlock[hashStr]
	// blockVotes should contain valid vote only, otherwise the block is invalid
	for _, v := range blockVotes.Votes {
		voteProto := v.ToProto()
		voteProtoCopy := proto.Clone(voteProto)
		vote, ok := voteProtoCopy.(*tbftpb.Vote)
		if !ok {
			return fmt.Errorf("interface transfer to *tbftpb.Vote failed")
		}
		vote.Endorsement = nil
		message := mustMarshal(vote)

		principal, err := ac.CreatePrincipal(
			protocol.ResourceNameConsensusNode,
			[]*common.EndorsementEntry{voteProto.Endorsement},
			message,
		)
		if err != nil {
			clog.Infof("verify block signatures block(%d-%x) error: %v",
				block.Header.BlockHeight, block.Header.BlockHash, err)
			return err
		}

		result, err := ac.VerifyPrincipal(principal)
		if err != nil {
			clog.Infof("verify block signatures block(%d-%x) error: %v",
				block.Header.BlockHeight, block.Header.BlockHash, err)
			return err
		}

		if !result {
			clog.Infof("verify block signatures block(%d-%x) error because result: %v",
				block.Header.BlockHeight, block.Header.BlockHash, result)
			return fmt.Errorf("verifyVote result: %v", result)
		}
	}

	clog.Debugf("VerifyBlockSignatures block (%d-%x) success",
		block.Header.BlockHeight, block.Header.BlockHash)
	return nil
}
