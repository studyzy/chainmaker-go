/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dbprovider

import (
	"fmt"
	"strings"

	rawsqlprovider "chainmaker.org/chainmaker/store-sqldb/v2"

	"chainmaker.org/chainmaker/protocol/v2"
	badgerdbprovider "chainmaker.org/chainmaker/store-badgerdb/v2"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"
	"github.com/mitchellh/mapstructure"
)

type DBFactory struct {
}

func (f *DBFactory) NewKvDB(chainId, providerName, dbFolder string, config map[string]interface{},
	logger protocol.Logger) (protocol.DBHandle, error) {
	providerName = strings.ToLower(providerName)
	if providerName == "leveldb" {
		dbConfig := &leveldbprovider.LevelDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &leveldbprovider.NewLevelDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: nil,
			ChainId:   chainId,
			DbFolder:  dbFolder,
		}
		return leveldbprovider.NewLevelDBHandle(input), nil
	}
	if providerName == "badgerdb" {
		dbConfig := &badgerdbprovider.BadgerDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &badgerdbprovider.NewBadgerDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: nil,
			ChainId:   chainId,
			DbFolder:  dbFolder,
		}
		return badgerdbprovider.NewBadgerDBHandle(input), nil
	}
	if providerName == "sql" {
		dbConfig := &rawsqlprovider.SqlDbConfig{}
		err := mapstructure.Decode(config, dbConfig)
		if err != nil {
			return nil, err
		}
		input := &rawsqlprovider.NewSqlDBOptions{
			Config:    dbConfig,
			Logger:    logger,
			Encryptor: nil,
			ChainId:   chainId,
			DbName:    dbFolder,
		}
		return rawsqlprovider.NewSqlDBHandle(input), nil
	}
	return nil, fmt.Errorf("unsupported provider:%s", providerName)
}

func (f *DBFactory) NewSqlDB(chainId, providerName, dbName string, config map[string]interface{},
	logger protocol.Logger) (protocol.SqlDBHandle, error) {
	dbConfig := &rawsqlprovider.SqlDbConfig{}
	err := mapstructure.Decode(config, dbConfig)
	if err != nil {
		return nil, err
	}
	input := &rawsqlprovider.NewSqlDBOptions{
		Config:    dbConfig,
		Logger:    logger,
		Encryptor: nil,
		ChainId:   chainId,
		DbName:    dbName,
	}
	return rawsqlprovider.NewSqlDBHandle(input), nil
}
