package conf

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/logger"
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
	Log             *logger.CMLogger
	VmMgr           protocol.VmManager
	Subscriber      *subscriber.EventSubscriber // block subsriber
}
