/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/spf13/cobra"
)

const (
	addTrustMember = iota
	removeTrustMember
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustMemberCrtPath, flagTrustMemberOrgId,
		flagTrustMemberRole, flagTrustMemberNodeId, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
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
		flagAdminCrtFilePaths, flagAdminKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustMemberCrtPath)

	return cmd
}

func configTrustMember(op int) error {
	var adminKeys, adminCrts []string

	if adminKeyFilePaths != "" {
		adminKeys = strings.Split(adminKeyFilePaths, ",")
	}
	if adminCrtFilePaths != "" {
		adminCrts = strings.Split(adminCrtFilePaths, ",")
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

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

	var payload *common.Payload
	switch op {
	case addTrustMember:
		payload, err = client.CreateChainConfigTrustMemberAddPayload(trustMemberOrgId, trustMemberNodeId,
			trustMemberRole, string(trustMemberBytes))
	case removeTrustMember:
		payload, err = client.CreateChainConfigTrustMemberDeletePayload(string(trustMemberBytes))
	default:
		err = fmt.Errorf("invalid trust member operation")
	}
	if err != nil {
		return err
	}
	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}

		endorsementEntrys[i] = e
	}

	resp, err := client.SendChainConfigUpdateRequest(payload, endorsementEntrys, -1, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}

	fmt.Printf("add or remove request response %+v\n", resp)
	return nil
}
