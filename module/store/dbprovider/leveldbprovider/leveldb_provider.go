/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package leveldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"

	//"chainmaker.org/chainmaker-go/store/dbprovider/kvdbprovider"
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
	"sync"
)

const defaultBloomFilterBits = 10

const (
	StoreBlockDBDir   = "store_block"
	StoreStateDBDir   = "store_state"
	StoreHistoryDBDir = "store_history"
)

var DbNameKeySep = []byte{0x00}

// Provider provides handle to db instances
type Provider struct {
	db        *leveldb.DB
	dbHandles map[string]*LevelDBHandle
	mutex     sync.Mutex

	logger *logImpl.CMLogger
}

// NewBlockProvider construct a new Provider for block operation with given chainId
func NewBlockProvider(chainId string) *Provider {
	return NewProvider(chainId, StoreBlockDBDir)
}

// NewStateProvider construct a new Provider for state operation with given chainId
func NewStateProvider(chainId string) *Provider {
	return NewProvider(chainId, StoreStateDBDir)
}

// NewStateProvider construct a new Provider for state operation with given chainId
func NewHistoryProvider(chainId string) *Provider {
	return NewProvider(chainId, StoreHistoryDBDir)
}

// NewProvider construct a new db Provider for given chainId and dir
func NewProvider(chainId, dbDir string) *Provider {
	dbOpts := &opt.Options{}
	writeBufferSize := localconf.ChainMakerConfig.StorageConfig.BlockWriteBufferSize
	if writeBufferSize <= 0 {
		//default value 4MB
		dbOpts.WriteBuffer = 4 * opt.MiB
	} else {
		dbOpts.WriteBuffer = writeBufferSize * opt.MiB
	}
	bloomFilterBits := localconf.ChainMakerConfig.StorageConfig.BloomFilterBits
	if bloomFilterBits <= 0 {
		bloomFilterBits = defaultBloomFilterBits
	}
	dbOpts.Filter = filter.NewBloomFilter(bloomFilterBits)
	dbPath := filepath.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, chainId, dbDir)
	db, err := leveldb.OpenFile(dbPath, dbOpts)
	if err != nil {
		panic(fmt.Sprintf("Error opening leveldbprovider: %s", err))
	}
	return &Provider{
		db:        db,
		dbHandles: make(map[string]*LevelDBHandle),
		mutex:     sync.Mutex{},

		logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
	}
}

// GetDBHandle returns a DBHandle for given dbname
func (p *Provider) GetDBHandle(dbName string) protocol.DBHandle {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	dbHandle := p.dbHandles[dbName]
	if dbHandle == nil {
		dbHandle = &LevelDBHandle{dbName: dbName, db: p.db, logger: logImpl.GetLogger(logImpl.MODULE_STORAGE)}
		p.dbHandles[dbName] = dbHandle
	}
	return dbHandle
}

// Close is used to close database
func (p *Provider) Close() error {
	err := p.db.Close()
	if err != nil {
		p.logger.Errorf("close leveldbprovider, err:%s", err.Error())
	}
	return err
}

// LevelDBHandle encapsulated handle to leveldb
type LevelDBHandle struct {
	dbName string
	db     *leveldb.DB

	logger *logImpl.CMLogger
}

// GetDbName returns associated dbname with this handle
func (h *LevelDBHandle) GetDbName() string {
	return h.dbName
}

// Get returns the value for the given key, or returns nil if none exists
func (h *LevelDBHandle) Get(key []byte) ([]byte, error) {
	value, err := h.db.Get(makeKeyWithDbName(h.dbName, key), nil)
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
		h.logger.Warn("writting leveldbprovider key [%#v] with nil value", key)
		return errors.New("error writing leveldbprovider with nil value")
	}
	wo := &opt.WriteOptions{Sync: true}
	err := h.db.Put(makeKeyWithDbName(h.dbName, key), value, wo)
	if err != nil {
		h.logger.Errorf("writing leveldbprovider key [%#v]", key)
		return errors.Wrapf(err, "error writing leveldbprovider key [%#v]", key)
	}
	return err
}

// Has return true if the given key exist, or return false if none exists
func (h *LevelDBHandle) Has(key []byte) (bool, error) {
	exist, err := h.db.Has(makeKeyWithDbName(h.dbName, key), nil)
	if err != nil {
		h.logger.Errorf("getting leveldbprovider key [%#v], err:%s", key, err.Error())
		return false, errors.Wrapf(err, "error getting leveldbprovider key [%#v]", key)
	}
	return exist, nil
}

// Delete deletes the given key
func (h *LevelDBHandle) Delete(key []byte) error {
	wo := &opt.WriteOptions{Sync: true}
	err := h.db.Delete(makeKeyWithDbName(h.dbName, key), wo)
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
	levelBatch := &leveldb.Batch{}
	for k, v := range batch.KVs() {
		key := makeKeyWithDbName(h.dbName, []byte(k))
		if v == nil {
			levelBatch.Delete(key)
		} else {
			levelBatch.Put(key, v)
		}
	}

	wo := &opt.WriteOptions{Sync: sync}
	if err := h.db.Write(levelBatch, wo); err != nil {
		h.logger.Errorf("write batch to leveldbprovider failed")
		return errors.Wrap(err, "error writing batch to leveldbprovider")
	}
	return nil
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (h *LevelDBHandle) NewIteratorWithRange(start []byte, limit []byte) protocol.Iterator {
	startKey := makeKeyWithDbName(h.dbName, start)
	limitKey := makeKeyWithDbName(h.dbName, limit)
	keyRange := &util.Range{Start: startKey, Limit: limitKey}
	return h.db.NewIterator(keyRange, nil)
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (h *LevelDBHandle) NewIteratorWithPrefix(prefix []byte) protocol.Iterator {
	return h.db.NewIterator(util.BytesPrefix(prefix), nil)
}

// Close closes the leveldb
func (h *LevelDBHandle) Close() error {
	return h.db.Close()
}

func makeKeyWithDbName(column string, key []byte) []byte {
	return append(append([]byte(column), DbNameKeySep...), key...)
}
