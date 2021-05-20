/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/query/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/query/model"
	sdk "chainmaker.org/chainmaker-sdk-go"
)

var (
	// sdk config file path
	sdkConfPath string

	chainId string

	// cert and key
	adminKeyFilePaths string
	adminCrtFilePaths string

	dbType string
	dbDest string
)

const (
	// TODO: wrap common flags to a separate package?
	//// Common flags
	// sdk config file path flag
	flagSdkConfPath = "sdk-conf-path"
	flagChainId     = "chain-id"

	//// Archive flags
	// Off-chain database type. eg. mysql,mongodb,pgsql
	flagDbType = "type"
	// Off-chain database destination. eg. user:password:localhost:port
	flagDbDest = "dest"
)

func QueryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "query blockchain data",
		Long:  "query blockchain data",
	}

	cmd.AddCommand(newTxCMD())
	cmd.AddCommand(newBlockCMD())
	cmd.AddCommand(newArchivedHeightCMD())

	return cmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	flags.StringVar(&chainId, flagChainId, "", "Chain ID")
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")
	flags.StringVar(&dbType, flagDbType, "", "Database type. eg. mysql")
	flags.StringVar(&dbDest, flagDbDest, "", "Database destination. eg. user:password:localhost:port")
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
