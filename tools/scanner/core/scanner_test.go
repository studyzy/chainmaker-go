/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"testing"

	"chainmaker.org/chainmaker-go/tools/scanner/config"
	"github.com/stretchr/testify/assert"
)

func TestHandle(t *testing.T) {
	scanConfig, err := config.LoadScanConfig("../config.yml")
	assert.Nil(t, err)
	assert.NotNil(t, scanConfig)

	logScanner, err := NewLogScanner(scanConfig.FileConfigs[0])
	assert.Nil(t, err)
	logScanner.(*logScannerImpl).handle(scanConfig.FileConfigs[0].RoleConfigs[0], "2020-12-09 15:36:50.028	[INFO]	[Blockchain]	blockchain/chainmaker_server.go:125	[Core] blockchain [chain1] start success")
}
