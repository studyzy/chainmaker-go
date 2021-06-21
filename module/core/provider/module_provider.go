package provider

import (
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/protocol"
)

type CoreProvider interface {
	NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error)
}
