/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	"github.com/stretchr/testify/assert"
	"testing"
)

//func TestProvider_GetDB(t *testing.T) {
//	conf := &localconf.CMConfig{}
//	conf.StorageConfig.MysqlConfig.Dsn = "root:123456@tcp(127.0.0.1:3306)/"
//	conf.StorageConfig.MysqlConfig.MaxIdleConns = 10
//	conf.StorageConfig.MysqlConfig.MaxOpenConns = 10
//	conf.StorageConfig.MysqlConfig.ConnMaxLifeTime = 60
//	var db SqlDBProvider
//	db.GetDB("test_chain_1", conf)
//}
func TestProvider_ExecSql(t *testing.T) {
	conf := &localconf.CMConfig{}
	conf.StorageConfig.MysqlConfig.Dsn = "file::memory:?cache=shared"
	conf.StorageConfig.MysqlConfig.DbType = "sqlite"
	conf.StorageConfig.MysqlConfig.MaxOpenConns = 10
	conf.StorageConfig.MysqlConfig.ConnMaxLifeTime = 60
	p := NewSqlDBProvider("chain1", conf)
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")

	count, err := p.ExecSql("update t1 set name='aa' where id=1", "")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), count)

	count, err = p.ExecSql("update t1 set name1='aa' where id=1", "")
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), count)
}
func TestProvider_QuerySql(t *testing.T) {
	conf := &localconf.CMConfig{}
	conf.StorageConfig.MysqlConfig.Dsn = "file::memory:?cache=shared"
	conf.StorageConfig.MysqlConfig.DbType = "sqlite"
	p := NewSqlDBProvider("chain1", conf)
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")
	p.ExecSql("insert into t1 values(2,'b')", "")
	row, err := p.QuerySql("select count(*) from t1", "")
	assert.Nil(t, err)
	var id int
	err = row.ScanColumns(&id)
	assert.Nil(t, err)
	assert.Equal(t, 2, id)
}
func TestProvider_QueryTableSql(t *testing.T) {
	conf := &localconf.CMConfig{}
	conf.StorageConfig.MysqlConfig.Dsn = "file::memory:?cache=shared"
	conf.StorageConfig.MysqlConfig.DbType = "sqlite"
	p := NewSqlDBProvider("chain1", conf)
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")
	p.ExecSql("insert into t1 values(2,'b')", "")
	rows, err := p.QueryTableSql("select * from t1", "")
	assert.Nil(t, err)
	defer rows.Close()
	var id int
	var name string
	for rows.Next() {
		rows.ScanColumns(&id, &name)
		t.Log(id, name)
	}
}
func initProvider() *SqlDBProvider {
	conf := &localconf.CMConfig{}
	conf.StorageConfig.MysqlConfig.Dsn = "file::memory:?cache=shared"
	conf.StorageConfig.MysqlConfig.DbType = "sqlite"
	p := NewSqlDBProvider("chain1", conf)
	return p
}
func initData(p *SqlDBProvider) {
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")
	p.ExecSql("insert into t1 values(2,'b')", "")
}
func TestProvider_DbTransaction(t *testing.T) {
	p := initProvider()
	initData(p)
	txName := "Block1"
	tx := p.BeginDbTransaction(txName)
	tx.BeginDbSavePoint("tx0")
	var count int64
	var err error
	count, err = tx.ExecSql("insert into t1 values(3,'c')")
	assert.Equal(t, int64(1), count)
	count, err = tx.ExecSql("insert into t1 values(4,'d')")
	assert.Equal(t, int64(1), count)
	tx.BeginDbSavePoint("tx1")
	count, err = tx.ExecSql("insert into t1 values(5,'e')")
	assert.Equal(t, int64(1), count)
	row, err := tx.QuerySql("select count(1) from t1")
	row.ScanColumns(&count)
	assert.Equal(t, int64(5), count)
	count, err = tx.ExecSql("insert into t1 values(2,'b')") //duplicate PK error
	assert.NotNil(t, err)
	tx.RollbackDbSavePoint("tx1")
	row, err = tx.QuerySql("select count(1) from t1")
	row.ScanColumns(&count)
	assert.Equal(t, int64(4), count)
	p.RollbackDbTransaction(txName)
	row, err = p.QuerySql("select count(1) from t1", "")
	row.ScanColumns(&count)
	assert.Equal(t, int64(2), count)
}
