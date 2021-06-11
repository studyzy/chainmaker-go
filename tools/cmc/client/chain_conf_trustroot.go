/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
)

const (
	addTrustRoot = iota
	removeTrustRoot
	updateTrustRoot
)

func configTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trustroot",
		Short: "trust root command",
		Long:  "trust root command",
	}
	cmd.AddCommand(addTrustRootCMD())
	cmd.AddCommand(removeTrustRootCMD())
	cmd.AddCommand(updateTrustRootCMD())

	return cmd
}

func addTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add trust root ca cert",
		Long:  "add trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustRoot(addTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func removeTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove trust root ca cert",
		Long:  "remove trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustRoot(removeTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func updateTrustRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update trust root ca cert",
		Long:  "update trust root ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustRoot(updateTrustRoot)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func configTrustRoot(op int) error {
	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}

	adminClient, err := createAdminWithConfig(adminKeyFilePaths, adminCrtFilePaths)
	if err != nil {
		return fmt.Errorf("create admin client failed, %s", err.Error())
	}
	defer adminClient.Stop()

	var trustRootBytes []string
	if op == addTrustRoot || op == updateTrustRoot {

		if len(trustRootPaths) == 0 {
			return fmt.Errorf("please specify trust root path")
		}
		for _,trustRootPath := range trustRootPaths{
			trustRoot, err := ioutil.ReadFile(trustRootPath)
			if err != nil {
				return err
			}
			trustRootBytes = append(trustRootBytes,string(trustRoot))
		}
	}

	var payloadBytes []byte
	switch op {
	case addTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootAddPayload(trustRootOrgId, []string(trustRootBytes))
	case removeTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootDeletePayload(trustRootOrgId)
	case updateTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootUpdatePayload(trustRootOrgId, []string(trustRootBytes))
	default:
		err = fmt.Errorf("invalid trust root operation")
	}
	if err != nil {
		return err
	}
	signedPayload, err := adminClient.SignChainConfigPayload(payloadBytes)
	if err != nil {
		return err
	}
	mergeSignedPayloadBytes, err := client.MergeChainConfigSignedPayload([][]byte{signedPayload})
	if err != nil {
		return err
	}
	resp, err := client.SendChainConfigUpdateRequest(mergeSignedPayloadBytes)
	if err != nil {
		return err
	}
	fmt.Printf("add or remove request response %+v\n", resp)

	return nil
}
