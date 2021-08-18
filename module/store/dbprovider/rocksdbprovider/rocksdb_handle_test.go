//+build rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rocksdbprovider

import (
	"bytes"
	"testing"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/store/dbprovider"
	"chainmaker.org/chainmaker-go/store/types"
	"github.com/stretchr/testify/assert"
)

var dbPath = "/tmp/rocksdbprovider/unit_test_db"
var dbName = "db_test"
var chainId = "testchain"
var log = logger.GetLoggerByChain(logger.MODULE_STORAGE, chainId)

func TestDBHandle_Put(t *testing.T) {
	rocksDbConfig := dbprovider.GetMockDBConfig("").BlockDbConfig.RocksDbConfig
	db := NewRocksDBHandle(chainId, rocksDbConfig.StorePath, rocksDbConfig, log) //dbPath：db文件的存储路径
	defer db.Close()

	key1 := []byte("key1")
	value1 := []byte("value1")
	err := db.Put(key1, value1)
	assert.Equal(t, nil, err)

	value, err := db.Get(key1)
	assert.Equal(t, nil, err)
	assert.True(t, bytes.Equal(value1, value))
}

//func TestDBHandle_WriteBatch_Bench(t *testing.T) {
//	rocksDbConfig := getDBConfig("./").BlockDbConfig.RocksDbConfig
//	rocksDbConfig.WriteBufferSize = 128
//	rocksDbConfig.DbWriteBufferSize = 128
//	rocksDbConfig.BlockCache = 128
//	rocksDbConfig.MaxWriteBufferNumber = 10
//	rocksDbConfig.MaxBackgroundCompactions = 4
//	rocksDbConfig.MaxBackgroundFlushes = 2
//	rocksDbConfig.BloomFilterBits = 10
//	rocksDbConfig.MaxOpenFiles = 1000
//
//	db := NewRocksDBHandle(chainId, rocksDbConfig.StorePath, rocksDbConfig, log)
//	defer db.Close()
//	batch := types.NewUpdateBatch()
//
//	cnt := 20000
//	timeStart := utils.CurrentTimeMillisSeconds()
//	for i := 0; i < 100; i ++ {
//		g := dbprovider.NewFullRandomEntryGenerator(0, cnt)
//		for j := 0; j < cnt; j ++ {
//			batch.Put(g.Key(j), g.Value(j))
//			//fmt.Println(fmt.Sprintf("key: %s, value: %s", string(g.Key(j)), string(g.Value(j))))
//		}
//		err := db.WriteBatch(batch, false)
//		assert.Equal(t, nil, err)
//		log.Infof(fmt.Sprintf("WriteBatch :%d st", i))
//	}
//
//	writeTime := utils.CurrentTimeMillisSeconds()
//
//	//fmt.Println(fmt.Sprintf("elapsedPrepare: %d ms", prepareTime - timeStart))
//	fmt.Println(fmt.Sprintf("elapsedWrite: %d ms", writeTime - timeStart))
//
//	//value, err := db.Get(key2)
//	//assert.Equal(t, nil, err)
//	//assert.True(t,  bytes.Equal(value2, value))
//
//}

func TestDBHandle_NewIteratorWithRange(t *testing.T) {
	rocksDbConfig := dbprovider.GetMockDBConfig("").BlockDbConfig.RocksDbConfig
	db := NewRocksDBHandle(chainId, rocksDbConfig.StorePath, rocksDbConfig, log) //dbPath：db文件的存储路径
	defer db.Close()

	batch := types.NewUpdateBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Put([]byte("key4"), []byte("value4"))
	batch.Put([]byte("key5"), []byte("value5"))

	err := db.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter := db.NewIteratorWithRange([]byte("key2"), []byte("key4"))
	defer iter.Release()
	var count int

	for iter.Next() {
		count++
		//key := string(iter.Key())
		//fmt.Println(fmt.Sprintf("key: %s", key))
	}
	assert.Equal(t, 2, count)
}

func TestDBHandle_NewIteratorWithPrefix(t *testing.T) {
	rocksDbConfig := dbprovider.GetMockDBConfig("").BlockDbConfig.RocksDbConfig
	db := NewRocksDBHandle(chainId, rocksDbConfig.StorePath, rocksDbConfig, log) //dbPath：db文件的存储路径
	defer db.Close()

	batch := types.NewUpdateBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))
	batch.Put([]byte("key4"), []byte("value4"))
	batch.Put([]byte("keyx"), []byte("value5"))

	err := db.WriteBatch(batch, true)
	assert.Equal(t, nil, err)

	iter := db.NewIteratorWithPrefix([]byte("key"))
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		//key := string(iter.Key())
		//fmt.Println(fmt.Sprintf("key: %s", key))
	}
	assert.Equal(t, 5, count)
}
