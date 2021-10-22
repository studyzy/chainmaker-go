/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/pubkey"

	"chainmaker.org/chainmaker-go/tools/cmc/archive"
	"chainmaker.org/chainmaker-go/tools/cmc/bulletproofs"
	"chainmaker.org/chainmaker-go/tools/cmc/cert"
	"chainmaker.org/chainmaker-go/tools/cmc/client"
	"chainmaker.org/chainmaker-go/tools/cmc/console"
	"chainmaker.org/chainmaker-go/tools/cmc/hibe"
	"chainmaker.org/chainmaker-go/tools/cmc/key"
	"chainmaker.org/chainmaker-go/tools/cmc/paillier"
	"chainmaker.org/chainmaker-go/tools/cmc/payload"
	"chainmaker.org/chainmaker-go/tools/cmc/query"
	"chainmaker.org/chainmaker-go/tools/cmc/tee"
	"github.com/spf13/cobra"
)

func main() {
	mainCmd := &cobra.Command{
		Use:   "cmc",
		Short: "ChainMaker CLI",
		Long: strings.TrimSpace(`Command line interface for interacting with ChainMaker daemon.
For detailed logs, please see ./sdk.log
`),
	}

	mainCmd.AddCommand(key.KeyCMD())
	mainCmd.AddCommand(cert.CertCMD())
	mainCmd.AddCommand(client.ClientCMD())
	mainCmd.AddCommand(hibe.HibeCMD())
	mainCmd.AddCommand(paillier.PaillierCMD())
	mainCmd.AddCommand(archive.NewArchiveCMD())
	mainCmd.AddCommand(query.NewQueryOnChainCMD())
	mainCmd.AddCommand(payload.NewPayloadCMD())
	mainCmd.AddCommand(console.NewConsoleCMD(mainCmd))
	mainCmd.AddCommand(bulletproofs.BulletproofsCMD())
	mainCmd.AddCommand(tee.NewTeeCMD())
	mainCmd.AddCommand(pubkey.NewPubkeyCMD())

	// 后续改成go-sdk
	//mainCmd.AddCommand(payload.PayloadCMD())
	//mainCmd.AddCommand(log.LogCMD())

	mainCmd.Execute()
}
