/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"context"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/emirpasic/gods/maps/treemap"
	"golang.org/x/sync/semaphore"
)

const defaultMaxBlockSize = 10

// StoreCacheMgr provide handle to cache instances
type StoreCacheMgr struct {
	sync.RWMutex
	pendingBlockUpdates map[int64]protocol.StoreBatcher
	blockSizeSem        *semaphore.Weighted
	cache               *storeCache
	cacheSize           int //block size in cache, if cache size <= 0, use defalut size = 10

	logger protocol.Logger
}

// NewStoreCacheMgr construct a new `StoreCacheMgr` with given chainId
func NewStoreCacheMgr(chainId string, logger protocol.Logger) *StoreCacheMgr {
	blockWriteBufferSize := localconf.ChainMakerConfig.StorageConfig.BlockWriteBufferSize
	if blockWriteBufferSize <= 0 {
		blockWriteBufferSize = defaultMaxBlockSize
	}
	storeCacheMgr := &StoreCacheMgr{
		pendingBlockUpdates: make(map[int64]protocol.StoreBatcher),
		blockSizeSem:        semaphore.NewWeighted(int64(blockWriteBufferSize)),
		cache:               newStoreCache(),
		cacheSize:           blockWriteBufferSize,
		logger:              logger,
	}
	return storeCacheMgr
}

// AddBlock cache a block with given block height and update batch
func (mgr *StoreCacheMgr) AddBlock(blockHeight int64, updateBatch protocol.StoreBatcher) {
	//wait for semaphore
	mgr.blockSizeSem.Acquire(context.Background(), 1)
	mgr.Lock()
	defer mgr.Unlock()
	mgr.pendingBlockUpdates[blockHeight] = updateBatch

	//update cache
	mgr.cache.addBatch(updateBatch)
	mgr.logger.Debugf("add block[%d] to cache, block size:%d", blockHeight, mgr.getPendingBlockSize())
}

// DelBlock delete block for the given block height
func (mgr *StoreCacheMgr) DelBlock(blockHeight int64) {
	//release semaphore
	mgr.blockSizeSem.Release(1)
	mgr.Lock()
	defer mgr.Unlock()
	batch, exist := mgr.pendingBlockUpdates[blockHeight]
	if !exist {
		return
	}
	mgr.cache.delBatch(batch)
	delete(mgr.pendingBlockUpdates, blockHeight)
	mgr.logger.Debugf("del block[%d] from cache, block size:%d", blockHeight, mgr.getPendingBlockSize())
}

// Get returns value if the key in cache, or returns nil if none exists.
func (mgr *StoreCacheMgr) Get(key string) ([]byte, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.cache.get(key)
}

// Has returns true if the key in cache, or returns false if none exists.
func (mgr *StoreCacheMgr) Has(key string) (bool, bool) {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.cache.has(key)
}

// LockForFlush used to lock cache until all cache item be flushed to db
func (mgr *StoreCacheMgr) LockForFlush() {
	mgr.blockSizeSem.Acquire(context.Background(), defaultMaxBlockSize)
}

// UnLockFlush used to unlock cache by release all semaphore
func (mgr *StoreCacheMgr) UnLockFlush() {
	mgr.blockSizeSem.Release(defaultMaxBlockSize)
}

func (mgr *StoreCacheMgr) getPendingBlockSize() int {
	return len(mgr.pendingBlockUpdates)
}

type storeCache struct {
	table *treemap.Map
}

func newStoreCache() *storeCache {
	storeCache := &storeCache{
		table: treemap.NewWithStringComparator(),
	}
	return storeCache
}

func (c *storeCache) addBatch(batch protocol.StoreBatcher) {
	for key, value := range batch.KVs() {
		c.table.Put(key, value)
	}
}

func (c *storeCache) delBatch(batch protocol.StoreBatcher) {
	for key := range batch.KVs() {
		c.table.Remove(key)
	}
}

func (c *storeCache) get(key string) ([]byte, bool) {
	if value, exist := c.table.Get(key); exist {
		result, ok := value.([]byte)
		if !ok {
			panic("type err: value is not []byte")
		}
		return result, true
	}
	return nil, false
}

// Has returns (isDelete, exist)
// if key exist in cache, exist = true
// if key exist in cache and value == nil, isDelete = true
func (c *storeCache) has(key string) (bool, bool) {
	value, exist := c.get(key)
	var isDelete bool
	if exist {
		isDelete = value == nil
		return isDelete, true
	} else {
		return false, false
	}
}

//func (c *storeCache) len() int {
//	return c.table.Size()
//}
