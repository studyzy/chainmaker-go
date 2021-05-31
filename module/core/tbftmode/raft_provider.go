/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbftmode

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider"
	"chainmaker.org/chainmaker-go/provider/conf"
)

const ConsensusTypeRAFT = "RAFT"

var NilRAFTProvider provider.CoreProvider = (*raftProvider)(nil)

type raftProvider struct {
}

func (rp *raftProvider) NewCoreEngine (config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}