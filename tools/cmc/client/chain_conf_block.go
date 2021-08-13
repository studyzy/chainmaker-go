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

func updateBlockConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "update block command",
		Long:  "update block command",
	}
	cmd.AddCommand(updateBlockIntervalCMD())

	return cmd
}

func updateBlockIntervalCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updateblockinterval",
		Short: "update block interval",
		Long:  "update block interval",
		RunE: func(_ *cobra.Command, _ []string) error {
			return updateBlockInterval()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
		flagBlockInterval,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagBlockInterval)

	return cmd
}

func updateBlockInterval() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	chainConfig, err := client.GetChainConfig()
	if err != nil {
		return fmt.Errorf("get chain config failed, %s", err.Error())
	}
	txTimestampVerify := chainConfig.Block.TxTimestampVerify
	txTimeout := int(chainConfig.Block.TxTimeout)
	blockTxCap := int(chainConfig.Block.BlockTxCapacity)
	blockSize := int(chainConfig.Block.BlockSize)
	blockUpdatePayload, err := client.CreateChainConfigBlockUpdatePayload(txTimestampVerify, txTimeout, blockTxCap, blockSize, blockInterval)
	if err != nil {
		return fmt.Errorf("create chain config block update payload failed, %s", err.Error())
	}
	adminClient, err := createAdminWithConfig(adminKeyFilePaths, adminCrtFilePaths)
	if err != nil {
		return fmt.Errorf("create admin client failed, %s", err.Error())
	}
	defer adminClient.Stop()
	signedPayload, err := adminClient.SignChainConfigPayload(blockUpdatePayload)
	if err != nil {
		return fmt.Errorf("sign chain config payload failed, %s", err.Error())
	}
	mergedSignedPayload, err := client.MergeChainConfigSignedPayload([][]byte{signedPayload})
	if err != nil {
		return fmt.Errorf("merge chain config signed payload failed, %s", err.Error())
	}
	resp, err := client.SendChainConfigUpdateRequest(mergedSignedPayload, -1, true)
	if err != nil {
		return fmt.Errorf("send chain config update request failed, %s", err.Error())
	}
	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return fmt.Errorf("check proposal request resp failed, %s", err.Error())
	}
	return nil
}
