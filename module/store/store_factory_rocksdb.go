// +build rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"chainmaker.org/chainmaker-go/store/dbprovider/rocksdbprovider"
	"chainmaker.org/chainmaker/protocol/v2"
)

func init() {
	newRocksdbHandle = func(chainId string, dbName string, logger protocol.Logger) protocol.DBHandle {
		provider := rocksdbprovider.NewProvider(chainId, dbName, logger)
		return provider.GetDBHandle(dbName)
	}
}
