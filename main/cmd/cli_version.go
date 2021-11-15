/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"fmt"

	"chainmaker.org/chainmaker-go/blockchain"
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
	//versionInfo := "::ChainMaker::  version(" + protocol.DefaultBlockVersion + ")"
	versionInfo := fmt.Sprintf("ChainMaker Version: %s\n", blockchain.CurrentVersion)

	if blockchain.BuildDateTime != "" {
		versionInfo += fmt.Sprintf("Build Time:%9s%s\n", " ", blockchain.BuildDateTime)
	}

	if blockchain.GitBranch != "" {
		versionInfo += fmt.Sprintf("Git Commit:%9s%s", " ", blockchain.GitBranch)
		if blockchain.GitCommit != "" {
			versionInfo += fmt.Sprintf("(%s)", blockchain.GitCommit)
		}
	}
	return fmt.Sprintf("\n%s\n%s%s\n%s\n", fragment, s, fragment, versionInfo)
}

func PrintVersion() {
	//fmt.Printf("ChainMaker version: %s\n", CurrentVersion)
	fmt.Println(logo())
	fmt.Println()
}
