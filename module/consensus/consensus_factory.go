/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consensus

import (
	"fmt"

	"chainmaker.org/chainmaker-go/consensus/dpos"

	"chainmaker.org/chainmaker-go/consensus/chainedbft"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"

	"chainmaker.org/chainmaker-go/consensus/raft"
	"chainmaker.org/chainmaker-go/consensus/solo"
	"chainmaker.org/chainmaker-go/consensus/tbft"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/protocol/v2"
)

type Factory struct {
}

// NewConsensusEngine new the consensus engine.
// consensusType specfies the consensus engine type.
// msgBus is used for send and receive messages.
func (f Factory) NewConsensusEngine(
	consensusType consensuspb.ConsensusType,
	chainID string,
	id string,
	nodeList []string,
	signer protocol.SigningMember,
	ac protocol.AccessControlProvider,
	dbHandle protocol.DBHandle,
	ledgerCache protocol.LedgerCache,
	proposalCache protocol.ProposalCache,
	blockVerifier protocol.BlockVerifier,
	blockCommitter protocol.BlockCommitter,
	netService protocol.NetService,
	msgBus msgbus.MessageBus,
	chainConf protocol.ChainConf,
	store protocol.BlockchainStore,
	helper protocol.HotStuffHelper,
) (protocol.ConsensusEngine, error) {
	switch consensusType {
	case consensuspb.ConsensusType_TBFT, consensuspb.ConsensusType_DPOS:
		config := tbft.ConsensusTBFTImplConfig{
			ChainID:     chainID,
			Id:          id,
			Signer:      signer,
			Ac:          ac,
			DbHandle:    dbHandle,
			LedgerCache: ledgerCache,
			ChainConf:   chainConf,
			NetService:  netService,
			MsgBus:      msgBus,
			Dpos:        dpos.NewDPoSImpl(chainConf, store),
		}

		return tbft.New(config)
	case consensuspb.ConsensusType_SOLO:
		return solo.New(chainID, id, signer, msgBus, chainConf)
	case consensuspb.ConsensusType_RAFT:
		config := raft.ConsensusRaftImplConfig{
			ChainID:        chainID,
			NodeId:         id,
			Singer:         signer,
			Ac:             ac,
			LedgerCache:    ledgerCache,
			BlockVerifier:  blockVerifier,
			BlockCommitter: blockCommitter,
			ChainConf:      chainConf,
			MsgBus:         msgBus,
		}
		return raft.New(config)
	case consensuspb.ConsensusType_HOTSTUFF:
		return chainedbft.New(chainID, id, signer, ac, ledgerCache,
			proposalCache, blockVerifier, blockCommitter, netService,
			store, msgBus, chainConf, helper)
	default:
	}
	return nil, fmt.Errorf("error consensusType: %s", consensusType)
}

// VerifyBlockSignatures verifies whether the signatures in block
// is qulified with the consensus algorithm. It should return nil
// error when verify successfully, and return corresponding error
// when failed.
func VerifyBlockSignatures(
	chainConf protocol.ChainConf,
	ac protocol.AccessControlProvider,
	store protocol.BlockchainStore,
	block *commonpb.Block,
	ledger protocol.LedgerCache,
) error {
	consensusType := chainConf.ChainConfig().Consensus.Type
	switch consensusType {
	case consensuspb.ConsensusType_TBFT, consensuspb.ConsensusType_DPOS:
		return tbft.VerifyBlockSignatures(chainConf, ac, block, store)
	case consensuspb.ConsensusType_RAFT:
		return raft.VerifyBlockSignatures(block)
	case consensuspb.ConsensusType_HOTSTUFF:
		return chainedbft.VerifyBlockSignatures(chainConf, ac, store, block, ledger)
	case consensuspb.ConsensusType_SOLO:
		fallthrough
	default:
	}
	return fmt.Errorf("error consensusType: %s", consensusType)
}
