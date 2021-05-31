package core

import (
	"chainmaker.org/chainmaker-go/core/hotstuff"
	tbftMode "chainmaker.org/chainmaker-go/core/tbftmode"
	"chainmaker.org/chainmaker-go/provider"
)

func init() {
	provider.RegisterCoreEngineProvider(tbftMode.ConsensusTypeSOLO, tbftMode.NilSOLOProvider)
	provider.RegisterCoreEngineProvider(tbftMode.ConsensusTypeRAFT, tbftMode.NilRAFTProvider)
	provider.RegisterCoreEngineProvider(tbftMode.ConsensusTypeTBFT, tbftMode.NilTBFTProvider)
	provider.RegisterCoreEngineProvider(hotstuff.ConsensusTypeHOTSTUFF, hotstuff.NilTHOTSTUFFProvider)
}
