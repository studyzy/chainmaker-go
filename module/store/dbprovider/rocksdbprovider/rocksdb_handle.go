// +build rocksdb

/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rocksdbprovider

import (
	"fmt"
	"os"
	"path/filepath"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/pkg/errors"
	"github.com/yiyanwannian/gorocksdb"
)

const (
	defaultWriteBufferSize          = 128
	defaultDBWriteBufferSize        = 128
	defaultBlockCacheSize           = 128
	defaultMaxWriteBufferNumber     = 10
	defaultMaxBackgroundCompactions = 4
	defaultMaxBackgroundFlushes     = 2
	defaultBloomFilterBits          = 10
	defaultMaxOpenFiles             = -1
)

const (
	KiB = 1024
	MiB = KiB * 1024
)

const (
	//StoreBlockDBDir blockdb folder name
	StoreBlockDBDir = "store_block"
	//StoreStateDBDir statedb folder name
	StoreStateDBDir = "store_state"
	//StoreHistoryDBDir historydb folder name
	StoreHistoryDBDir = "store_history"
	//StoreResultDBDir resultdb folder name
	StoreResultDBDir = "store_result"
)

var DbNameKeySep = []byte{0x00}

// RocksDBHandle encapsulated handle to rocksdb
type RocksDBHandle struct {
	db           *gorocksdb.DB
	readOptions  *gorocksdb.ReadOptions
	writeOptions *gorocksdb.WriteOptions
	logger       protocol.Logger
}

func NewRocksDBHandle(chainId string, dbFolder string, dbconfig *localconf.RocksDbConfig,
	logger protocol.Logger) *RocksDBHandle {

	dbOpts := NewRocksdbConfig()
	if dbconfig.WriteBufferSize > 0 {
		dbOpts.writeBufferSize = dbconfig.WriteBufferSize * MiB
	}
	if dbconfig.DbWriteBufferSize > 0 {
		dbOpts.dbWriteBufferSize = dbconfig.DbWriteBufferSize * MiB
	}
	if dbconfig.BlockCache > 0 {
		dbOpts.blockCache = dbconfig.BlockCache * MiB
	}
	if dbconfig.BloomFilterBits > 0 {
		dbOpts.bloomFilterBits = dbconfig.BloomFilterBits * MiB
	}
	if dbconfig.MaxWriteBufferNumber > 0 {
		dbOpts.maxWriteBufferNumber = dbconfig.MaxWriteBufferNumber
	}
	if dbconfig.MaxBackgroundCompactions > 0 {
		dbOpts.maxBackgroundCompactions = dbconfig.MaxBackgroundCompactions
	}
	if dbconfig.MaxBackgroundFlushes > 0 {
		dbOpts.maxBackgroundFlushes = dbconfig.MaxBackgroundFlushes
	}
	if dbconfig.MaxOpenFiles > 0 {
		dbOpts.maxOpenFiles = dbconfig.MaxOpenFiles
	}

	dbconfig.StorePath = filepath.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, chainId, dbFolder)
	rocksdbOpts := dbOpts.ToOptions()

	err := createDirIfNotExist(dbconfig.StorePath)
	if err != nil {
		panic(fmt.Sprintf("Error create dir %s by rocksdbprovider: %s", dbconfig.StorePath, err))
	}
	db, err := gorocksdb.OpenDb(rocksdbOpts, dbconfig.StorePath)
	if err != nil {
		panic(fmt.Sprintf("Error opening %s by rocksdbprovider: %s", dbconfig.StorePath, err))
	}
	logger.Debugf("open rocksdb:%s", dbconfig.StorePath)
	return &RocksDBHandle{
		db:           db,
		readOptions:  gorocksdb.NewDefaultReadOptions(),
		writeOptions: gorocksdb.NewDefaultWriteOptions(),
		logger:       logger,
	}
}

// RocksDBConfig config of rocksdb
type RocksDBConfig struct {
	bloomFilterBits          int
	writeBufferSize          int
	dbWriteBufferSize        int
	maxWriteBufferNumber     int
	maxBackgroundCompactions int
	maxBackgroundFlushes     int
	blockCache               int
	maxOpenFiles             int
}

// NewRocksdbConfig create a new rocksdb config
func NewRocksdbConfig() *RocksDBConfig {
	dbOpts := RocksDBConfig{
		bloomFilterBits:          defaultBloomFilterBits,
		writeBufferSize:          defaultWriteBufferSize * MiB,
		dbWriteBufferSize:        defaultDBWriteBufferSize * MiB,
		maxWriteBufferNumber:     defaultMaxWriteBufferNumber,
		maxBackgroundCompactions: defaultMaxBackgroundCompactions,
		maxBackgroundFlushes:     defaultMaxBackgroundFlushes,
		blockCache:               defaultBlockCacheSize * MiB,
		maxOpenFiles:             defaultMaxOpenFiles}
	return &dbOpts
}

// ToOptions convert rocksdb config to options
func (config *RocksDBConfig) ToOptions() *gorocksdb.Options {
	options := gorocksdb.NewDefaultOptions()
	options.SetCreateIfMissing(true) // 不存在则创建

	options.SetWriteBufferSize(config.writeBufferSize)
	options.SetDbWriteBufferSize(config.dbWriteBufferSize)
	options.SetMaxWriteBufferNumber(config.maxWriteBufferNumber)
	options.SetMaxBackgroundCompactions(config.maxBackgroundCompactions)
	options.SetMaxBackgroundFlushes(config.maxBackgroundFlushes)
	options.SetMaxOpenFiles(config.maxOpenFiles)
	options.SetLevelCompactionDynamicLevelBytes(true)
	options.SetBytesPerSync(1048576)

	blockBasedTableOptions := gorocksdb.NewDefaultBlockBasedTableOptions()
	blockBasedTableOptions.SetBlockSize(16 * KiB)
	blockBasedTableOptions.SetCacheIndexAndFilterBlocks(true)
	blockBasedTableOptions.SetPinL0FilterAndIndexBlocksInCache(true)
	blockBasedTableOptions.SetFormatVersion(4)

	blockBasedTableOptions.SetBlockCache(gorocksdb.NewLRUCache(uint64(config.blockCache)))
	//blockBasedTableOptions.SetFilterPolicy(gorocksdb.NewBloomFilter(config.bloomFilterBits)) // 布隆过滤器
	blockBasedTableOptions.SetBlockCacheCompressed(gorocksdb.NewLRUCache(uint64(config.blockCache)))

	//kNoCompression = 0x0,
	//kSnappyCompression = 0x1,
	//kZlibCompression = 0x2,
	//kBZip2Compression = 0x3,
	//kLZ4Compression = 0x4,
	//kLZ4HCCompression = 0x5,
	//kXpressCompression = 0x6,
	//kZSTD = 0x7,
	//kZSTDNotFinalCompression = 0x40,
	//kDisableCompressionOption = 0xff,
	//options.SetCompression(0x1)

	options.SetBlockBasedTableFactory(blockBasedTableOptions)
	options.SetAllowConcurrentMemtableWrites(true)
	return options
}

func createDirIfNotExist(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		// 创建文件夹
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get returns the value for the given key, or returns nil if none exists
func (h *RocksDBHandle) Get(key []byte) ([]byte, error) {
	value, err := h.db.GetBytes(h.readOptions, key)
	if err != nil {
		h.logger.Errorf("getting rocksdbprovider key [%#v], err:%s", key, err.Error())
		return nil, errors.Wrapf(err, "error getting rocksdbprovider key [%#v]", key)
	}
	return value, nil
}

// Put saves the key-values
func (h *RocksDBHandle) Put(key []byte, value []byte) error {
	if value == nil {
		h.logger.Warn("writing rocksdbprovider key [%#v] with nil value", key)
		return errors.New("error writing rocksdbprovider with nil value")
	}
	err := h.db.Put(h.writeOptions, key, value)
	if err != nil {
		h.logger.Errorf("writing rocksdbprovider key [%#v]", key)
		return errors.Wrapf(err, "error writing rocksdbprovider key [%#v]", key)
	}
	return err
}

// Has return true if the given key exist, or return false if none exists
func (h *RocksDBHandle) Has(key []byte) (bool, error) {
	value, err := h.db.Get(h.readOptions, key)
	if value == nil {
		h.logger.Errorf("can not get rocksdbprovider key [%#v]", key)
		return false, errors.Wrapf(err, "can not get rocksdbprovider key [%#v]", key)
	}
	if err != nil {
		h.logger.Errorf("getting rocksdbprovider key [%#v], err:%s", key, err.Error())
		return false, errors.Wrapf(err, "error getting rocksdbprovider key [%#v]", key)
	}
	return value.Exists(), nil
}

// Delete deletes the given key
func (h *RocksDBHandle) Delete(key []byte) error {
	err := h.db.Delete(h.writeOptions, key)
	if err != nil {
		h.logger.Errorf("deleting rocksdbprovider key [%#v]", key)
		return errors.Wrapf(err, "error deleting rocksdbprovider key [%#v]", key)
	}
	return err
}

// WriteBatch writes a batch in an atomic operation
func (h *RocksDBHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	if batch.Len() == 0 {
		return nil
	}

	startTime := utils.CurrentTimeMillisSeconds()
	writeBatch := gorocksdb.NewWriteBatch()
	elapsedWriteBatchTime := utils.CurrentTimeMillisSeconds() - startTime

	for k, v := range batch.KVs() {
		key := []byte(k)
		if v == nil {
			writeBatch.Delete(key)
		} else {
			writeBatch.Put(key, v)
		}
	}
	elapsedBatchTime := utils.CurrentTimeMillisSeconds() - startTime

	wo := gorocksdb.NewDefaultWriteOptions()
	wo.SetSync(sync)
	elapsedWriteOptionsTime := utils.CurrentTimeMillisSeconds() - startTime

	if err := h.db.Write(wo, writeBatch); err != nil {
		h.logger.Errorf("write batch to rocksdbprovider failed")
		return errors.Wrap(err, "error writing batch to rocksdbprovider")
	}
	elapsedWriteDBTime := utils.CurrentTimeMillisSeconds() - startTime

	h.logger.Debugf("rocksdb write batch time used: newWriteBatchTime: %d, batchTime: %d, writeOptionsTime: %d, writeDBTime: %d, total: %d",
		elapsedWriteBatchTime, elapsedBatchTime - elapsedWriteBatchTime, elapsedWriteOptionsTime - elapsedBatchTime,
		elapsedWriteDBTime - elapsedWriteOptionsTime, utils.CurrentTimeMillisSeconds() - startTime)

	return nil
}

// CompactRange compacts the underlying DB for the given key range.
func (h *RocksDBHandle) CompactRange(start, limit []byte) error {
	h.db.CompactRange(gorocksdb.Range{
		Start: start,
		Limit: limit,
	})

	return nil
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (h *RocksDBHandle) NewIteratorWithRange(startKey []byte, limitKey []byte) protocol.Iterator {
	//startKey := makeKeyWithDbName(h.db.Name(), start)
	//limitKey := makeKeyWithDbName(h.db.Name(), limit)

	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetIterateLowerBound(startKey)
	ro.SetIterateUpperBound(limitKey)
	it := h.db.NewIterator(ro)
	dbIter := NewRocksdbIterator(it)
	dbIter.iter.SeekToFirst()
	return dbIter
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (h *RocksDBHandle) NewIteratorWithPrefix(prefix []byte) protocol.Iterator {
	//prefixKey := makeKeyWithDbName(h.db.Name(), prefix)
	ro := gorocksdb.NewDefaultReadOptions()
	it := h.db.NewIterator(ro)
	it.Seek(prefix)
	return NewRocksdbIterator(it)
}

// Close closes the rocksdb
func (h *RocksDBHandle) Close() error {
	h.db.Close()
	return nil
}

//func makeKeyWithDbName(column string, key []byte) []byte {
//	return append(append([]byte(column), DbNameKeySep...), key...)
//}

func bytesPrefix(prefix []byte) *gorocksdb.Range {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}
	return &gorocksdb.Range{prefix, limit}
}