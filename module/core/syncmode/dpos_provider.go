package syncmode

import (
	"chainmaker.org/chainmaker-go/core/provider"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker-go/protocol"
)

const ConsensusTypeDPOS = "DPOS"

var NilDPOSProvider provider.CoreProvider = (*dposProvider)(nil)

type dposProvider struct {
}

func (tp *dposProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
