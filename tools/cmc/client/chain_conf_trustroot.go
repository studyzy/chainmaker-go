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

	"chainmaker.org/chainmaker/pb-go/common"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
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
		flagSdkConfPath, flagOrgId, flagEnableCertHash, flagTrustRootCrtPath, flagTrustRootOrgId, flagAdminOrgIds,
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagAdminOrgIds)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagTrustRootOrgId)
	cmd.MarkFlagRequired(flagTrustRootCrtPath)

	return cmd
}

func configTrustRoot(op int) error {
	adminOrgIdSlice := strings.Split(adminOrgIds, ",")
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 || len(adminOrgIdSlice) == 0 || len(adminKeys) != len(adminCrts) || len(adminOrgIdSlice) != len(adminCrts) {
		return fmt.Errorf(ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil && !strings.Contains(err.Error(), "user cert havenot on chain yet, and try again") {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err)
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

	var payloadBytes *common.Payload
	switch op {
	case addTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootAddPayload(trustRootOrgId, string(trustRootBytes))
	case removeTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootDeletePayload(trustRootOrgId)
	case updateTrustRoot:
		payloadBytes, err = client.CreateChainConfigTrustRootUpdatePayload(trustRootOrgId, string(trustRootBytes))
	default:
		err = errors.New("invalid trust root operation")
	}
	if err != nil {
		return err
	}

	signedPayloads := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		_, privKey, err := dealUserKey(adminKeys[i])
		if err != nil {
			return err
		}
		crtBytes, crt, err := dealUserCrt(adminCrts[i])
		if err != nil {
			return err
		}

		signedPayload, err := signChainConfigPayload(payloadBytes, crtBytes, privKey, crt, adminOrgIdSlice[i])
		if err != nil {
			return err
		}
		signedPayloads[i] = signedPayload
	}

	//mergedSignedPayloadBytes, err := client.MergeChainConfigSignedPayload(signedPayloads)
	//if err != nil {
	//	return err
	//}

	resp, err := client.SendChainConfigUpdateRequest(payloadBytes, signedPayloads, 0, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return err
	}
	fmt.Printf("trustroot response %+v\n", resp)
	return nil
}
