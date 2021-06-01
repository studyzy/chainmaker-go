/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hotstuffmode

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/core/hotstuffmode/proposer"
	"chainmaker.org/chainmaker-go/core/hotstuffmode/verifier"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/chainedbft"
	txpoolpb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider/conf"
	"chainmaker.org/chainmaker-go/subscriber"
	"github.com/google/martian/log"
)

// CoreEngine is a block handle engine.
// One core engine for one chain.
type CoreEngine struct {
	chainId   string             // chainId, identity of a chain
	chainConf protocol.ChainConf // chain config

	msgBus         msgbus.MessageBus       // message bus, transfer messages with other modules
	blockProposer  protocol.BlockProposer  // block proposer, to generate new block when node is proposer
	BlockVerifier  protocol.BlockVerifier  // block verifier, to verify block that proposer generated
	BlockCommitter protocol.BlockCommitter // block committer, to commit block to store after consensus
	txScheduler    protocol.TxScheduler    // transaction scheduler, schedule transactions run in vm
	HotStuffHelper protocol.HotStuffHelper

	txPool          protocol.TxPool          // transaction pool, cache transactions to be pack in block
	vmMgr           protocol.VmManager       // vm manager
	blockchainStore protocol.BlockchainStore // blockchain store, to store block, transactions in DB
	snapshotManager protocol.SnapshotManager // snapshot manager, manage state data that not store yet

	quitC         <-chan interface{}          // quit chan, reserved for stop core engine running
	proposedCache protocol.ProposalCache      // cache proposed block and proposal status
	log           protocol.Logger             // logger
	subscriber    *subscriber.EventSubscriber // block subsriber
}

type CoreEngineConfig struct {
	ChainId         string
	MsgBus          msgbus.MessageBus
	ChainConf       protocol.ChainConf
	TxPool          protocol.TxPool
	VmMgr           protocol.VmManager
	BlockchainStore protocol.BlockchainStore
	SnapshotManager protocol.SnapshotManager
	Identity        protocol.SigningMember
	LedgerCache     protocol.LedgerCache
	ProposalCache   protocol.ProposalCache // proposal cache
	AC              protocol.AccessControlProvider
	Subscriber      *subscriber.EventSubscriber
}

// NewCoreEngine new a core engine.
func NewCoreEngine(cf *conf.CoreEngineConfig) (*CoreEngine, error) {
	core := &CoreEngine{
		msgBus:          cf.MsgBus,
		txPool:          cf.TxPool,
		vmMgr:           cf.VmMgr,
		blockchainStore: cf.BlockchainStore,
		snapshotManager: cf.SnapshotManager,
		proposedCache:   cf.ProposalCache,
		chainConf:       cf.ChainConf,
		txScheduler:     common.NewTxScheduler(cf.VmMgr, cf.ChainConf),
		log:             cf.Log,
	}
	core.quitC = make(<-chan interface{})

	var err error
	// new a bock proposer
	proposerConfig := proposer.BlockProposerConfig{
		ChainId:         cf.ChainId,
		TxPool:          core.txPool,
		SnapshotManager: core.snapshotManager,
		MsgBus:          cf.MsgBus,
		Identity:        cf.Identity,
		LedgerCache:     cf.LedgerCache,
		TxScheduler:     core.txScheduler,
		ProposalCache:   core.proposedCache,
		ChainConf:       cf.ChainConf,
		AC:              cf.AC,
		BlockchainStore: cf.BlockchainStore,
	}
	core.blockProposer, err = proposer.NewBlockProposer(proposerConfig, cf.Log)
	if err != nil {
		return nil, err
	}

	// new a block verifier
	verifierConfig := verifier.BlockVerifierConfig{
		ChainId:         cf.ChainId,
		MsgBus:          cf.MsgBus,
		SnapshotManager: core.snapshotManager,
		BlockchainStore: core.blockchainStore,
		LedgerCache:     cf.LedgerCache,
		TxScheduler:     core.txScheduler,
		ProposedCache:   core.proposedCache,
		ChainConf:       cf.ChainConf,
		AC:              cf.AC,
		TxPool:          core.txPool,
		VmMgr:           cf.VmMgr,
	}
	core.BlockVerifier, err = verifier.NewBlockVerifier(verifierConfig, cf.Log)
	if err != nil {
		return nil, err
	}

	// new a block committer
	committerConfig := common.BlockCommitterConfig{
		ChainId:         cf.ChainId,
		BlockchainStore: core.blockchainStore,
		SnapshotManager: core.snapshotManager,
		TxPool:          core.txPool,
		LedgerCache:     cf.LedgerCache,
		ProposedCache:   core.proposedCache,
		ChainConf:       cf.ChainConf,
		MsgBus:          cf.MsgBus,
		Subscriber:      cf.Subscriber,
		Verifier:        core.BlockVerifier,
	}
	core.BlockCommitter, err = common.NewBlockCommitter(committerConfig, cf.Log)
	if err != nil {
		return nil, err
	}

	return core, nil
}

// OnQuit called when quit subsribe message from message bus
func (c *CoreEngine) OnQuit() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreEngine) OnMessage(message *msgbus.Message) {
	// 1. receive proposal status from consensus
	// 2. receive verify block from consensus
	// 3. receive commit block message from consensus
	// 4. receive propose signal from txpool
	// 5. receive build proposal signal from chained bft consensus

	switch message.Topic {
	case msgbus.ProposeState:
		if proposeStatus, ok := message.Payload.(bool); ok {
			c.blockProposer.OnReceiveProposeStatusChange(proposeStatus)
		}
	case msgbus.VerifyBlock:
		if block, ok := message.Payload.(*commonpb.Block); ok {
			c.BlockVerifier.VerifyBlock(block, protocol.CONSENSUS_VERIFY)
		}
	case msgbus.CommitBlock:
		if block, ok := message.Payload.(*commonpb.Block); ok {
			if err := c.BlockCommitter.AddBlock(block); err != nil {
				c.log.Warnf("put block(%d,%x) error %s",
					block.Header.BlockHeight,
					block.Header.BlockHash,
					err.Error())
			}
		}
	case msgbus.TxPoolSignal:
		if signal, ok := message.Payload.(*txpoolpb.TxPoolSignal); ok {
			c.blockProposer.OnReceiveTxPoolSignal(signal)
		}
	case msgbus.BuildProposal:
		if proposal, ok := message.Payload.(*chainedbft.BuildProposal); ok {
			c.blockProposer.OnReceiveChainedBFTProposal(proposal)
		}
	}
}

// Start, initialize core engine
func (c *CoreEngine) Start() {
	c.msgBus.Register(msgbus.ProposeState, c)
	c.msgBus.Register(msgbus.VerifyBlock, c)
	c.msgBus.Register(msgbus.CommitBlock, c)
	c.msgBus.Register(msgbus.TxPoolSignal, c)
	c.msgBus.Register(msgbus.BuildProposal, c)
	c.blockProposer.Start()
}

// Stop, stop core engine
func (c *CoreEngine) Stop() {
	defer log.Infof("core stoped.")
	c.blockProposer.Stop()
}

func (c *CoreEngine) GetBlockCommitter() protocol.BlockCommitter {
	return c.BlockCommitter
}

func (c *CoreEngine) GetBlockVerifier() protocol.BlockVerifier {
	return c.BlockVerifier
}

func (c *CoreEngine) DiscardAboveHeight(baseHeight int64) {
	c.log.Debugf("DiscardAboveHeight ..11")
}

func (c *CoreEngine) GetHotStuffHelper() protocol.HotStuffHelper {
	return c.HotStuffHelper
}
