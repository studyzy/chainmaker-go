/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tee

import (
	sdk "chainmaker.org/chainmaker-sdk-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	caCertFile string
	reportFile string
	enclaveId  string
)

var (
	sdkConfPath        string
	clientKeyFilePaths string
	clientCrtFilePaths string
	orgId              string
	chainId            string
)

var (
	teeFlags *pflag.FlagSet
)

func NewTeeCMD() *cobra.Command {
	teeCmd := &cobra.Command{
		Use:   "tee",
		Short: "trust execute environment command.",
		Long:  "trust execute environment command.",
	}

	teeFlags = &pflag.FlagSet{}
	teeFlags.StringVar(&sdkConfPath, "sdk-conf-path", "", "specify sdk config path")
	teeFlags.StringVar(&clientKeyFilePaths, "client-key-file-paths", "", "specify client key file paths, use ',' to separate")
	teeFlags.StringVar(&clientCrtFilePaths, "client-crt-file-paths", "", "specify client cert file paths, use ',' to separate")
	teeFlags.StringVar(&orgId, "org-id", "", "specify the orgId, such as wx-org1.chainmaker.com")
	teeFlags.StringVar(&chainId, "chain-id", "", "specify the chain id, such as: chain1, chain2 etc.")

	teeCmd.AddCommand(uploadCaCertCmd())
	teeCmd.AddCommand(uploadReportCmd())

	return teeCmd
}

func createClientWithConfig() (*sdk.ChainClient, error) {
	chainClient, err := sdk.NewChainClient(sdk.WithConfPath(sdkConfPath), sdk.WithUserKeyFilePath(clientKeyFilePaths),
		sdk.WithUserCrtFilePath(clientCrtFilePaths), sdk.WithChainClientOrgId(orgId), sdk.WithChainClientChainId(chainId))
	if err != nil {
		return nil, err
	}

	return chainClient, nil
}
