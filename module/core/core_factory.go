/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/core/provider"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v2"
)

type coreEngineFactory struct {
}

var once sync.Once
var _instance *coreEngineFactory

// Factory return the global core engine factory.
//nolint: revive
func Factory() *coreEngineFactory {
	once.Do(func() { _instance = new(coreEngineFactory) })
	return _instance
}

// NewConsensusEngine new the core engine.
// consensusType specifies the core engine type.
// consensusConfig specifies the necessary config parameters.
func (cf *coreEngineFactory) NewConsensusEngine(consensusType string,
	providerConf *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	p := provider.NewCoreEngineProviderByConsensusType(consensusType)
	var storeHelper conf.StoreHelper
	if providerConf.ChainConf.ChainConfig().Contract.EnableSqlSupport {
		storeHelper = common.NewSQLStoreHelper(providerConf.ChainConf.ChainConfig().ChainId)
	} else {
		storeHelper = common.NewKVStoreHelper(providerConf.ChainConf.ChainConfig().ChainId)
	}
	providerConf.StoreHelper = storeHelper

	return p.NewCoreEngine(providerConf)
}
