/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badgerdbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker/protocol/test"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var dbPath = filepath.Join(os.TempDir(), fmt.Sprintf("%d_unit_test_db", time.Now().UnixNano()))
var log = &test.GoLogger{}
var dbConfig = &localconf.BadgerDbConfig{
	StorePath: dbPath,
}

func TestDBHandle_Put(t *testing.T) {
	dbHandle := NewBadgerDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	defer dbHandle.Close()

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

	iter := dbHandle.NewIteratorWithRange(key1, []byte("key3"))
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
	}
	assert.Equal(t, 2, count)
}

func TestDBHandle_NewIteratorWithPrefix(t *testing.T) {
	dbHandle := NewBadgerDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
	defer dbHandle.Close()

	batch := types.NewUpdateBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Put([]byte("key4"), []byte("value4"))
	batch.Put([]byte("keyx"), []byte("value5"))

	err := dbHandle.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter := dbHandle.NewIteratorWithPrefix([]byte("key"))
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		key := string(iter.Key())
		fmt.Println(fmt.Sprintf("key: %s", key))
	}
	assert.Equal(t, 5, count)
}

func TestTempFolder(t *testing.T) {
	t.Log(os.TempDir())
}
