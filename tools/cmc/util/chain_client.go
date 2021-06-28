// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	sdk "chainmaker.org/chainmaker-sdk-go"
)

// CreateChainClientWithSDKConf create a chain client with sdk config file path
func CreateChainClientWithSDKConf(sdkConfPath, chainId string) (*sdk.ChainClient, error) {
	var (
		cc  *sdk.ChainClient
		err error
	)

	if chainId != "" {
		cc, err = sdk.NewChainClient(
			sdk.WithConfPath(sdkConfPath),
			sdk.WithChainClientChainId(chainId),
		)
	} else {
		cc, err = sdk.NewChainClient(
			sdk.WithConfPath(sdkConfPath),
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

func AttachAndRequiredFlags(cmd *cobra.Command, flags *pflag.FlagSet, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
		cmd.MarkFlagRequired(name)
	}
}

func AttachFlags(cmd *cobra.Command, flags *pflag.FlagSet, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}
