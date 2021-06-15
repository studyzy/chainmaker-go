package conf

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/subscriber"
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