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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/common"
	sdk "chainmaker.org/chainmaker/sdk-go"
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
	uploadCaCertCmd.MarkFlagRequired("admin-crt-file-paths")

	return uploadCaCertCmd
}

func cliUploadCaCert() error {
	adminKeys, adminCrts, err := createMultiSignAdmins(adminKeyFilePaths, adminCrtFilePaths)
	if err != nil {
		return err
	}

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
