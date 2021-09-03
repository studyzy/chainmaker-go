/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package syncmode

import (
	"chainmaker.org/chainmaker-go/core/provider"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v2"
)

const ConsensusTypeTBFT = "TBFT"

var NilTBFTProvider provider.CoreProvider = (*tbftProvider)(nil)

type tbftProvider struct {
}

func (tp *tbftProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
