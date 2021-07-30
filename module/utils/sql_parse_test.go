/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTableName(t *testing.T) {
	table := map[string]bool{}

	table["insert into t2 values(1,'drop table dbo.t1')"] = true
	table["UPDATE t2 set sql='drop table anotherdb.t2'"] = true
	table["delete  from Teacher t1 where id=1"] = true
	table["create   table db2.t1"] = false
	table["truncate table db3.t1"] = false
	table["select * from t1 where id in (SELECT * from db4.view1 ) "] = false

	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			tableNames := GetSqlTableName(sql)
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
func TestMultiStatement(t *testing.T) {
	table := map[string]int{}

	table["insert into t2 values(1,'drop table dbo.t1')"] = 1
	table["UPDATE t2 set name='drop table anotherdb.t2' where id=1"] = 1
	table["select * from table1; select * from table2"] = 2
	table["create   table db2.t1;drop table t2;select * from t3 where sqlStr='drop table t4;'"] = 3
	table["insert into t1 values(1,'a');update t1 set a=b"] = 2
	table["select * from t1 where id in (SELECT * from db4.view1 ) "] = 1

	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			count := GetSqlStatementCount(sql)
			t.Log(sql, count)
			assert.Equal(t, result, count)
		})
	}
}
