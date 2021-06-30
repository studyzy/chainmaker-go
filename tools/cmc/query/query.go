// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package query

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// sdk config file path
	sdkConfPath string

	chainId string
)

const (
	// TODO: wrap common flags to a separate package?
	//// Common flags
	// sdk config file path flag
	flagSdkConfPath = "sdk-conf-path"
	flagChainId     = "chain-id"
)

func NewQueryOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "query on-chain blockchain data",
		Long:  "query on-chain blockchain data",
	}

	cmd.AddCommand(newQueryTxOnChainCMD())
	cmd.AddCommand(newQueryBlockByHeightOnChainCMD())
	cmd.AddCommand(newQueryBlockByHashOnChainCMD())
	cmd.AddCommand(newQueryBlockByTxIdOnChainCMD())
	cmd.AddCommand(newQueryArchivedHeightOnChainCMD())

	return cmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
}
