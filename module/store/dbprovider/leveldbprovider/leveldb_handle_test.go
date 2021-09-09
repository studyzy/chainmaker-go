/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package leveldbprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

var dbPath = filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano()))
var log = &test.GoLogger{}
var dbConfig = &LevelDbConfig{
	StorePath: dbPath,
}

func TestDBHandle_NewLevelDBHandle(t *testing.T) {
	defer func() {
		err := recover()
		//assert.Equal(t, strings.Contains(err.(string), "Error create dir"), true)
		fmt.Println(err)
	}()
	dbConfigTest := &LevelDbConfig{
		StorePath:            dbPath,
		BlockWriteBufferSize: 2,
	}
	op := &NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfigTest, Logger: log}
	dbHandle1 := NewLevelDBHandle(op)
	dbHandle1.Close()

	dbConfigTest = &LevelDbConfig{
		StorePath: dbPath,
	}
	dbHandle2 := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfigTest, Logger: log})
	dbHandle2.Close()

	dbConfigTest = &LevelDbConfig{
		StorePath: filepath.Join("/"),
	}
	dbHandle3 := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfigTest, Logger: log})
	dbHandle3.Close()
}

func TestDBHandle_Put(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	//defer dbHandle.Close()

	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1, value)
	value, err = dbHandle.Get([]byte("another key"))
	assert.Nil(t, err)
	assert.Nil(t, value)

	dbHandle.Close()

	_, err = dbHandle.Get(key1)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "getting leveldbprovider key"), true)

	err = dbHandle.Put(key1, nil)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "writing leveldbprovider with nil value"), true)

	key2 := []byte("key2")
	value2 := []byte("value2")
	err = dbHandle.Put(key2, value2)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "writing leveldbprovider key"), true)
}

func TestDBHandle_Delete(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	//defer dbHandle.Close()

	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	exist, err := dbHandle.Has(key1)
	assert.True(t, exist)

	err = dbHandle.Delete(key1)
	assert.Nil(t, err)

	exist, err = dbHandle.Has(key1)
	assert.False(t, exist)

	dbHandle.Close()

	err = dbHandle.Delete(key1)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "deleting leveldbprovider key"), true)

	exist, err = dbHandle.Has(key1)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "getting leveldbprovider key"), true)
}

func TestDBHandle_WriteBatch(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	//defer dbHandle.Close()
	batch := types.NewUpdateBatch()

	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err = dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	key3 := []byte("key3")
	value3 := []byte("")
	batch.Put(key3, value3)
	err = dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	value, err := dbHandle.Get(key2)
	assert.Nil(t, err)
	assert.Equal(t, value2, value)

	dbHandle.Close()

	err = dbHandle.WriteBatch(batch, true)
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "error writing batch to leveldb provider"), true)
}

func TestDBHandle_CompactRange(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	err = dbHandle.CompactRange(key1, key2)
	assert.Nil(t, err)
}

func TestDBHandle_NewIteratorWithRange(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()
	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	batch.Put(key1, value1)
	batch.Put(key2, value2)
	err := dbHandle.WriteBatch(batch, true)
	assert.Nil(t, err)

	iter, err := dbHandle.NewIteratorWithRange(key1, []byte("key3"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	assert.Equal(t, 2, count)

	_, err = dbHandle.NewIteratorWithRange([]byte(""), []byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator range should not start"), true)
}

func TestDBHandle_NewIteratorWithPrefix(t *testing.T) {
	dbHandle := NewLevelDBHandle(&NewLevelDBOptions{ChainId: "chain1", DbFolder: "test", Config: dbConfig, Logger: log}) //dbPath：db文件的存储路径
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Put([]byte("key4"), []byte("value4"))
	batch.Put([]byte("keyx"), []byte("value5"))

	err := dbHandle.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter, err := dbHandle.NewIteratorWithPrefix([]byte("key"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		//key := string(iter.Key())
		//fmt.Println(fmt.Sprintf("key: %s", key))
	}
	assert.Equal(t, 5, count)

	_, err = dbHandle.NewIteratorWithPrefix([]byte(""))
	assert.NotNil(t, err)
	assert.Equal(t, strings.Contains(err.Error(), "iterator prefix should not be empty key"), true)
}

func TestTempFolder(t *testing.T) {
	t.Log(os.TempDir())
}
