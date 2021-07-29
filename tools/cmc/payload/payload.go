/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package payload

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	chainId       string
	contractName  string
	version       string
	runtime       string
	method        string
	kvPairs       string
	sequence      int
	byteCodePath  string
	orgId         string
	adminKeyPath  string
	adminCertPath string
	sdkConfPath   string
)

func NewPayloadCMD() *cobra.Command {
	payloadCmd := &cobra.Command{
		Use:   "payload",
		Short: "Payload command",
		Long:  "Payload command",
	}

	payloadCmd.AddCommand(jsonCMD())
	payloadCmd.AddCommand(createCMD())
	payloadCmd.AddCommand(signCMD())
	//payloadCmd.AddCommand(mergeCMD())

	return payloadCmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVarP(&chainId, "chain-id", "c", "chain1", "specify chain id")
	flags.StringVarP(&contractName, "contract-name", "n", "contract", "specify contract name")
	flags.StringVarP(&version, "version", "v", "version", "specify contract version")
	flags.StringVarP(&runtime, "runtime", "r", "WASMER_RUST", "specify contract runtime")
	flags.StringVarP(&method, "method", "m", "init", "specify method")
	flags.StringVarP(&kvPairs, "kv-pairs", "k", "tx_scheduler_timeout:15;tx_scheduler_validate_timeout:20",
		"specify key value pairs")
	flags.IntVarP(&sequence, "sequence", "s", 1, "specify sequence")
	flags.StringVarP(&byteCodePath, "byte-code-path", "p", "./fact.wasm", "specify byte code path")
	flags.StringVar(&sdkConfPath, "sdk-conf-path", "", "specify sdk config path")

}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}
