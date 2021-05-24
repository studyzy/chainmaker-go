// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
)

var (
	// sdk config file path
	sdkConfPath string

	chainId string

	// cert and key
	adminKeyFilePaths string
	adminCrtFilePaths string

	dbType          string
	dbDest          string
	targetBlkHeight int64
	blocks          int64
	secretKey       string
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
	flagBlocks = "blocks"
	// Secret Key for calc Hmac
	flagSecretKey = "secret-key"
)

func NewArchiveCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "archive blockchain data",
		Long:  "archive blockchain data and restore blockchain data",
	}

	cmd.AddCommand(newDumpCMD())
	cmd.AddCommand(newQueryOffChainCMD())

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
	flags.Int64Var(&targetBlkHeight, flagTargetBlockHeight, 10000, "Height of the target block for this archive task")
	flags.Int64Var(&blocks, flagBlocks, 1000, "Number of blocks to be archived this time")
	flags.StringVar(&secretKey, flagSecretKey, "", "Secret Key for calc Hmac")
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}

// initDb Connecting database, migrate tables.
func initDb() (*gorm.DB, error) {
	// parse params
	dbName := model.DbName(chainId)
	dbDestSlice := strings.Split(dbDest, ":")
	if len(dbDestSlice) != 4 {
		return nil, errors.New("invalid database destination")
	}

	// initialize database
	db, err := mysql.InitDb(dbDestSlice[0], dbDestSlice[1], dbDestSlice[2], dbDestSlice[3], dbName, true)
	if err != nil {
		return nil, err
	}

	// migrate blockinfo,sysinfo tables
	err = db.AutoMigrate(&model.BlockInfo{}, &model.Sysinfo{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
