/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package blockchain is an instance with an unique chainid. Will be initilized when chainmaker server startup.
package blockchain

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/net"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/subscriber"
)

// Blockchain is a block chain service. It manage all the modules of the chain.
type Blockchain struct {
	log *logger.CMLogger

	genesis string
	// chain id
	chainId string

	// message bus
	msgBus msgbus.MessageBus

	// net, shared with other blockchains
	net net.Net

	// netService
	netService protocol.NetService

	// store
	store protocol.BlockchainStore

	// consensus
	consensus protocol.ConsensusEngine

	// tx pool
	txPool protocol.TxPool

	// core engine
	coreEngine *core.CoreEngine

	// vm manager
	vmMgr protocol.VmManager

	// id management (idmgmt)
	identity protocol.SigningMember

	// access control
	ac protocol.AccessControlProvider

	// sync
	syncServer protocol.SyncService

	ledgerCache protocol.LedgerCache

	proposalCache protocol.ProposalCache

	snapshotManager protocol.SnapshotManager

	lastBlock *common.Block

	chainConf protocol.ChainConf

	// chainNodeList is the list of nodeIDs belong to this chain.
	chainNodeList []string

	eventSubscriber *subscriber.EventSubscriber

	spv protocol.Spv
}

// NewBlockchain create a new Blockchain instance.
func NewBlockchain(genesis string, chainId string, msgBus msgbus.MessageBus, net net.Net) *Blockchain {
	return &Blockchain{
		log:     logger.GetLoggerByChain(logger.MODULE_BLOCKCHAIN, chainId),
		genesis: genesis,
		chainId: chainId,
		msgBus:  msgBus,
		net:     net,
	}
}

func (bc *Blockchain) getConsensusType() consensus.ConsensusType {
	if bc.chainId == "" {
		panic("chainId is nil")
	}
	return bc.chainConf.ChainConfig().Consensus.Type
}

// GetAccessControl get the protocol.AccessControlProvider of instance.
func (bc *Blockchain) GetAccessControl() protocol.AccessControlProvider {
	return bc.ac
}
