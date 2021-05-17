/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package archive

import (
	sdk "chainmaker.org/chainmaker-sdk-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// sdk config file path
	sdkConfPath string

	chainId string

	// cert and key
	adminKeyFilePaths string
	adminCrtFilePaths string

	dbType            string
	dbDest            string
	targetBlockHeight uint64
	blockInterval     uint64
)

const (
	// TODO: wrap common flags to a separate package?
	//// Common flags
	// sdk config file path flag
	flagSdkConfPath    = "sdk-conf-path"
	flagOrgId          = "org-id"
	flagSyncResult     = "sync-result"
	flagEnableCertHash = "enable-cert-hash"
	flagChainId        = "chain-id"
	// Admin private key file & cert file paths flags, use ',' to separate
	// The key in the list corresponds to the cert one to one.
	// If there are only one pair, the single-sign mode will be used.
	// If there are multiple pairs, the multi-sign mode will be used,
	// with the first pair used to initiate the multi-sign request and the rest used for the multi-sign vote
	flagAdminKeyFilePaths = "admin-key-file-paths"
	flagAdminCrtFilePaths = "admin-crt-file-paths"

	//// Archive flags
	// Off-chain database type. eg. mysql,mongodb,pgsql
	flagDbType = "type"
	// Off-chain database destination. eg. user:password:localhost:port
	flagDbDest = "dest"
	// Archive target block height, stop archiving (include this block) after reaching this height.
	flagTargetBlockHeight = "target-block-height"
	// Number of blocks to be archived this time
	flagBlockInterval = "block-interval"
)

func ArchiveCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "archive blockchain data",
		Long:  "archive blockchain data and restore blockchain data",
	}

	cmd.AddCommand(dumpCMD())

	return cmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&adminKeyFilePaths, flagAdminKeyFilePaths, "", "specify admin key file paths, use ',' to separate")
	flags.StringVar(&adminCrtFilePaths, flagAdminCrtFilePaths, "", "specify admin cert file paths, use ',' to separate")
	flags.StringVar(&dbType, flagDbType, "", "Database type. eg. mysql")
	flags.StringVar(&dbDest, flagDbDest, "", "Database destination. eg. user:password:localhost:port")
	flags.Uint64Var(&targetBlockHeight, flagTargetBlockHeight, 10000, "Height of the target block for this archive task")
	flags.Uint64Var(&blockInterval, flagBlockInterval, 1000, "Number of blocks to be archived this time")
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}

// TODO: abstract this function, copied from client package
// createChainClient create a chain client
func createChainClient(keyFilePath, crtFilePath, chainId string) (*sdk.ChainClient, error) {
	cc, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfPath),
		sdk.WithChainClientChainId(chainId),
		sdk.WithUserKeyFilePath(keyFilePath),
		sdk.WithUserCrtFilePath(crtFilePath),
	)
	if err != nil {
		return nil, err
	}

	// Enable certificate compression
	err = cc.EnableCertHash()
	if err != nil {
		return nil, err
	}
	return cc, nil
}
