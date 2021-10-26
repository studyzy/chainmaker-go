/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package solo

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"chainmaker.org/chainmaker/logger/v2"
)

var clog *logger.CMLogger

// ConsensusSoloImpl is the implementation of solo algorithm
// and it implements the ConsensusEngine interface.
type ConsensusSoloImpl struct {
	chainID string
	id      string
	singer  protocol.SigningMember
	msgbus  msgbus.MessageBus

	verifyingBlock *common.Block

	mtx       sync.Mutex
	chainConf protocol.ChainConf
}

//New ...
func New(chainID string, uid string, singer protocol.SigningMember,
	msgBus msgbus.MessageBus, chainConf protocol.ChainConf) (*ConsensusSoloImpl, error) {
	clog = logger.GetLoggerByChain(logger.MODULE_CONSENSUS, chainID)

	clog.Infof("New ConsensusSoloImpl with uid: %s", uid)

	consensus := &ConsensusSoloImpl{}
	consensus.chainID = chainID
	consensus.id = uid
	consensus.singer = singer
	consensus.msgbus = msgBus
	consensus.chainConf = chainConf

	return consensus, nil
}

//Start implements the Init method of ConsensusEngine interface.
func (consensus *ConsensusSoloImpl) Start() error {
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.VerifyResult, consensus)
	go consensus.procProposerStatus()

	clog.Infof("ConsensusSoloImpl %s started", consensus.id)
	return nil
}

// Stop implements the Stop method of ConsensusEngine interface.
// TODO: implement Stop method
func (consensus *ConsensusSoloImpl) Stop() error {
	clog.Infof("ConsensusSoloImpl %s stoped", consensus.id)
	return nil
}

// procProposerStatus send proposerState msgBus to core
func (consensus *ConsensusSoloImpl) procProposerStatus() {
	// wait core to start, then publish ProposerState to core, only execute once
	time.Sleep(3 * time.Second)
	consensus.msgbus.Publish(msgbus.ProposeState, true)
}

//OnMessage receive core busmsg and process block
func (consensus *ConsensusSoloImpl) OnMessage(message *msgbus.Message) {
	clog.Infof("%s OnMessage receive topic: %s", consensus.id, message.Topic)
	switch message.Topic {
	case msgbus.ProposedBlock:
		consensus.handleProposedBlock(message)
	case msgbus.VerifyResult:
		consensus.handleVerifyResult(message)
	}
}

//handleProposedBlock process proposedblock msg
func (consensus *ConsensusSoloImpl) handleProposedBlock(message *msgbus.Message) {
	if _, ok := message.Payload.(*consensuspb.ProposalBlock); !ok {
		clog.Errorf("id: %s ProposedBlock msg is invaild",
			consensus.id)
		return
	}

	proposedBlock, ok := message.Payload.(*consensuspb.ProposalBlock)
	if !ok {
		panic("message.Payload not a ProposalBlock")
	}
	block := proposedBlock.Block
	clog.Infof("handle proposedBlock start, id: %s, height: %d", consensus.id, block.Header.BlockHeight)
	clog.DebugDynamic(func() string {
		str := fmt.Sprintf("ProposedBlock block: %v", block)
		if len(str) > 2048 {
			str = str[:2048] + " ......"
		}
		return str
	})

	hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
	if err != nil {
		clog.Errorf("%s sign block error %s", consensus.id, err)
	}

	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig

	consensus.mtx.Lock()
	defer consensus.mtx.Unlock()

	consensus.verifyingBlock = block
	consensus.msgbus.Publish(msgbus.VerifyBlock, block)

}

//handleVerifyResult process verifyresult msg
func (consensus *ConsensusSoloImpl) handleVerifyResult(message *msgbus.Message) {
	if _, ok := message.Payload.(*consensuspb.VerifyResult); !ok {
		clog.Errorf("id: %s verifyingBlock msg is invaild",
			consensus.id)
		return
	}

	consensus.mtx.Lock()
	defer consensus.mtx.Unlock()

	verifyResult, ok := message.Payload.(*consensuspb.VerifyResult)
	if !ok {
		panic("message.Payload not a VerifyResult")
	}
	clog.Infof("handle verifyResult start, id: %s verifyResult: %s BlockInfo: %v",
		consensus.id, verifyResult.Code, verifyResult.VerifiedBlock.Header.BlockHeight)
	clog.DebugDynamic(func() string {
		str := fmt.Sprintf("verifyingBlock: %v", consensus.verifyingBlock)
		if len(str) > 2048 {
			str = str[:2048] + " ......"
		}
		return str
	})

	if consensus.verifyingBlock == nil {
		clog.Errorf("%s CommitBlock verifyingBlock nil")
		return
	}
	//check block height
	if verifyResult.VerifiedBlock.Header.BlockHeight !=
		consensus.verifyingBlock.Header.BlockHeight {
		clog.Errorf("unmatch block height %d", verifyResult.VerifiedBlock.Header.BlockHeight)
		return
	}
	//check block hash
	if ok := bytes.Equal(verifyResult.VerifiedBlock.Header.BlockHash,
		consensus.verifyingBlock.Header.BlockHash); !ok {
		clog.Errorf("unmatch block hash %s", verifyResult.VerifiedBlock.Header.BlockHash)
		return
	}
	if verifyResult.Code == consensuspb.VerifyResult_FAIL {
		clog.Errorf("block verified failed")
		consensus.verifyingBlock = nil
		return
	}
	clog.Infof("publish CommitBlock, height = %v", consensus.verifyingBlock.Header.BlockHeight)
	consensus.msgbus.Publish(msgbus.CommitBlock, consensus.verifyingBlock)
	consensus.verifyingBlock = nil
}

//OnQuit ...
func (consensus *ConsensusSoloImpl) OnQuit() {
	clog.Infof("on quit")
}

//CanProposeBlock ...
func (consensus *ConsensusSoloImpl) CanProposeBlock() bool {
	return true
}

// TODO
func (consensus *ConsensusSoloImpl) VerifyBlockSignatures(block *common.Block) error {
	return nil
}
