// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"chainmaker.org/chainmaker-go/common/log"
	sdk "chainmaker.org/chainmaker-sdk-go"
)

// CreateChainClientWithSDKConf create a chain client with sdk config file path
func CreateChainClientWithSDKConf(sdkConfPath, chainId string) (*sdk.ChainClient, error) {
	logger, _ := log.InitSugarLogger(&log.LogConfig{
		Module:       "[SDK]",
		LogPath:      "./sdk.log",
		LogLevel:     log.LEVEL_ERROR,
		MaxAge:       30,
		JsonFormat:   false,
		ShowLine:     true,
		LogInConsole: true,
	})

	var (
		cc  *sdk.ChainClient
		err error
	)

	if chainId != "" {
		cc, err = sdk.NewChainClient(
			sdk.WithConfPath(sdkConfPath),
			sdk.WithChainClientLogger(logger),
			sdk.WithChainClientChainId(chainId),
		)
	} else {
		cc, err = sdk.NewChainClient(
			sdk.WithConfPath(sdkConfPath),
			sdk.WithChainClientLogger(logger),
		)
	}
	if err != nil {
		return nil, err
	}

	// Enable certificate compression
	err = cc.EnableCertHash()
	if err != nil {
		return nil, err
	}
	return cc, nil
}
