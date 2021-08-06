/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

const (
	addTrustMember = iota
	removeTrustMember
	updateTrustMember
)

func configTrustMemberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trustmember",
		Short: "trust member command",
		Long:  "trust member command",
	}
	cmd.AddCommand(addTrustMemberCMD())
	cmd.AddCommand(removeTrustMemberCMD())

	return cmd
}

func addTrustMemberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add trust member",
		Long:  "add trust member",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustMember(addTrustMember)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustMemberCrtPath, flagTrustMemberOrgId, flagTrustMemberRole, flagTrustMemberNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustMemberOrgId)
	cmd.MarkFlagRequired(flagTrustMemberCrtPath)
	cmd.MarkFlagRequired(flagTrustMemberRole)
	cmd.MarkFlagRequired(flagTrustMemberNodeId)
	return cmd
}

func removeTrustMemberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove trust member ca cert",
		Long:  "remove trust member ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustMember(removeTrustMember)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustMemberCrtPath,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustMemberCrtPath)

	return cmd
}

func configTrustMember(op int) error {
	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}

	adminClient, err := createAdminWithConfig(adminKeyFilePaths, adminCrtFilePaths)
	if err != nil {
		return fmt.Errorf("create admin client failed, %s", err.Error())
	}
	defer adminClient.Stop()

	var trustMemberBytes []byte
	if op == addTrustMember || op == removeTrustMember {

		if flagTrustMemberCrtPath == "" {
			return fmt.Errorf("please specify trust member path")
		}
		trustMemberBytes, err = ioutil.ReadFile(trustMemberInfoPath)
		if err != nil {
			return err
		}
	}

	var payloadBytes []byte
	switch op {
	case addTrustMember:
		payloadBytes, err = client.CreateChainConfigTrustMemberAddPayload(trustMemberOrgId, trustMemberNodeId, trustMemberRole, string(trustMemberBytes))
	case removeTrustMember:
		payloadBytes, err = client.CreateChainConfigTrustMemberDeletePayload(string(trustMemberBytes))
	default:
		err = fmt.Errorf("invalid trust member operation")
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
