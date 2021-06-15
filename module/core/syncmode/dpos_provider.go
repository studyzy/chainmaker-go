package syncmode

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider"
	"chainmaker.org/chainmaker-go/provider/conf"
)

const ConsensusTypeDPOS = "DPOS"

var NilDPOSProvider provider.CoreProvider = (*dposProvider)(nil)

type dposProvider struct {
}

func (tp *dposProvider) NewCoreEngine (config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
