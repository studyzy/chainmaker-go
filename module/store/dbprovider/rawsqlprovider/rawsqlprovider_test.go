/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

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
var confProvideTest = &localconf.SqlDbConfig{
	SqlDbType: "sqlite",
	Dsn:       filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
}

func TestReplaceMySqlDsn(t *testing.T) {
	tables := []struct {
		dsn    string
		dbName string
		result string
	}{
		{dsn: "root:123456@tcp(127.0.0.1:3306)/", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True"},
		{dsn: "root:123456@tcp(127.0.0.1)/", dbName: "db1", result: "root:123456@tcp(127.0.0.1)/db1?charset=utf8mb4&parseTime=True"},
		{dsn: "root:123456@tcp(localhost)/", dbName: "db1", result: "root:123456@tcp(localhost)/db1?charset=utf8mb4&parseTime=True"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql?parseTime=True&loc=Local", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?parseTime=True&loc=Local&charset=utf8mb4"},
		{dsn: "root:123456@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4&parseTime=True&loc=Local", dbName: "db1", result: "root:123456@tcp(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True&loc=Local"},
		{dsn: "a:b@tcp", dbName: "db1", result: "a:b@tcp"},
	}
	for _, tcase := range tables {
		t.Run(tcase.dsn, func(t *testing.T) {
			newDsn := replaceMySqlDsn(tcase.dsn, tcase.dbName)
			assert.Equal(t, tcase.result, newDsn)
		})
	}
}

func TestNewSqlDBHandle1(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "connect to mysql error"), true)
	}()
	conf := &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn:       filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())+":memory:"),
	}
	dbHandle := NewSqlDBHandle("test1", conf, log)
	dbHandle.Close()

	conf = &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn:       filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())),
	}
	fmt.Println(conf.Dsn)
	dbHandle = NewSqlDBHandle("test1", conf, log)
	dbHandle.Close()

	conf = &localconf.SqlDbConfig{
		SqlDbType: "mysql",
		Dsn:       "root:123456@tcp(127.0.0.1:3306)",
	}
	dbHandle = NewSqlDBHandle("test1", conf, log)
	dbHandle.Close()
}

func TestNewSqlDBHandle2(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "unknown sql db type:test"), true)
	}()
	conf := &localconf.SqlDbConfig{
		SqlDbType: "test",
	}
	dbHandle := NewSqlDBHandle("test1", conf, log)
	dbHandle.Close()
}

func TestNewSqlDBHandle3(t *testing.T) {
	defer func() {
		err := recover()
		//assert.Equal(t, strings.Contains(err.(string), "failed to create folder for sqlite path"), true)
		fmt.Println(err)
	}()
	conf := &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn:       filepath.Join("/", fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano())),
	}
	dbHandle := NewSqlDBHandle("test1", conf, log)
	dbHandle.Close()
}

func TestSqlDBHandle_CreateDatabaseIfNotExist(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	defer dbHandle.Close()

	res, err := dbHandle.CreateDatabaseIfNotExist("test2")
	assert.Nil(t, err)
	assert.True(t, res)

	res, err = dbHandle.CreateDatabaseIfNotExist("test1")
	assert.Nil(t, err)
	assert.True(t, res)

	dbHandle.dbType, err = ParseSqlDbType("mysql")
	res, err = dbHandle.CreateDatabaseIfNotExist("test3")
	assert.NotNil(t, err)
	assert.False(t, res)

	res, err = dbHandle.CreateDatabaseIfNotExist("test1")
	assert.Nil(t, err)
	assert.True(t, res)
}

func TestSqlDBHandle_CreateTableIfNotExist(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "Unsupported db type:mysql"), true)
	}()
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	defer dbHandle.Close()

	err := dbHandle.CreateTableIfNotExist(&SavePoint{})
	assert.Nil(t, err)

	_, err = dbHandle.ExecSql("DROP TABLE save_points", "")
	assert.Nil(t, err)

	_, err = dbHandle.ExecSql("DROP TABLE save_points", "")
	assert.NotNil(t, err)

	err = dbHandle.CreateTableIfNotExist("test")
	assert.NotNil(t, err)

	err = dbHandle.CreateTableIfNotExist(&SavePoint{})
	assert.Nil(t, err)

	dbHandle.dbType, err = ParseSqlDbType("mysql")
	err = dbHandle.CreateTableIfNotExist(&SavePoint{})
	assert.NotNil(t, err)
}

func TestSqlDBHandle_Save(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	//defer dbHandle.Close()

	err := dbHandle.CreateTableIfNotExist(&SavePoint{})
	assert.Nil(t, err)

	_, err = dbHandle.Save(&SavePoint{})
	assert.Nil(t, err)

	_, err = dbHandle.Save(&SavePoint{})
	assert.Nil(t, err)

	_, err = dbHandle.Save("test")
	assert.NotNil(t, err)

	dbHandle.Close()
	_, err = dbHandle.Save(&SavePoint{})
	assert.NotNil(t, err)
}

func TestSqlDBHandle_QuerySingle(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	//defer dbHandle.Close()

	point := &SavePoint{
		BlockHeight: 10,
	}

	sql, value := point.GetInsertSql()
	_, err := dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	err = dbHandle.CompactRange([]byte("1"), []byte("2"))
	assert.NotNil(t, err)

	res, err := dbHandle.QuerySingle(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height = ?", point.GetTableName()), 10)
	assert.Nil(t, err)

	data, err := res.Data()
	blockHeight := string(data["block_height"])
	assert.Nil(t, err)
	assert.Equal(t, "10", blockHeight)

	res, err = dbHandle.QuerySingle(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height = ?", point.GetTableName()), 11)
	assert.Nil(t, err)
	data, err = res.Data()
	assert.Equal(t, 0, len(data))

	dbHandle.Close()
	_, err = dbHandle.QuerySingle(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height = ?", point.GetTableName()), 10)
	assert.NotNil(t, err)
}

func TestSqlDBHandle_QueryMulti(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	//defer dbHandle.Close()

	point1 := &SavePoint{
		BlockHeight: 20,
	}
	point2 := &SavePoint{
		BlockHeight: 21,
	}
	point3 := &SavePoint{
		BlockHeight: 22,
	}

	sql, value := point1.GetInsertSql()
	_, err := dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	sql, value = point2.GetInsertSql()
	_, err = dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	sql, value = point3.GetInsertSql()
	_, err = dbHandle.ExecSql(sql, value...)
	assert.Nil(t, err)

	res, err := dbHandle.QueryMulti(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height>=? AND block_height<=?", point1.GetTableName()), 20, 29)
	assert.Nil(t, err)
	count := 0
	for res.Next() {
		count++
	}
	assert.Equal(t, 3, count)

	dbHandle.Close()
	_, err = dbHandle.QueryMulti(fmt.Sprintf("SELECT block_height FROM %s WHERE block_height>=?", point1.GetTableName()), 20)
	assert.NotNil(t, err)
}

func TestSqlDBHandle_BeginDbTransaction(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	//defer dbHandle.Close()

	txName1 := "1234567890"
	txName2 := "1234567890123"
	_, err := dbHandle.BeginDbTransaction(txName1)
	assert.Nil(t, err)

	_, err = dbHandle.GetDbTransaction(txName1)
	assert.Nil(t, err)

	_, err = dbHandle.GetDbTransaction(txName2)
	assert.NotNil(t, err)

	_, err = dbHandle.BeginDbTransaction(txName1)
	assert.NotNil(t, err)

	dbHandle.Close()
	_, err = dbHandle.BeginDbTransaction(txName2)
	assert.NotNil(t, err)
}

func TestSqlDBHandle_CommitDbTransaction(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	defer dbHandle.Close()

	txName1 := "1234567890"
	txName2 := "1234567890123"
	err := dbHandle.CommitDbTransaction(txName1)
	assert.NotNil(t, err)

	_, err = dbHandle.BeginDbTransaction(txName1)
	assert.Nil(t, err)

	err = dbHandle.CommitDbTransaction(txName1)
	assert.Nil(t, err)

	_, err = dbHandle.BeginDbTransaction(txName2)
	assert.Nil(t, err)
}

func TestSqlDBHandle_RollbackDbTransaction(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	defer dbHandle.Close()

	txName1 := "1234567890"
	err := dbHandle.RollbackDbTransaction(txName1)
	assert.NotNil(t, err)

	_, err = dbHandle.BeginDbTransaction(txName1)
	assert.Nil(t, err)

	err = dbHandle.RollbackDbTransaction(txName1)
	assert.Nil(t, err)
}

func TestSqlDBHandle_createDatabase(t *testing.T) {
	dbHandle := NewSqlDBHandle("test1", confProvideTest, log)
	defer dbHandle.Close()

	err := dbHandle.createDatabase("root:123456@tcp(127.0.0.1:3306)", "test2")
	assert.NotNil(t, err)
}
