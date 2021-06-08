/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider"
	"chainmaker.org/chainmaker-go/provider/conf"
	"sync"
)

type coreEngineFactory struct {
}

var once sync.Once
var _instance *coreEngineFactory

// Factory return the global core engine factory.
func Factory() *coreEngineFactory {
	once.Do(func() { _instance = new(coreEngineFactory) })
	return _instance
}

// NewCoreEngine new the core engine.
// consensusType specifies the core engine type.
// consensusConfig specifies the necessary config parameters.
func (cf *coreEngineFactory) NewConsensusEngine(consensusType string, consensusConfig *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	p := provider.NewCoreEngineProviderByConsensusType(consensusType)
	return p.NewCoreEngine(consensusConfig)
}
