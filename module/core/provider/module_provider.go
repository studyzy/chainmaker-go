/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v2"
)

type CoreProvider interface {
	NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error)
}
