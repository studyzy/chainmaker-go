/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package syncmode

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider"
	"chainmaker.org/chainmaker-go/provider/conf"
)

const ConsensusTypeSOLO = "SOLO"

var NilSOLOProvider provider.CoreProvider = (*soloProvider)(nil)

type soloProvider struct {
}

func (sp *soloProvider) NewCoreEngine (config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}