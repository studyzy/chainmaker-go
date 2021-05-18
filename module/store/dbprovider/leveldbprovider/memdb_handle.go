/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package leveldbprovider

import (
	"bytes"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type MemdbHandle struct {
	db *memdb.DB
}

type bytesComparer struct{}

func (bytesComparer) Compare(a, b []byte) int {
	return bytes.Compare(a, b)
}
func NewMemdbHandle() *MemdbHandle {
	return &MemdbHandle{db: memdb.New(&bytesComparer{}, 1000)}
}
func (db *MemdbHandle) Get(key []byte) ([]byte, error) {
	return db.db.Get(key)
}

// Put saves the key-values
func (db *MemdbHandle) Put(key []byte, value []byte) error {
	return db.db.Put(key, value)
}

// Has return true if the given key exist, or return false if none exists
func (db *MemdbHandle) Has(key []byte) (bool, error) {
	return db.db.Contains(key), nil
}

// Delete deletes the given key
func (db *MemdbHandle) Delete(key []byte) error {
	return db.db.Delete(key)
}

// WriteBatch writes a batch in an atomic operation
func (db *MemdbHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	for k, v := range batch.KVs() {
		db.db.Put([]byte(k), v)
	}
	return nil
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (db *MemdbHandle) NewIteratorWithRange(start []byte, limit []byte) protocol.Iterator {
	return db.db.NewIterator(&util.Range{Start: start, Limit: limit})
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (db *MemdbHandle) NewIteratorWithPrefix(prefix []byte) protocol.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix))
}
func (db *MemdbHandle) Close() error {
	db.db.Reset()
	return nil
}
