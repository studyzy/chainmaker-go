/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/archive"
	"chainmaker.org/chainmaker-go/tools/cmc/query"
)

func main() {
	mainCmd := &cobra.Command{
		Use:   "cmc",
		Short: "ChainMaker CLI",
		Long: strings.TrimSpace(`Command line interface for interacting with ChainMaker daemon.
For detailed logs, please see ./sdk.log
`),
	}

	//mainCmd.AddCommand(key.KeyCMD())
	//mainCmd.AddCommand(cert.CertCMD())
	//mainCmd.AddCommand(client.ClientCMD())
	//mainCmd.AddCommand(hibe.HibeCMD())
	//mainCmd.AddCommand(paillier.PaillierCMD())
	mainCmd.AddCommand(archive.NewArchiveCMD())
	mainCmd.AddCommand(query.NewQueryOnChainCMD())
	//mainCmd.AddCommand(console.NewConsoleCMD(mainCmd))
	//mainCmd.AddCommand(bulletproofs.BulletproofsCMD())
	//mainCmd.AddCommand(tee.NewTeeCMD())

	// 后续改成go-sdk
	//mainCmd.AddCommand(payload.PayloadCMD())
	//mainCmd.AddCommand(log.LogCMD())

	mainCmd.Execute()
}
