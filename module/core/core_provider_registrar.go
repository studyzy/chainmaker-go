/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	hotstuffMode "chainmaker.org/chainmaker-go/core/hotstuffmode"
	"chainmaker.org/chainmaker-go/core/provider"
	syncMode "chainmaker.org/chainmaker-go/core/syncmode"
)

func init() {
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeSOLO, syncMode.NilSOLOProvider)
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeRAFT, syncMode.NilRAFTProvider)
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeTBFT, syncMode.NilTBFTProvider)
	provider.RegisterCoreEngineProvider(syncMode.ConsensusTypeDPOS, syncMode.NilDPOSProvider)
	provider.RegisterCoreEngineProvider(hotstuffMode.ConsensusTypeHOTSTUFF, hotstuffMode.NilTHOTSTUFFProvider)
}
