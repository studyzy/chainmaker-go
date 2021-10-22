/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tee

import (
	"encoding/hex"
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

func uploadReportCmd() *cobra.Command {
	uploadReportCmd := &cobra.Command{
		Use:   "upload_report",
		Long:  "upload report of trust execute environment.",
		Short: "upload report of trust execute environment.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return cliUploadReport()
		},
	}

	flags := &pflag.FlagSet{}
	flags.StringVar(&reportFile, "report", "", "specify report filename")

	uploadReportCmd.Flags().AddFlagSet(teeFlags)
	uploadReportCmd.Flags().AddFlagSet(flags)
	uploadReportCmd.MarkFlagRequired("report")
	uploadReportCmd.MarkFlagRequired("sdk-conf-path")
	uploadReportCmd.MarkFlagRequired("admin-key-file-paths")

	return uploadReportCmd
}

func cliUploadReport() error {
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string

	file, err := os.Open(reportFile)
	if err != nil {
		return fmt.Errorf("open file '%s' error: %v", reportFile, err)
	}
	defer file.Close()

	reportBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read file '%v' error: %v", reportFile, err)
	}
	reportData := hex.EncodeToString(reportBytes)

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

	payload, err := client.CreateSaveEnclaveReportPayload("global_enclave_id", reportData, "")
	if err != nil {
		return fmt.Errorf("save enclave report failed, %s", err.Error())
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
