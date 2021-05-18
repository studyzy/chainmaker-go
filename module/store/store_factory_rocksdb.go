// +build rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/rocksdbprovider"
)

func init() {
	newRocksdbHandle = func(chainId string, dbName string) protocol.DBHandle {
		provider := rocksdbprovider.NewProvider(chainId, dbName)
		return provider.GetDBHandle(dbName)
	}
}
