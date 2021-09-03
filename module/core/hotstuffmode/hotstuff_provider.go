/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hotstuffmode

import (
	"chainmaker.org/chainmaker-go/core/provider"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v2"
)

const ConsensusTypeHOTSTUFF = "HOTSTUFF"

var NilTHOTSTUFFProvider provider.CoreProvider = (*hotstuffProvider)(nil)

type hotstuffProvider struct {
}

func (hp *hotstuffProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
