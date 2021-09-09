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

	"chainmaker.org/chainmaker-go/store/dbprovider/badgerdbprovider"
	"chainmaker.org/chainmaker/protocol/v2"
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
	return nil, fmt.Errorf("unsupported provider:%s", providerName)
}
