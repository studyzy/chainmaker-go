/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localconf

import (
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/config"

	"github.com/stretchr/testify/assert"
)

func TestLoadConf(_ *testing.T) {
	fmt.Printf("system logger file path: %s\n", ChainMakerConfig.LogConfig.SystemLog.FilePath)
	fmt.Printf("brief logger file path: %s\n", ChainMakerConfig.LogConfig.BriefLog.FilePath)
	fmt.Printf("event logger file path: %s\n", ChainMakerConfig.LogConfig.EventLog.FilePath)
	fmt.Printf("net config provider : %s\n", ChainMakerConfig.NetConfig.Provider)
	fmt.Printf("rpc port: %d\n", ChainMakerConfig.RpcConfig.Port)
}

func TestUpdateDebugConfig(t *testing.T) {
	pairs := []*config.ConfigKeyValue{
		{Key: "IsCliOpen", Value: "true"},
		{Key: "IsHttpOpen", Value: "true"},
		{Key: "invalid", Value: "true"},
	}
	err := UpdateDebugConfig(pairs)
	assert.NoError(t, err)
	assert.True(t, ChainMakerConfig.DebugConfig.IsCliOpen)
	assert.True(t, ChainMakerConfig.DebugConfig.IsHttpOpen)
}
