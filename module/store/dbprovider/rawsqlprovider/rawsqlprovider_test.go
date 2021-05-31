/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceMySqlDsn(t *testing.T) {
	tables := []struct {
		dsn    string
		dbName string
		result string
	}{
		{dsn: "root:123456@tcp(127.0.0.1:3306)/", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4"},
		{dsn: "root:123456@tcp(127.0.0.1)/", dbName: "db1", result: "root:123456@tcp(127.0.0.1)/db1?charset=utf8mb4"},
		{dsn: "root:123456@tcp(localhost)/", dbName: "db1", result: "root:123456@tcp(localhost)/db1?charset=utf8mb4"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql?parseTime=True&loc=Local", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?parseTime=True&loc=Local&charset=utf8mb4"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True&loc=Local"},
	}
	for _, tcase := range tables {
		t.Run(tcase.dsn, func(t *testing.T) {
			newDsn := replaceMySqlDsn(tcase.dsn, tcase.dbName)
			assert.Equal(t, tcase.result, newDsn)
		})
	}
}
