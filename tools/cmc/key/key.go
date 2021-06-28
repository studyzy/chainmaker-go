/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package key

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/common/cert"
	"chainmaker.org/chainmaker-go/common/crypto"
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
		Long: strings.TrimSpace(
			fmt.Sprintf(`Generate the private key of the specified crypto algorithm.
Supported algorithms: RSA512 RSA1024 RSA2048 RSA3072 SM2 ECC_P256 ECC_P384 ECC_P521 ECC_Secp256k1
Example:
$ cmc key gen -a ECC_P256 -p ./ -n ca.key
`,
			),
		),
		RunE: func(_ *cobra.Command, _ []string) error {
			return generatePrivateKey()
		},
	}

	flags := genCmd.Flags()
	flags.StringVarP(&algo, "algo", "a", "", "specify key generate algorithm. eg. RSA512,RSA1024,RSA2048,RSA3072,SM2,ECC_P256,ECC_P384,ECC_P521,ECC_Secp256k1")
	flags.StringVarP(&path, "path", "p", "", "specify storage path")
	flags.StringVarP(&name, "name", "n", "", "specify storage file name")

	return genCmd
}

func generatePrivateKey() error {
	if keyType, ok := crypto.AsymAlgoMap[algo]; ok {
		_, err := cert.CreatePrivKey(keyType, path, name)
		return err
	}

	if keyType, ok := crypto.AsymAlgoMap[strings.ToUpper(algo)]; ok {
		_, err := cert.CreatePrivKey(keyType, path, name)
		return err
	}
	return fmt.Errorf("unsupported algorithm %s", algo)
}
