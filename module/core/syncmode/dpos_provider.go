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

const ConsensusTypeDPOS = "DPOS"

var NilDPOSProvider provider.CoreProvider = (*dposProvider)(nil)

type dposProvider struct {
}

func (tp *dposProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
