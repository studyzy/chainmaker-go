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

const (
	defaultDbType = "mysql"
)

var (
	// sdk config file path
	sdkConfPath string

	chainId string

	dbType                  string
	dbDest                  string
	target                  string
	blocks                  int64
	secretKey               string
	restoreStartBlockHeight int64
)

const (
	//// Common flags
	flagSdkConfPath = "sdk-conf-path"
	flagChainId     = "chain-id"

	//// Archive flags
	// Off-chain database type. eg. mysql,mongodb,pgsql
	flagDbType = "type"
	// Off-chain database destination. eg. user:password:localhost:port
	flagDbDest = "dest"
	// 1.Archive target block height, stop archiving (include this block) after reaching this height.
	// 2.Archive target date, archive all blocks before this date.
	flagTarget = "target"
	// Number of blocks to be archived this time
	flagBlocks = "blocks"
	// Secret Key for calc Hmac
	flagSecretKey = "secret-key"
	// block height of restore
	flagStartBlockHeight = "start-block-height"
)

func NewArchiveCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "archive blockchain data",
		Long:  "archive blockchain data and restore blockchain data",
	}

	cmd.AddCommand(newDumpCMD())
	cmd.AddCommand(newRestoreCMD())
	cmd.AddCommand(newQueryOffChainCMD())

	return cmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&dbType, flagDbType, "mysql", "Database type. eg. mysql")
	flags.StringVar(&dbDest, flagDbDest, "", "Database destination. eg. user:password:localhost:port")
	flags.StringVar(&target, flagTarget, "", "Height or Date of the target block for this archive task\neg. 100 (block height) or `2006-01-02 15:04:05` (date)")
	flags.Int64Var(&blocks, flagBlocks, 1000, "Number of blocks to be archived this time")
	flags.StringVar(&secretKey, flagSecretKey, "", "Secret Key for calc Hmac")
	flags.Int64Var(&restoreStartBlockHeight, flagStartBlockHeight, 0, "Restore starting block height")
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}

func markFlagsRequired(cmd *cobra.Command, names []string) {
	for _, name := range names {
		_ = cmd.MarkFlagRequired(name)
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

	// migrate sysinfo table
	err = db.AutoMigrate(&model.Sysinfo{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
