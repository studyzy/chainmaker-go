/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus"
)

// ChainConf chainconf interface
type ChainConf interface {
	Init() error                                                             // init
	ChainConfig() *config.ChainConfig                                        // get the latest chainconfig
	GetChainConfigFromFuture(blockHeight int64) (*config.ChainConfig, error) // get chainconfig by (blockHeight-1)
	GetChainConfigAt(blockHeight int64) (*config.ChainConfig, error)         // get chainconfig by blockHeight
	GetConsensusNodeIdList() ([]string, error)                               // get node list
	CompleteBlock(block *common.Block) error                                 // callback after insert block to db success
	AddWatch(w Watcher)                                                      // add watcher
	AddVmWatch(w VmWatcher)                                                  // add vm watcher
}

// Watcher chainconfig watcher
type Watcher interface {
	Module() string                              // module
	Watch(chainConfig *config.ChainConfig) error // callback the chainconfig
}

// Verifier verify consensus data
type Verifier interface {
	Verify(consensusType consensus.ConsensusType, chainConfig *config.ChainConfig) error
}

// VmWatcher native vm watcher
type VmWatcher interface {
	Module() string                                          // module
	ContractNames() []string                                 // watch the contract
	Callback(contractName string, payloadBytes []byte) error // callback
}
