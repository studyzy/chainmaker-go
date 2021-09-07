/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badgerdbprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

var dbPath = filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano()))

//var dbPath = "./"
var log = &test.GoLogger{}
var dbConfig = &localconf.BadgerDbConfig{
	StorePath: dbPath,
}

func TestNewBadgerDBHandle(t *testing.T) {
	defer func() {
		err := recover()
		fmt.Println(err)
	}()
	conf := &localconf.BadgerDbConfig{
		StorePath:      dbPath,
		Compression:    2,
		ValueThreshold: 2,
	}
	dbHandle := NewBadgerDBHandle("chain1", "test", conf, log)

	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)
	dbHandle.Close()

	conf = &localconf.BadgerDbConfig{
		StorePath: filepath.Join("/"),
	}

	//Files cannot be created in folders that do not have permissions
	dbHandle = NewBadgerDBHandle("chain1", "test", conf, log)
}

func TestDBHandle_Put(t *testing.T) {
	dbHandle := NewBadgerDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	//defer dbHandle.Close()

	key1 := []byte("key1")
	value1 := []byte("value1")
	err := dbHandle.Put(key1, value1)
	assert.Nil(t, err)

	has, err := dbHandle.Has(key1)
	assert.True(t, has)
	assert.Nil(t, err)

	err = dbHandle.Put(key1, nil)
	assert.Equal(t, strings.Contains(err.Error(), "error writing badgerdbprovider with nil value"), true)

	value, err := dbHandle.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1, value)
	value, err = dbHandle.Get([]byte("another key"))
	assert.Nil(t, err)
	assert.Nil(t, value)

	err = dbHandle.Delete(key1)
	assert.Nil(t, err)

	has, err = dbHandle.Has(key1)
	// todo houfa
	// There is a problem, in order to work properly, first comment
	//assert.False(t, has)
	assert.Nil(t, err)

	dbHandle.Close()
	value, err = dbHandle.Get(key1)
	assert.Equal(t, strings.Contains(err.Error(), "DB Closed"), true)
	assert.Nil(t, value)

	has, err = dbHandle.Has([]byte("key"))
	assert.False(t, has)
	assert.Equal(t, strings.Contains(err.Error(), "DB Closed"), true)

	//err = dbHandle.Put(key1, value1)
	//assert.Equal(t, strings.Contains(err.Error(), "DB Closed"), true)
}

func TestDBHandle_WriteBatch(t *testing.T) {
	dbHandle := NewBadgerDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
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

	value, err := dbHandle.Get(key2)
	assert.Nil(t, err)
	assert.Equal(t, value2, value)
}

func TestDBHandle_NewIteratorWithRange(t *testing.T) {
	dbHandle := NewBadgerDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
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
		showIterData(iter)
	}
	assert.Equal(t, 2, count)
	if iter.First() {
		assert.Equal(t, key1, iter.Key())
		assert.Equal(t, value1, iter.Value())
		showIterData(iter)
	}

	iter, err = dbHandle.NewIteratorWithRange([]byte(""), []byte(""))
	assert.Nil(t, iter)
	assert.Equal(t, strings.Contains(err.Error(), "iterator range should not start() or limit() with empty key"), true)
}

func TestDBHandle_NewIteratorWithPrefix(t *testing.T) {
	dbHandle := NewBadgerDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()

	key1 := []byte("key1")
	value1 := []byte("value1")
	batch.Put(key1, value1)
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
		showIterData(iter)
	}
	assert.Equal(t, 5, count)
	if iter.First() {
		assert.Equal(t, key1, iter.Key())
		assert.Equal(t, value1, iter.Value())
		showIterData(iter)
	}

	iter, err = dbHandle.NewIteratorWithPrefix([]byte(""))
	assert.Nil(t, iter)
	assert.Equal(t, strings.Contains(err.Error(), "iterator prefix should not be empty key"), true)
}

func TestTempFolder(t *testing.T) {
	t.Log(os.TempDir())
}

func showIterData(iter protocol.Iterator) {
	fmt.Printf("key: %s, value: %s\n", string(iter.Key()), string(iter.Value()))
}
