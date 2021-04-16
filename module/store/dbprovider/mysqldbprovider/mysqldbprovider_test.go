/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mysqldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	"testing"
)

func TestProvider_GetDB(t *testing.T) {
	conf := &localconf.CMConfig{}
	conf.StorageConfig.MysqlConfig.Dsn = "root:123456@tcp(127.0.0.1:3306)/"
	conf.StorageConfig.MysqlConfig.MaxIdleConns = 10
	conf.StorageConfig.MysqlConfig.MaxOpenConns = 10
	conf.StorageConfig.MysqlConfig.ConnMaxLifeTime = 60
	var db Provider
	db.GetDB("test_chain_1", conf)
}

