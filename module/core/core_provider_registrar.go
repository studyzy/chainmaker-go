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
	provider.RegisterCoreEngineProvider(hotstuffMode.ConsensusTypeHOTSTUFF, hotstuffMode.NilTHOTSTUFFProvider)
}
