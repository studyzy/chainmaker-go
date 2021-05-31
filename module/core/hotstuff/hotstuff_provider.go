/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hotstuff

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider"
	"chainmaker.org/chainmaker-go/provider/conf"
)

const ConsensusTypeHOTSTUFF = "HOTSTUFF"

var NilTHOTSTUFFProvider provider.CoreProvider = (*hotstuffProvider)(nil)

type hotstuffProvider struct {
}

func (hp *hotstuffProvider) NewCoreEngine (config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}