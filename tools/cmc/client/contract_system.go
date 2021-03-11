/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
)

const CREATE_USER_FAILED_FORMAT = "create user client failed, %s"

func systemContractCMD() *cobra.Command {
	systemContractCmd := &cobra.Command{
		Use:   "system",
		Short: "system contract command",
		Long:  "system contract command",
	}

	systemContractCmd.AddCommand(getChainInfoCMD())
	systemContractCmd.AddCommand(getBlockByHeightCMD())
	systemContractCmd.AddCommand(getTxByTxIdCMD())
	return systemContractCmd
}

func getChainInfoCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getchaininfo",
		Short: "get chain info",
		Long:  "get chain info",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainInfo()
		},
	}

	attachFlags(cmd, []string{
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagParams, flagTimeout, flagClientCrtFilePaths, flagClientKeyFilePaths, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func getBlockByHeightCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "get block by height",
		Long:  "get block by height",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getBlockByHeight()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagChainId, flagBlockHeight, flagWithRWSet,
		flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagBlockHeight)
	cmd.MarkFlagRequired(flagWithRWSet)

	return cmd
}

func getTxByTxIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "get tx by tx id",
		Long:  "get tx by tx id",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getTxByTxId()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagChainId, flagTxId,
		flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagTxId)

	return cmd
}

func getChainInfo() error {
	var (
		err error
	)

	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf(CREATE_USER_FAILED_FORMAT, err.Error())
	}
	defer client.Stop()
	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.GetChainInfo()
	if err != nil {
		return fmt.Errorf("get chain info failed, %s", err.Error())
	}

	fmt.Printf("get chain info resp: %+v\n", resp)

	return nil
}

func getBlockByHeight() error {
	var (
		err error
	)

	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf(CREATE_USER_FAILED_FORMAT, err.Error())
	}
	defer client.Stop()
	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.GetBlockByHeight(blockHeight, withRWSet)
	if err != nil {
		return fmt.Errorf("get block by height failed, %s", err.Error())
	}

	fmt.Printf("get block by height resp: %+v\n", resp)

	return nil
}

func getTxByTxId() error {
	var (
		err error
	)

	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf(CREATE_USER_FAILED_FORMAT, err.Error())
	}
	defer client.Stop()
	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.GetTxByTxId(txId)
	if err != nil {
		return fmt.Errorf("get block by height failed, %s", err.Error())
	}

	fmt.Printf("get block by height resp: %+v\n", resp)

	return nil
}
