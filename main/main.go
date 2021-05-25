/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/main/cmd"
	"fmt"
	"github.com/spf13/cobra"
)

// ./chainmaker start -c ../config/wx-org1/chainmaker.yml
func main() {
	mainCmd := &cobra.Command{Use: "chainmaker"}
	mainCmd.AddCommand(cmd.StartCMD())
	mainCmd.AddCommand(cmd.VersionCMD())
	mainCmd.AddCommand(cmd.ConfigCMD())

	err := mainCmd.Execute()
	if err != nil {
		fmt.Println(err)
	}

	return
}
