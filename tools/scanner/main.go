/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	"chainmaker.org/chainmaker-go/tools/scanner/cmd"
	"github.com/spf13/cobra"
)

func main() {
	mainCmd := &cobra.Command{
		Use:   "scanner",
		Short: "Log scanner tool",
		Long:  "Log scanner tool",
	}

	mainCmd.AddCommand(cmd.StartCMD())
	mainCmd.AddCommand(cmd.ConfigCMD())

	err := mainCmd.Execute()
	if err != nil {
		fmt.Println(err.Error())
	}
}
