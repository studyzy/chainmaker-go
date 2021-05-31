/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package leveldbprovider

import (
	"os"
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker/protocol/test"
	"github.com/stretchr/testify/assert"
)

var dbPath = filepath.Join(os.TempDir(), "unit_test_db")
var log = &test.GoLogger{}
var dbConfig = &localconf.LevelDbConfig{
	StorePath: dbPath,
}

func TestDBHandle_Put(t *testing.T) {
	dbHandle := NewLevelDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
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
	dbHandle := NewLevelDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
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
	dbHandle := NewLevelDBHandle("chain1", "test", dbConfig, log) //dbPath：db文件的存储路径
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
func TestTempFolder(t *testing.T) {
	t.Log(os.TempDir())
}
