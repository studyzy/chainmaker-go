/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/common/msgbus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/subscriber"
)

// CoreFactory is a factory to create core engine.
type CoreFactory struct {
	chainId string // 链标识

	msgBus          msgbus.MessageBus        // msgBus
	txPool          protocol.TxPool          // txpool
	vmMgr           protocol.VmManager       // vm manager
	blockchainStore protocol.BlockchainStore // blockchain store
	snapshotManager protocol.SnapshotManager // snapshot manager
	identity        protocol.SigningMember   // identity
	ledgerCache     protocol.LedgerCache     // ledger cache
	proposalCache   protocol.ProposalCache   // proposal cache
	chainConf       protocol.ChainConf       // chain config
	log             protocol.Logger          // chain config

	ac protocol.AccessControlProvider

	subscriber *subscriber.EventSubscriber
}

type CoreOption func(cf *CoreFactory) error

func WithMsgBus(msgBus msgbus.MessageBus) CoreOption {
	return func(cf *CoreFactory) error {
		cf.msgBus = msgBus
		return nil
	}
}

func WithSubscriber(subscriber *subscriber.EventSubscriber) CoreOption {
	return func(cf *CoreFactory) error {
		cf.subscriber = subscriber
		return nil
	}
}

func WithTxPool(txpool protocol.TxPool) CoreOption {
	return func(cf *CoreFactory) error {
		cf.txPool = txpool
		return nil
	}
}

func WithVmMgr(vmMgr protocol.VmManager) CoreOption {
	return func(cf *CoreFactory) error {
		cf.vmMgr = vmMgr
		return nil
	}
}

func WithBlockchainStore(blockchainStore protocol.BlockchainStore) CoreOption {
	return func(cf *CoreFactory) error {
		cf.blockchainStore = blockchainStore
		return nil
	}
}

func WithSnapshotManager(snapshotManager protocol.SnapshotManager) CoreOption {
	return func(cf *CoreFactory) error {
		cf.snapshotManager = snapshotManager
		return nil
	}
}

func WithSigningMember(identity protocol.SigningMember) CoreOption {
	return func(cf *CoreFactory) error {
		cf.identity = identity
		return nil
	}
}

func WithLedgerCache(ledgerCache protocol.LedgerCache) CoreOption {
	return func(cf *CoreFactory) error {
		cf.ledgerCache = ledgerCache
		return nil
	}
}

func WithChainId(chainId string) CoreOption {
	return func(cf *CoreFactory) error {
		cf.chainId = chainId
		return nil
	}
}

func WithChainConf(chainConf protocol.ChainConf) CoreOption {
	return func(cf *CoreFactory) error {
		cf.chainConf = chainConf
		return nil
	}
}

func WithAccessControl(ac protocol.AccessControlProvider) CoreOption {
	return func(cf *CoreFactory) error {
		cf.ac = ac
		return nil
	}
}

func WithProposalCache(pc protocol.ProposalCache) CoreOption {
	return func(cf *CoreFactory) error {
		cf.proposalCache = pc
		return nil
	}
}

func WithCoreLogger(log protocol.Logger) CoreOption {
	return func(cf *CoreFactory) error {
		cf.log = log
		return nil
	}
}

func (cf *CoreFactory) NewCoreWithOptions(opts ...CoreOption) (*CoreEngine, error) {
	if err := cf.Apply(opts...); err != nil {
		return nil, err
	}
	core, err := NewCoreEngine(cf)
	if err != nil {
		return nil, err
	}
	return core, nil
}

func (cf *CoreFactory) Apply(opts ...CoreOption) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cf); err != nil {
			return err
		}
	}
	return nil
}
