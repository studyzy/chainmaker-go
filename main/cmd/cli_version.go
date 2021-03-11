/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"chainmaker.org/chainmaker-go/localconf"
	"fmt"
	"github.com/common-nighthawk/go-figure"

	"github.com/spf13/cobra"
)

func VersionCMD() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show ChainMaker version",
		Long:  "Show ChainMaker version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			PrintVersion()
			return nil
		},
	}
}

func logo() string {
	fig := figure.NewFigure("ChainMaker", "slant", true)
	s := fig.String()
	fragment := "================================================================================="
	versionInfo := "::ChainMaker::  version(" + localconf.CurrentVersion + ")"
	return fmt.Sprintf("\n%s\n%s%s\n%s\n", fragment, s, fragment, versionInfo)
}

func PrintVersion() {
	//fmt.Printf("ChainMaker version: %s\n", CurrentVersion)
	fmt.Println(logo())
	fmt.Println()
}
