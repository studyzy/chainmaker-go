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
		Use:   "trustroot",
		Short: "trust member command",
		Long:  "trust member command",
	}
	cmd.AddCommand(addTrustMemberCMD())
	cmd.AddCommand(removeTrustMemberCMD())
	cmd.AddCommand(updateTrustMemberCMD())

	return cmd
}

func addTrustMemberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add trust member ca cert",
		Long:  "add trust member ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustMember(addTrustMember)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustMemberCrtPath, flagTrustMemberOrgId, flagTrustMemberRole, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustMemberOrgId)
	cmd.MarkFlagRequired(flagTrustMemberCrtPath)
	cmd.MarkFlagRequired(flagTrustMemberRole)
	cmd.MarkFlagRequired(flagNodeId)
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustMemberCrtPath, flagTrustMemberOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustMemberOrgId)
	cmd.MarkFlagRequired(flagTrustMemberCrtPath)

	return cmd
}

func updateTrustMemberCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "update trust member ca cert",
		Long:  "update trust member ca cert",
		RunE: func(_ *cobra.Command, _ []string) error {
			return configTrustMember(updateTrustMember)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustMemberCrtPath, flagTrustMemberOrgId, flagTrustMemberRole, flagNodeId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustMemberOrgId)
	cmd.MarkFlagRequired(flagTrustMemberCrtPath)
	cmd.MarkFlagRequired(flagTrustMemberRole)
	cmd.MarkFlagRequired(flagNodeId)
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
	if op == addTrustMember || op == updateTrustMember {

		if flagTrustMemberCrtPath == "" {
			return fmt.Errorf("please specify trust member path")
		}
		trustMemberBytes, err = ioutil.ReadFile(flagTrustMemberCrtPath)
		if err != nil {
			return err
		}
	}

	var payloadBytes []byte
	switch op {
	case addTrustMember:
		payloadBytes, err = client.CreateChainConfigTrustMemberAddPayload(trustMemberOrgId, trustMemberNodeId, trustMemberRole, string(trustMemberBytes))
	case removeTrustMember:
		payloadBytes, err = client.CreateChainConfigTrustMemberDeletePayload(trustMemberOrgId)
	case updateTrustMember:
		payloadBytes, err = client.CreateChainConfigTrustMemberUpdatePayload(trustMemberOrgId, trustMemberNodeId, trustMemberRole, string(trustMemberBytes))
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
