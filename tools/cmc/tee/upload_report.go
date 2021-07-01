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

	return uploadReportCmd
}

func cliUploadReport() error {
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

	_, err = client.SaveEnclaveReport("global_enclave_id", reportData, "", true, 3)
	if err != nil {
		return fmt.Errorf("save ca cert failed, %s", err.Error())
	}

	fmt.Println("command execute successfully.")
	return nil
}
