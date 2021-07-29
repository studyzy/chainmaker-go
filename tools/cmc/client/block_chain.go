/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
)

func blockChainsCMD() *cobra.Command {
	chainConfigCmd := &cobra.Command{
		Use:   "blockchains",
		Short: "blockchains command",
		Long:  "blockchains command",
	}
	chainConfigCmd.AddCommand(checkNewBlockchainsCMD())
	return chainConfigCmd
}

func checkNewBlockchainsCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checknew",
		Short: "check new blockchains",
		Long:  "check new blockchains",
		RunE: func(_ *cobra.Command, _ []string) error {
			return checkNewBlockchains()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagChainId, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func checkNewBlockchains() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	err = client.CheckNewBlockChainConfig()
	if err != nil {
		return fmt.Errorf("check new blockchains failed, %s", err.Error())
	}
	fmt.Printf("check new blockchains ok \n")
	return nil
}
