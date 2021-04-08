/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStandardSqlVerify_VerifyDDLSql(t *testing.T) {
	table := map[string]bool{}
	table["create   table t1"] = true
	table["alter   table t1 add column"] = true
	table["create index idx_1"] = true
	table["drop TABLE t1"] = true
	table["truncate table t1"] = true
	table["CREATE view v1  "] = true
	table["alter view v11 as select * from t1"] = true
	table["select * from t1 where name='create table t1'"] = false
	table["insert into t2 values(1,'drop table t2')"] = false
	v := &StandardSqlVerify{}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			err := v.VerifyDDLSql(sql)
			if result {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
func TestStandardSqlVerify_VerifyDMLSql(t *testing.T) {
	table := map[string]bool{}

	table["insert into t2 values(1,'drop table t2')"] = true
	table["UPDATE t2 set sql='drop table t2'"] = true
	table["delete  from t1 where id=1"] = true
	table["create   table t1"] = false
	table["truncate table t1"] = false
	table["SELECT * from view1  "] = false

	v := &StandardSqlVerify{}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			err := v.VerifyDMLSql(sql)
			if result {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
func TestStandardSqlVerify_VerifyDQLSql(t *testing.T) {
	table := map[string]bool{}
	table["SElect id,name from t1 where x=333"] = true
	table["select * from t1 inner join t2 where t1.id=t2.id"] = true
	table["select count(*) from t3"] = true
	table["SELECT * from view1  "] = true
	table["insert into t2 values(1,'select * from table')"] = false
	table["UPDATE t2 set sql='select count(*) from table t2'"] = false
	table["delete  from t1 where id=1"] = false
	table["create   table t1"] = false
	table["truncate table t1"] = false

	v := &StandardSqlVerify{}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			err := v.VerifyDQLSql(sql)
			if result {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
