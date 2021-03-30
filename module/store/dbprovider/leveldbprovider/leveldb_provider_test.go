/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package leveldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/store/types"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

var dbPath = filepath.Join(os.TempDir(), "unit_test_db")
var dbName = "db_test"
var log = &logger.GoLogger{}
var dbConfig = &localconf.LevelDbConfig{
	StorePath: dbPath,
}

func TestDBHandle_Put(t *testing.T) {
	db := NewLevelDBProvider("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	dbHandle := db.GetDBHandle(dbName)                        //dbName：db的逻辑表名，不同业务采用不同的dbName，其他模块使用db，定义自己的dbName,避免重复
	defer db.Close()

	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1, value)
}

func TestDBHandle_WriteBatch(t *testing.T) {
	db := NewLevelDBProvider("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	dbHandle := db.GetDBHandle(dbName)                        //dbName：db的逻辑表名，不同业务采用不同的dbName，其他模块使用db，定义自己的dbName,避免重复
	defer db.Close()
	batch := types.NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key2)
	assert.Nil(t, err)
	assert.Equal(t, value2, value)

}

func TestDBHandle_NewIteratorWithRange(t *testing.T) {
	db := NewLevelDBProvider("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	dbHandle := db.GetDBHandle(dbName)                        //dbName：db的逻辑表名，不同业务采用不同的dbName，其他模块使用db，定义自己的dbName,避免重复
	defer db.Close()

	batch := types.NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	iter := dbHandle.NewIteratorWithRange(key1, []byte("key3"))
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	assert.Equal(t, 2, count)
}
func TestTempFolder(t *testing.T) {
	t.Log(os.TempDir())
}
