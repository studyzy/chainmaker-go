package net

import (
	liquid "chainmaker.org/chainmaker/chainmaker-net-liquid/liquidnet"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/protocol/v2"
)

var GlobalNetLogger protocol.Logger

func init() {
	GlobalNetLogger = logger.GetLogger(logger.MODULE_NET)
	liquid.InitLogger(GlobalNetLogger, func(chainId string) protocol.Logger {
		return logger.GetLoggerByChain(logger.MODULE_NET, chainId)
	})
}
