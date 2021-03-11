/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

func ConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show config command",
		Long:  "Show config command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig()
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "../tools/scanner/config.yml", "specify config file path")

	return cmd
}

func showConfig() error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
