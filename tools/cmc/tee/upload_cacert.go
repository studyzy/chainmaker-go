/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tee

import (
	"fmt"
	"io/ioutil"
	"os"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func uploadCaCertCmd() *cobra.Command {
	uploadCaCertCmd := &cobra.Command{
		Use:   "upload_ca_cert",
		Long:  "upload ca_cert of trust execute environment.",
		Short: "upload ca_cert of trust execute environment.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return cliUploadCaCert()
		},
	}

	flags := &pflag.FlagSet{}
	flags.StringVar(&caCertFile, "ca_cert", "", "specify ca_cert filename")

	uploadCaCertCmd.Flags().AddFlagSet(teeFlags)
	uploadCaCertCmd.Flags().AddFlagSet(flags)
	uploadCaCertCmd.MarkFlagRequired("ca_cert")
	uploadCaCertCmd.MarkFlagRequired("sdk-conf-path")
	uploadCaCertCmd.MarkFlagRequired("admin-key-file-paths")

	return uploadCaCertCmd
}

func cliUploadCaCert() error {
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string

	file, err := os.Open(caCertFile)
	if err != nil {
		return fmt.Errorf("open file '%s' error: %v", reportFile, err)
	}
	defer file.Close()

	cacertData, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read file '%v' error: %v", reportFile, err)
	}

	client, err := createClientWithConfig()
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()

	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		adminKeys, adminCrts, err = createMultiSignAdmins(adminKeyFilePaths, adminCrtFilePaths)
		if err != nil {
			return err
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		adminKeys, adminOrgs, err = createMultiSignAdminsForPK(adminKeyFilePaths, adminOrgIds)
		if err != nil {
			return err
		}
	}

	err = client.CheckNewBlockChainConfig()
	if err != nil {
		return fmt.Errorf("check new blockchains failed, %s", err.Error())
	}

	payload, err := client.CreateSaveEnclaveCACertPayload(string(cacertData), "")
	if err != nil {
		return fmt.Errorf("save enclave ca cert failed, %s", err.Error())
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			e, err := sdkutils.MakePkEndorserWithPath(adminKeys[i], crypto.HashAlgoMap[client.GetHashType()], adminOrgs[i], payload)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		}
	}

	resp, err := client.SendContractManageRequest(payload, endorsementEntrys, 5, false)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}

	fmt.Println("command execute successfully.")
	return nil
}
