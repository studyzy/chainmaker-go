/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestGetTableName(t *testing.T) {
	table := map[string]bool{}

	table["insert into t2 values(1,'drop table dbo.t1')"] = true
	table["UPDATE t2 set sql='drop table anotherdb.t2'"] = true
	table["delete  from Teacher t1 where id=1"] = true
	table["create   table db2.t1"] = false
	table["truncate table db3.t1"] = false
	table["select * from t1 where id in (SELECT * from db4.view1 ) "] = false

	v := &StandardSqlVerify{}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			tableNames := v.getSqlTableName(sql)
			t.Log(sql, tableNames)
			assert.Equal(t, result, !containDot(tableNames))
		})
	}
}
func containDot(strs []string) bool {
	for _, str := range strs {
		if strings.Contains(str, ".") {
			return true
		}
	}
	return false
}
