/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tee

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

var (
	caCertFile string
	reportFile string
)

var (
	sdkConfPath        string
	clientKeyFilePaths string
	clientCrtFilePaths string
	orgId              string
	chainId            string
	adminKeyFilePaths  string
	adminCrtFilePaths  string
	adminOrgIds        string
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
	teeFlags.StringVar(&sdkConfPath, "sdk-conf-path", "",
		"specify sdk config path")
	teeFlags.StringVar(&clientKeyFilePaths, "client-key-file-paths", "",
		"specify client key file paths, use ',' to separate")
	teeFlags.StringVar(&clientCrtFilePaths, "client-crt-file-paths", "",
		"specify client cert file paths, use ',' to separate")
	teeFlags.StringVar(&orgId, "org-id", "",
		"specify the orgId, such as wx-org1.chainmaker.com")
	teeFlags.StringVar(&chainId, "chain-id", "",
		"specify the chain id, such as: chain1, chain2 etc.")
	teeFlags.StringVar(&adminKeyFilePaths, "admin-key-file-paths", "",
		"specify admin key file paths, use ',' to separate")
	teeFlags.StringVar(&adminCrtFilePaths, "admin-crt-file-paths", "",
		"specify admin cert file paths, use ',' to separate")
	teeFlags.StringVar(&adminOrgIds, "admin-org-ids", "",
		"specify admin org-ids, use ',' to separate")

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

func createMultiSignAdmins(adminKeyFilePaths string, adminCrtFilePaths string) ([]string, []string, error) {
	var adminKeys, adminCrts []string

	if adminKeyFilePaths != "" {
		adminKeys = strings.Split(adminKeyFilePaths, ",")
	}
	if adminCrtFilePaths != "" {
		adminCrts = strings.Split(adminCrtFilePaths, ",")
	}
	if len(adminKeys) != len(adminCrts) {
		return nil, nil, fmt.Errorf("admin keys num(%v) is not equals certs num(%v)", len(adminKeys), len(adminCrts))
	}

	return adminKeys, adminCrts, nil
}

func createMultiSignAdminsForPK(adminKeyFilePaths string, adminOrgIds string) ([]string, []string, error) {
	var adminKeys, adminOrgs []string

	if adminKeyFilePaths != "" {
		adminKeys = strings.Split(adminKeyFilePaths, ",")
	}
	if adminOrgIds != "" {
		adminOrgs = strings.Split(adminOrgIds, ",")
	}
	if len(adminKeys) != len(adminOrgs) {
		return nil, nil, fmt.Errorf("admin keys num(%v) is not equals org-id num(%v)", len(adminKeys), len(adminOrgs))
	}

	return adminKeys, adminOrgs, nil
}
