/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package leveldbprovider

import (
	"bytes"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/syndtr/goleveldb/leveldb/memdb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type MemdbHandle struct {
	db     *memdb.DB
	closed bool
}

var errClosed = errors.New("leveldb: closed")

type bytesComparer struct{}

func (bytesComparer) Compare(a, b []byte) int {
	return bytes.Compare(a, b)
}
func NewMemdbHandle() *MemdbHandle {
	return &MemdbHandle{db: memdb.New(&bytesComparer{}, 1000)}
}
func (db *MemdbHandle) Get(key []byte) ([]byte, error) {
	if db.closed {
		return nil, errClosed
	}
	return db.db.Get(key)
}

// Put saves the key-values
func (db *MemdbHandle) Put(key []byte, value []byte) error {
	if db.closed {
		return errClosed
	}
	return db.db.Put(key, value)
}

// Has return true if the given key exist, or return false if none exists
func (db *MemdbHandle) Has(key []byte) (bool, error) {
	if db.closed {
		return false, errClosed
	}
	return db.db.Contains(key), nil
}

// Delete deletes the given key
func (db *MemdbHandle) Delete(key []byte) error {
	if db.closed {
		return errClosed
	}
	return db.db.Delete(key)
}

// WriteBatch writes a batch in an atomic operation
func (db *MemdbHandle) WriteBatch(batch protocol.StoreBatcher, sync bool) error {
	if db.closed {
		return errClosed
	}
	for k, v := range batch.KVs() {
		if err := db.db.Put([]byte(k), v); err != nil {
			return err
		}
	}
	return nil
}

// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
// start is included in the results and limit is excluded.
func (db *MemdbHandle) NewIteratorWithRange(start []byte, limit []byte) (protocol.Iterator, error) {
	if len(start) == 0 || len(limit) == 0 {
		return nil, fmt.Errorf("iterator range should not start(%s) or limit(%s) with empty key",
			string(start), string(limit))
	}
	if db.closed {
		return nil, errClosed
	}
	return db.db.NewIterator(&util.Range{Start: start, Limit: limit}), nil
}

// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
func (db *MemdbHandle) NewIteratorWithPrefix(prefix []byte) (protocol.Iterator, error) {
	if len(prefix) == 0 {
		return nil, fmt.Errorf("iterator prefix should not be empty key")
	}
	if db.closed {
		return nil, errClosed
	}
	return db.db.NewIterator(util.BytesPrefix(prefix)), nil
}
func (db *MemdbHandle) CompactRange(start []byte, limit []byte) error {
	return nil
}

func (db *MemdbHandle) Close() error {
	if db.closed {
		return errClosed
	}
	db.db.Reset()
	db.closed = true
	return nil
}
