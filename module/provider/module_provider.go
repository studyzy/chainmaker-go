package provider

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/provider/conf"
)

type CoreProvider interface {
	NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error)
}
