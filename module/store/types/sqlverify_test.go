/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
func TestStandardSqlVerify_ForbiddenCheck(t *testing.T) {
	table := map[string]bool{}
	table["SElect id,name from t1 where x=333"] = true
	table["SElect id,name from t1 where x='a';drop table student;--"] = false
	table["SElect id,name from db3.t1 where x=333"] = false
	table[" use db2;select * from t1 inner join t2 where t1.id=t2.id"] = false
	v := &StandardSqlVerify{}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			err := v.checkForbiddenSql(sql)
			if result {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				t.Log(sql, err)
			}
		})
	}
}
func TestFindStringRange(t *testing.T) {
	table := map[string][][2]int{}
	table["'abc'"] = [][2]int{{0, 4}}
	table["\"abc\""] = [][2]int{{0, 4}}
	table["'a\"b\"c'"] = [][2]int{{0, 6}}
	table["\"'abc'\""] = [][2]int{{0, 6}}
	table["'a''b''c'"] = [][2]int{{0, 8}}
	table["'ab','c'd"] = [][2]int{{0, 3}, {5, 7}}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			r := findStringRange(sql)
			t.Log(sql, r)
			assert.Equal(t, result, r)
		})
	}
}
func TestStandardSqlVerify_checkHasForbiddenKeyword(t *testing.T) {
	table := map[string]bool{}
	table["SElect id,name from t1 where x=newid()"] = false
	table["SElect id,name from t1 where x='select * from rand()'"] = true
	table["SElect * from t1 where x=333 and y='select 'xx' a,sysdate()'"] = true
	table[" update t2 set time=now()"] = false
	table[" update t2 set now1=now2"] = true
	table[" create table t1 (id int primary key identity ,name varchar(10))"] = false
	table["create table t2 (identity1 int primary key,name varchar(10))"] = true
	v := &StandardSqlVerify{}
	for sql, result := range table {
		t.Run(sql, func(t *testing.T) {
			formatSql, _ := v.getFmtSql(sql)
			err := v.checkHasForbiddenKeyword(formatSql)
			if result {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				t.Log(sql, err)
			}
		})
	}
}
