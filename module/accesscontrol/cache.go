/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"sync"
	"sync/atomic"
)

type cacheItem struct {
	key   string
	value interface{}

	// set to 1 when accessed, and set to 0 when scanned
	referenced int32
}

type simpleCache struct {
	// maintains the map from key to value
	table map[string]*cacheItem

	// holds a list of cached items.
	items []*cacheItem

	// stores the position to be scanned next time
	position int

	// read lock for get, and write lock for add
	lock sync.RWMutex

	isEnabled bool
}

func newSimpleCache(cacheSize int) *simpleCache {
	var cache simpleCache
	cache.isEnabled = true
	if cacheSize <= 0 {
		cacheSize = 0
		cache.isEnabled = false
		return &cache
	}
	cache.position = 0
	cache.items = make([]*cacheItem, cacheSize)
	cache.table = map[string]*cacheItem{}

	return &cache
}

func (cache *simpleCache) get(key string) (interface{}, bool) {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	if !cache.isEnabled {
		return nil, false
	}

	item, ok := cache.table[key]
	if !ok {
		return nil, false
	}

	// when accessed, this flag is set to 1
	atomic.StoreInt32(&item.referenced, 1)

	return item.value, true
}

func (cache *simpleCache) add(key string, value interface{}) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if !cache.isEnabled {
		return
	}

	if old, ok := cache.table[key]; ok {
		old.value = value
		atomic.StoreInt32(&old.referenced, 1)
		return
	}

	var item cacheItem
	item.key = key
	item.value = value
	atomic.StoreInt32(&item.referenced, 1)

	size := len(cache.items)
	num := len(cache.table)
	if num < size {
		// not full yet
		cache.table[key] = &item
		cache.items[num] = &item
		return
	}

	// scan for an item to delete (delete when flag is set to 0)
	for {
		victim := cache.items[cache.position]
		if atomic.LoadInt32(&victim.referenced) == 0 {
			// found one with flag 0, can delete this and store the new item at place
			delete(cache.table, victim.key)
			cache.table[key] = &item
			cache.items[cache.position] = &item
			cache.position = (cache.position + 1) % size
			return
		}

		// flag is 1, so set the flag to 0 indicating it can be deleted the next time
		atomic.StoreInt32(&victim.referenced, 0)
		cache.position = (cache.position + 1) % size
	}
}

func (cache *simpleCache) reset() {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if !cache.isEnabled {
		return
	}

	cacheSize := len(cache.items)
	cache.position = 0
	cache.items = make([]*cacheItem, cacheSize)
	cache.table = map[string]*cacheItem{}
}
