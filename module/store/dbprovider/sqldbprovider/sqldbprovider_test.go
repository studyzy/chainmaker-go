/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqldbprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}

//func TestProvider_GetDB(t *testing.T) {
//	conf := &localconf.CMConfig{}
//	conf.StorageConfig.MysqlConfig.Dsn = "root:123456@tcp(127.0.0.1:3306)/"
//	conf.StorageConfig.MysqlConfig.MaxIdleConns = 10
//	conf.StorageConfig.MysqlConfig.MaxOpenConns = 10
//	conf.StorageConfig.MysqlConfig.ConnMaxLifeTime = 60
//	var db SqlDBHandle
//	db.GetDB("test_chain_1", conf)
//}
var conf = &localconf.SqlDbConfig{
	Dsn:        "file::memory:?cache=shared",
	SqlDbType:  "sqlite",
	SqlLogMode: "info",
}

func TestNewSqlDBHandle1(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "failed to open mysql:root:123456@tcp(127.0.0.1:3306)"), true)
	}()
	conf :=  &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn: filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
		SqlLogMode: "Warn",
	}
	p := NewSqlDBHandle("test1", conf, log)
	p.Close()

	conf = &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn: filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())),
		SqlLogMode: "Error",
	}
	fmt.Println(conf.Dsn)
	p = NewSqlDBHandle("test1", conf, log)
	p.Close()

	conf = &localconf.SqlDbConfig{
		SqlDbType: "mysql",
		Dsn: "root:123456@tcp(127.0.0.1:3306)",
	}
	p = NewSqlDBHandle("test1", conf, log)
	p.Close()
}

func TestNewSqlDBHandle2(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "uknow sql db type"), true)
	}()
	conf :=  &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn: filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
		SqlLogMode: "test",
	}
	p := NewSqlDBHandle("test1", conf, log)
	p.Close()

	conf =  &localconf.SqlDbConfig{
		SqlDbType: "test",
		Dsn: filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
		SqlLogMode: "test",
	}
	p = NewSqlDBHandle("test1", conf, log)
	p.Close()
}

func TestNewSqlDBHandle3(t *testing.T) {
	defer func() {
		err := recover()
		//assert.Equal(t, strings.Contains(err.(string), "failed to open sqlite path"), true)
		//t.Logf("%#v", err)
		fmt.Println(err)
	}()
	conf :=  &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn: filepath.Join("/"),
		SqlLogMode: "test",
	}
	p := NewSqlDBHandle("test1", conf, log)
	p.Close()
}


func Test_createDatabase(t *testing.T) {
	err := createDatabase("root:123456@tcp(127.0.0.1:3306)/", "test1")
	assert.Equal(t, strings.Contains(err.Error(), "connection refused"), true)
}

func TestProvider_ExecSql(t *testing.T) {

	p := NewSqlDBHandle("chain1", conf, log)
	defer p.Close()
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

	p := NewSqlDBHandle("chain1", conf, log)
	//defer p.Close()
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")
	p.ExecSql("insert into t1 values(2,'b')", "")
	row, err := p.QuerySingle("select count(*) from t1", "")
	assert.Nil(t, err)
	var id int
	err = row.ScanColumns(&id)
	assert.Nil(t, err)
	assert.Equal(t, 2, id)

	row, err = p.QuerySingle("select name from t1 where id=?", 3)
	assert.Nil(t, err)
	assert.True(t, row.IsEmpty())
	data, err := row.Data()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(data))
	p.Close()
	row, err = p.QuerySingle("select name from t1 where id=?", 3)
	assert.Nil(t, row)
	assert.NotNil(t, err)
}
func TestProvider_QueryTableSql(t *testing.T) {

	p := NewSqlDBHandle("chain1", conf, log)
	defer p.Close()
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")
	p.ExecSql("insert into t1 values(2,'b')", "")
	rows, err := p.QueryMulti("select * from t1", "")
	assert.Nil(t, err)
	defer rows.Close()
	var id int
	var name string
	for rows.Next() {
		rows.ScanColumns(&id, &name)
		t.Log(id, name)
	}
}
func initProvider() *SqlDBHandle {
	p := NewSqlDBHandle("chain1", conf, log)
	return p
}
func initData(p *SqlDBHandle) {
	p.ExecSql("create table t1(id int primary key,name varchar(5))", "")
	p.ExecSql("insert into t1 values(1,'a')", "")
	p.ExecSql("insert into t1 values(2,'b')", "")
}

func TestProvider_DbTransaction(t *testing.T) {
	p := initProvider()
	defer p.Close()
	initData(p)
	txName := "Block1"
	tx, _ := p.BeginDbTransaction(txName)
	tx.BeginDbSavePoint("tx0")
	var count int64
	var err error
	getTx, err := p.GetDbTransaction(txName)
	assert.Nil(t, err)
	assert.NotNil(t, getTx)
	count, _ = tx.ExecSql("insert into t1 values(3,'c')")
	assert.Equal(t, int64(1), count)
	count, _ = tx.ExecSql("insert into t1 values(4,'d')")
	assert.Equal(t, int64(1), count)
	tx.BeginDbSavePoint("tx1")
	count, _ = tx.ExecSql("insert into t1 values(5,'e')")
	assert.Equal(t, int64(1), count)
	row, _ := tx.QuerySingle("select count(*) from t1")
	row.ScanColumns(&count)
	assert.Equal(t, int64(5), count)
	count, err = tx.ExecSql("insert into t1 values(2,'b')") //duplicate PK error
	assert.NotNil(t, err)
	tx.RollbackDbSavePoint("tx1")
	row, err = tx.QuerySingle("select count(*) from t1")
	row.ScanColumns(&count)
	assert.Nil(t, err)
	assert.Equal(t, int64(4), count)
	p.RollbackDbTransaction(txName)
	row, err = p.QuerySingle("select count(1) from t1", "")
	assert.Nil(t, err)
	row.ScanColumns(&count)
	assert.Equal(t, int64(2), count)
}

func TestProvider_RollbackEmptyTx(t *testing.T) {
	p := initProvider()
	defer p.Close()
	initData(p)
	txName := "Block1"
	tx, _ := p.BeginDbTransaction(txName)
	tx.BeginDbSavePoint("tx0")
	var count int64
	var err error
	count, err = tx.ExecSql("insert into t1 values(3,'c','error')")
	assert.NotNil(t, err)
	tx.RollbackDbSavePoint("tx0")
	row, err := tx.QuerySingle("select count(*) from t1")
	assert.Nil(t, err)
	row.ScanColumns(&count)
	assert.Equal(t, int64(2), count)
	p.RollbackDbTransaction(txName)
	row, err = p.QuerySingle("select count(1) from t1")
	assert.Nil(t, err)
	row.ScanColumns(&count)
	assert.Equal(t, int64(2), count)
}

func TestProvider_RollbackSavepointByInvalidSql(t *testing.T) {
	p := initProvider()
	defer p.Close()
	initData(p)
	txName := "Block1"
	tx, _ := p.BeginDbTransaction(txName)
	tx.BeginDbSavePoint("tx0")
	var count int64
	var err error
	count, err = tx.ExecSql("insert into t1 values(3,'c')")
	assert.Nil(t, err)
	row, err := tx.QuerySingle("select count(*) from t1")
	row.ScanColumns(&count)
	assert.Equal(t, int64(3), count)
	count, err = tx.ExecSql("insert into t1 values(4,'cc")
	assert.NotNil(t, err)
	tx.RollbackDbSavePoint("tx0")
	row, err = tx.QuerySingle("select count(*) from t1")
	row.ScanColumns(&count)
	assert.Equal(t, int64(2), count)
	p.RollbackDbTransaction(txName)
	row, err = p.QuerySingle("select count(1) from t1")
	row.ScanColumns(&count)
	assert.Equal(t, int64(2), count)
}
func TestSqlDBHandle_QuerySql(t *testing.T) {
	p := initProvider()
	defer p.Close()
	p.ExecSql("create table t1(id int primary key,name varchar(50),birthdate datetime,photo blob)", "")
	var bin = []byte{1, 2, 3, 4, 0xff}
	p.ExecSql("insert into t1 values(?,?,?,?)", 1, "Devin", time.Now(), bin)
	p.ExecSql("insert into t1 values(?,?,?,?)", 2, "Edward", time.Now(), bin)
	p.ExecSql("insert into t1 values(?,?,?,?)", 3, "Devin", time.Now(), bin)
	result, err := p.QuerySingle("select * from t1 where name=?", "Devin")
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(result.Data())
}
func TestReplaceDsn(t *testing.T) {
	tb := make(map[string]string)
	dbName := "blockdb_chain1"
	tb["root:123@456@tcp(127.0.0.1)/mysql?charset=utf8mb4&parseTime=True&loc=Local"] = "root:123@456@tcp(127.0.0.1)/blockdb_chain1?charset=utf8mb4&parseTime=True&loc=Local"
	tb["root:123@456@tcp(127.0.0.1)/{0}"] = "root:123@456@tcp(127.0.0.1)/blockdb_chain1"
	tb["root:123@456@tcp(127.0.0.1)/"] = "root:123@456@tcp(127.0.0.1)/blockdb_chain1"
	tb["root:123@mysql@tcp(127.0.0.1)/mysql"] = "root:123@mysql@tcp(127.0.0.1)/blockdb_chain1"
	tb["root:123@mysql@tcp(127.0.0.1)mysql"] = "root:123@mysql@tcp(127.0.0.1)mysql"
	for dsn, result := range tb {
		replaced := replaceMySqlDsn(dsn, dbName)
		assert.Equal(t, result, replaced)
	}
}

func TestSqlDBHandle_CreateDatabaseIfNotExist(t *testing.T) {
	p := initProvider()
	defer p.Close()

	res, err := p.CreateDatabaseIfNotExist("chain1")
	assert.False(t, res)
	assert.Nil(t, err)

	res, err = p.CreateDatabaseIfNotExist("test2")
	assert.False(t, res)
	assert.Nil(t, err)

	p.dbType, _ = ParseSqlDbType("mysql")
	res, err = p.CreateDatabaseIfNotExist("test2")
	assert.False(t, res)
	assert.Equal(t, strings.Contains(err.Error(), "syntax error"), true)
}

func TestSqlDBHandle_CreateTableIfNotExist(t *testing.T) {
	p := initProvider()
	defer p.Close()

	err := p.CreateTableIfNotExist(&User{})
	assert.Nil(t, err)

	err = p.CreateTableIfNotExist(&User{})
	assert.Nil(t, err)
}

func TestSqlDBHandle_Save(t *testing.T) {
	p := initProvider()
	//defer p.Close()

	user := &User{
		age: 12,
	}
	err := p.CreateTableIfNotExist(&User{})
	assert.Nil(t, err)
	count, err := p.Save(user)
	assert.Nil(t, err)
	assert.Equal(t, count, int64(1))
	p.Close()
	count, err = p.Save(user)
	assert.NotNil(t, err)
	assert.Equal(t, count, int64(0))
}

func TestSqlDBHandle_CommitDbTransaction(t *testing.T) {
	p := initProvider()
	defer p.Close()
	initData(p)
	txName := "Block1"
	tx, _ := p.BeginDbTransaction(txName)
	var count int64
	var err error
	count, _ = tx.ExecSql("insert into t1 values(3,'c')")
	assert.Equal(t, int64(1), count)
	count, _ = tx.ExecSql("insert into t1 values(4,'d')")
	assert.Equal(t, int64(1), count)

	err = tx.ChangeContextDb("test1")
	assert.Nil(t, err)
	err = tx.ChangeContextDb("")
	assert.Nil(t, err)

	row, err := tx.QuerySingle("select * from t1 where id=?", 10)
	assert.True(t, row.IsEmpty())
	assert.Nil(t, err)
	assert.Nil(t, row.ScanColumns())

	rows, err := tx.QueryMulti("select * from t1 where id=?", 4)
	assert.Nil(t, err)
	count = 0
	for rows.Next() {
		count++
	}
	fmt.Println(count)
	assert.Equal(t, count, int64(1))

	err = p.CommitDbTransaction(txName)
	assert.Nil(t, err)

	err = p.CommitDbTransaction(txName)
	assert.Equal(t, strings.Contains(err.Error(), "transaction not found or closed"), true)
}

func TestSqlDBHandle_Close(t *testing.T) {
	defer func() {
		err := recover()
		// dbHandle closed,
		assert.Nil(t, err)
	}()
	p := initProvider()
	//defer p.Close()
	initData(p)
	txName := "Block1"
	tx, _ := p.BeginDbTransaction(txName)
	var count int64
	count, _ = tx.ExecSql("insert into t1 values(3,'c')")
	assert.Equal(t, int64(1), count)
	count, _ = tx.ExecSql("insert into t1 values(4,'d')")
	assert.Equal(t, int64(1), count)

	p.Close()
	p.CommitDbTransaction(txName)
}