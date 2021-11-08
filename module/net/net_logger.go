/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"chainmaker.org/chainmaker/logger/v2"
	liquid "chainmaker.org/chainmaker/net-liquid/liquidnet"
	"chainmaker.org/chainmaker/protocol/v2"
)

var GlobalNetLogger protocol.Logger

func init() {
	GlobalNetLogger = logger.GetLogger(logger.MODULE_NET)
	liquid.InitLogger(GlobalNetLogger, func(chainId string) protocol.Logger {
		return logger.GetLoggerByChain(logger.MODULE_NET, chainId)
	})
}
