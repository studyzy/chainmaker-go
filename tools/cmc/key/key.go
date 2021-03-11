/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package key

import (
	"strings"

	"chainmaker.org/chainmaker-go/common/cert"
	"chainmaker.org/chainmaker-go/common/crypto"
	"github.com/spf13/cobra"
)

var (
	algo string
	path string
	name string
)

func KeyCMD() *cobra.Command {
	keyCmd := &cobra.Command{
		Use:   "key",
		Short: "ChainMaker key command",
		Long:  "ChainMaker key command",
	}
	keyCmd.AddCommand(genCMD())
	return keyCmd
}

func genCMD() *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Private key generate",
		Long:  "Private key generate",
		RunE: func(_ *cobra.Command, _ []string) error {
			return generatePrivateKey()
		},
	}

	flags := genCmd.Flags()
	flags.StringVarP(&algo, "algo", "a", "", "specify key generate algorithm")
	flags.StringVarP(&path, "path", "p", "", "specify storage path")
	flags.StringVarP(&name, "name", "n", "", "specify storage name")

	return genCmd
}

func generatePrivateKey() error {
	keyType := crypto.AsymAlgoMap[strings.ToUpper(algo)]
	_, err := cert.CreatePrivKey(keyType, path, name)
	return err
}
