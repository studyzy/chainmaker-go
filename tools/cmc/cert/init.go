/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cert

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	algo             string
	path             string
	name             string
	keyPath          string
	hash             string
	org              string
	cn               string
	ou               string
	isCA             bool
	caKeyPath        string
	caCertPath       string
	csrPath          string
	crlPath          string
	crtPath          string
	sdkConfPath      string
	nodeCertPath     string
	nodePkPath       string
	pubkeyOrCertPath string
)

const (
	flagCaKeyPath        = "ca-key-path"
	flagCaCertPath       = "ca-cert-path"
	flagCrlPath          = "crl-path"
	flagCrtPath          = "crt-path"
	flagSdkConfPath      = "sdk-conf-path"
	flagNodeCertPath     = "node-cert-path"
	flagNodePkPath       = "node-pk-path"
	flagCertOrPubkeyPath = "pubkey-cert-path"
)

var requiredFlags = map[string]bool{
	"path": true,
	"name": true,
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVarP(&algo, "algo", "a", "", "specify key generate algorithm")
	flags.StringVarP(&path, "path", "p", "", "specify storage path")
	flags.StringVarP(&name, "name", "n", "", "specify storage name")
	flags.StringVarP(&keyPath, "key-path", "k", "", "specify key path")
	flags.StringVarP(&hash, "hash", "H", "", "specify hash algorithm")
	flags.StringVarP(&org, "org", "o", "", "specify organization")
	flags.StringVarP(&cn, "cn", "c", "", "specify common name")
	flags.StringVarP(&ou, "ou", "O", "", "specify organizational unit")
	flags.BoolVar(&isCA, "is-ca", false, "specify is certificate authority")
	flags.StringVarP(&caKeyPath, flagCaKeyPath, "K", "", "specify certificate authority key path")
	flags.StringVarP(&caCertPath, flagCaCertPath, "C", "", "specify certificate authority certificate path")
	flags.StringVarP(&csrPath, "csr-path", "r", "", "specify certificate request path")
	flags.StringVar(&crlPath, flagCrlPath, "", "specify crl file path")
	flags.StringVar(&crtPath, flagCrtPath, "", "specify crt file path")
	flags.StringVar(&nodeCertPath, flagNodeCertPath, "", "specify node cert path")
	flags.StringVar(&nodePkPath, flagNodePkPath, "", "specify node cert path")
	flags.StringVar(&pubkeyOrCertPath, flagCertOrPubkeyPath, "", "specify user pubkey path or cert path")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk_conf path")
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
			if requiredFlags[name] {
				cmd.MarkFlagRequired(name)
			}
		} else {
			fmt.Printf("Could not find flag '%s' to attach to command '%s'", name, cmd.Name())
		}
	}
}
