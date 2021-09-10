// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"encoding/binary"
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
)

const (
	defaultDbType                 = "mysql"
	configBlockArchiveErrorString = "config block do not need archive"
)

var (
	// sdk config file path
	sdkConfPath string

	chainId string

	dbType                  string
	dbDest                  string
	target                  string
	blocks                  uint64
	secretKey               string
	restoreStartBlockHeight uint64
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
	flags.StringVar(&target, flagTarget, "", "Height or Date of the target block for this archive task."+
		" eg."+
		" 100 (block height) or \"2006-01-02 15:04:05\" (date)")
	flags.Uint64Var(&blocks, flagBlocks, 1000, "Number of blocks to be archived this time")
	flags.StringVar(&secretKey, flagSecretKey, "", "Secret Key for calc Hmac")
	flags.Uint64Var(&restoreStartBlockHeight, flagStartBlockHeight, 0, "Restore starting block height")
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

// hmac SM3(Fchain_id+Fblock_height+Fblock_with_rwset+key)
func hmac(chainId string, blkHeight uint64, blkWithRWSetBytes []byte, secretKey string) (string, error) {
	blkHeightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(blkHeightBytes, blkHeight)

	var data []byte
	data = append(data, []byte(chainId)...)
	data = append(data, blkHeightBytes...)
	data = append(data, blkWithRWSetBytes...)
	data = append(data, []byte(secretKey)...)
	return util.SM3(data)
}
