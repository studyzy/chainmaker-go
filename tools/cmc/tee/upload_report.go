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
	"chainmaker.org/chainmaker/pb-go/common"
	sdk "chainmaker.org/chainmaker/sdk-go"

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
	uploadReportCmd.MarkFlagRequired("admin-crt-file-paths")

	return uploadReportCmd
}

func cliUploadReport() error {
	adminKeys, adminCrts, err := createMultiSignAdmins(adminKeyFilePaths, adminCrtFilePaths)
	if err != nil {
		return err
	}

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
		e, err := sdk.SignPayloadWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}
		endorsementEntrys[i] = e
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
