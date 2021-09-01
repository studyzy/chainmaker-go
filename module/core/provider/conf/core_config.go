/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package conf

import (
	"chainmaker.org/chainmaker-go/subscriber"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

type CoreEngineConfig struct {
	ChainId         string
	TxPool          protocol.TxPool
	SnapshotManager protocol.SnapshotManager
	MsgBus          msgbus.MessageBus
	Identity        protocol.SigningMember
	LedgerCache     protocol.LedgerCache
	ProposalCache   protocol.ProposalCache
	ChainConf       protocol.ChainConf
	AC              protocol.AccessControlProvider
	BlockchainStore protocol.BlockchainStore
	Log             protocol.Logger
	VmMgr           protocol.VmManager
	Subscriber      *subscriber.EventSubscriber // block subsriber
	StoreHelper     StoreHelper
}

type StoreHelper interface {
	RollBack(*commonpb.Block, protocol.BlockchainStore) error
	BeginDbTransaction(protocol.BlockchainStore, string)
	GetPoolCapacity() int
}
