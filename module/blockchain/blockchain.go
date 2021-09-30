/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package blockchain is an instance with an unique chainid. Will be initilized when chainmaker server startup.
package blockchain

import (
	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
)

const (
	moduleNameSubscriber    = "Subscriber"
	moduleNameStore         = "Store"
	moduleNameLedger        = "Ledger"
	moduleNameChainConf     = "ChainConf"
	moduleNameAccessControl = "AccessControl"
	moduleNameNetService    = "NetService"
	moduleNameVM            = "VM"
	moduleNameTxPool        = "TxPool"
	moduleNameCore          = "Core"
	moduleNameConsensus     = "Consensus"
	moduleNameSync          = "Sync"
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
	net protocol.Net

	// netService
	netService protocol.NetService

	// store
	store protocol.BlockchainStore

	// consensus
	consensus protocol.ConsensusEngine

	// tx pool
	txPool protocol.TxPool

	// core engine
	coreEngine protocol.CoreEngine

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

	initModules  map[string]struct{}
	startModules map[string]struct{}
}

// NewBlockchain create a new Blockchain instance.
func NewBlockchain(genesis string, chainId string, msgBus msgbus.MessageBus, net protocol.Net) *Blockchain {
	return &Blockchain{
		log:          logger.GetLoggerByChain(logger.MODULE_BLOCKCHAIN, chainId),
		genesis:      genesis,
		chainId:      chainId,
		msgBus:       msgBus,
		net:          net,
		initModules:  make(map[string]struct{}),
		startModules: make(map[string]struct{}),
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
