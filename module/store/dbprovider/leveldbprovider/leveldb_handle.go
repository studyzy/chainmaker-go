/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package leveldbprovider

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const defaultBloomFilterBits = 10
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

// LevelDBHandle encapsulated handle to leveldb
type LevelDBHandle struct {
	writeLock sync.Mutex
	db        *leveldb.DB
	logger    protocol.Logger
}

func NewLevelDBHandle(chainId string, dbFolder string, dbconfig *localconf.LevelDbConfig,
	logger protocol.Logger) *LevelDBHandle {
	dbOpts := &opt.Options{}
	writeBufferSize := dbconfig.BlockWriteBufferSize
	if writeBufferSize <= 0 {
		//default value 4MB
		dbOpts.WriteBuffer = 4 * opt.MiB
	} else {
		dbOpts.WriteBuffer = writeBufferSize * opt.MiB
	}
	bloomFilterBits := dbconfig.BloomFilterBits
	if bloomFilterBits <= 0 {
		bloomFilterBits = defaultBloomFilterBits
	}
	dbOpts.Filter = filter.NewBloomFilter(bloomFilterBits)
	dbPath := filepath.Join(dbconfig.StorePath, chainId, dbFolder)
	err := createDirIfNotExist(dbPath)
	if err != nil {
		panic(fmt.Sprintf("Error create dir %s by leveldbprovider: %s", dbPath, err))
	}
	db, err := leveldb.OpenFile(dbPath, dbOpts)
	if err != nil {
		panic(fmt.Sprintf("Error opening %s by leveldbprovider: %s", dbPath, err))
	}
	logger.Debugf("open leveldb:%s", dbPath)
	return &LevelDBHandle{
		db:     db,
		logger: logger,
	}
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
func (h *LevelDBHandle) Get(key []byte) ([]byte, error) {
	value, err := h.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		value = nil
		err = nil
	}
	if err != nil {
		h.logger.Errorf("getting leveldbprovider key [%#v], err:%s", key, err.Error())
		return nil, errors.Wrapf(err, "error getting leveldbprovider key [%#v]", key)
	}
	return value, nil
}

// Put saves the key-values
func (h *LevelDBHandle) Put(key []byte, value []byte) error {
	if value == nil {
		h.logger.Warn("writing leveldbprovider key [%#v] with nil value", key)
		return errors.New("error writing leveldbprovider with nil value")
	}
	wo := &opt.WriteOptions{Sync: true}
	err := h.db.Put(key, value, wo)
	if err != nil {
		h.logger.Errorf("writing leveldbprovider key [%#v]", key)
		return errors.Wrapf(err, "error writing leveldbprovider key [%#v]", key)
	}
	return err
}

// Has return true if the given key exist, or return false if none exists
func (h *LevelDBHandle) Has(key []byte) (bool, error) {
	exist, err := h.db.Has(key, nil)
	if err != nil {
		h.logger.Errorf("getting leveldbprovider key [%#v], err:%s", key, err.Error())
		return false, errors.Wrapf(err, "error getting leveldbprovider key [%#v]", key)
	}
	return exist, nil
}

// Delete deletes the given key
func (h *LevelDBHandle) Delete(key []byte) error {
	wo := &opt.WriteOptions{Sync: true}
	err := h.db.Delete(key, wo)
	if err != nil {
		h.logger.Errorf("deleting leveldbprovider key [%#v]", key)
		return errors.Wrapf(err, "error deleting leveldbprovider key [%#v]", key)
	}
	return err
}

// WriteBatch writes a batch in an atomic operation
func (h *LevelDBHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	if batch.Len() == 0 {
		return nil
	}
	h.writeLock.Lock()
	defer h.writeLock.Unlock()
	levelBatch := &leveldb.Batch{}
	for k, v := range batch.KVs() {
		key := []byte(k)
		if v == nil {
			levelBatch.Delete(key)
		} else {
			levelBatch.Put(key, v)
		}
	}

	wo := &opt.WriteOptions{Sync: sync}
	if err := h.db.Write(levelBatch, wo); err != nil {
		h.logger.Errorf("write batch to leveldb provider failed")
		return errors.Wrap(err, "error writing batch to leveldb provider")
	}
	return nil
}

// CompactRange compacts the underlying DB for the given key range.
func (h *LevelDBHandle) CompactRange(start, limit []byte) error {
	return h.db.CompactRange(util.Range{
		Start: start,
		Limit: limit,
	})
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (h *LevelDBHandle) NewIteratorWithRange(startKey []byte, limitKey []byte) (protocol.Iterator, error) {
	if len(startKey) == 0 || len(limitKey) == 0 {
		return nil, fmt.Errorf("iterator range should not start(%s) or limit(%s) with empty key",
			string(startKey), string(limitKey))
	}
	keyRange := &util.Range{Start: startKey, Limit: limitKey}
	return h.db.NewIterator(keyRange, nil), nil
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (h *LevelDBHandle) NewIteratorWithPrefix(prefix []byte) (protocol.Iterator, error) {
	if len(prefix) == 0 {
		return nil, fmt.Errorf("iterator prefix should not be empty key")
	}

	return h.db.NewIterator(util.BytesPrefix(prefix), nil), nil
}

// Close closes the leveldb
func (h *LevelDBHandle) Close() error {
	h.writeLock.Lock()
	defer h.writeLock.Unlock()
	return h.db.Close()
}
