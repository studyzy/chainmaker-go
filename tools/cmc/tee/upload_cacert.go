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

	return uploadCaCertCmd
}

func cliUploadCaCert() error {
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

	_, err = client.SaveEnclaveCACert(string(cacertData), "", true, 3)
	if err != nil {
		return fmt.Errorf("save ca cert failed, %s", err.Error())
	}

	fmt.Println("command execute successfully.")
	return nil
}
