/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/common"
	sdk "chainmaker.org/chainmaker/sdk-go"
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
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
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
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
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
		flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func configTrustRoot(op int) error {
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 {
		return errAdminOrgIdKeyCertIsEmpty
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

	var trustRootBytes []byte
	if op == addTrustRoot || op == updateTrustRoot {
		if trustRootPath == "" {
			return fmt.Errorf("please specify trust root path")
		}
		trustRootBytes, err = ioutil.ReadFile(trustRootPath)
		if err != nil {
			return err
		}
	}

	var payload *common.Payload
	switch op {
	case addTrustRoot:
		payload, err = client.CreateChainConfigTrustRootAddPayload(trustRootOrgId, string(trustRootBytes))
	case removeTrustRoot:
		payload, err = client.CreateChainConfigTrustRootDeletePayload(trustRootOrgId)
	case updateTrustRoot:
		payload, err = client.CreateChainConfigTrustRootUpdatePayload(trustRootOrgId, string(trustRootBytes))
	default:
		err = errors.New("invalid trust root operation")
	}
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdk.SignPayloadWithPath(adminKeys[i], adminCrts[i], payload)
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
	fmt.Printf("trustroot response %+v\n", resp)
	return nil
}
