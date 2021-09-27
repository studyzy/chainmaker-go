/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"fmt"

	"chainmaker.org/chainmaker/localconf/v2"
	"github.com/spf13/cobra"
)

func ConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show chainmaker config",
		Long:  "Show chainmaker config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			initLocalConfig(cmd)
			return showConfig()
		},
	}
	attachFlags(cmd, []string{flagNameOfConfigFilepath})
	return cmd
}

func showConfig() error {
	json, err := localconf.ChainMakerConfig.PrettyJson()
	if err != nil {
		return err
	}

	fmt.Println(json)
	return nil
}
