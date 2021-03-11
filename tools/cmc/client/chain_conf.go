/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"github.com/spf13/cobra"
)

func chainConfigCMD() *cobra.Command {
	chainConfigCmd := &cobra.Command{
		Use:   "chainconfig",
		Short: "chain config command",
		Long:  "chain config command",
	}
	chainConfigCmd.AddCommand(queryChainConfigCMD())
	chainConfigCmd.AddCommand(updateBlockConfigCMD())
	chainConfigCmd.AddCommand(configTrustRootCMD())
	chainConfigCmd.AddCommand(configConsensueNodeCMD())
	return chainConfigCmd
}

func queryChainConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "query chain config",
		Long:  "query chain config",
		RunE: func(_ *cobra.Command, _ []string) error {
			return queryChainConfig()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func queryChainConfig() error {
	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetChainConfig()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}
	fmt.Printf("Query chain config resp:\n %+v \n", chainConfig)
	return nil
}
