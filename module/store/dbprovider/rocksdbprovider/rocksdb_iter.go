// +build rocksdb

// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rocksdbprovider

import (
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/yiyanwannian/gorocksdb"
)

type rocksdbIter struct {
	first bool
	iter  *gorocksdb.Iterator
}

func NewRocksdbIterator(iter *gorocksdb.Iterator) *rocksdbIter {
	return &rocksdbIter{first: true, iter: iter}
}

func (i *rocksdbIter) Valid() bool {
	return i.iter.Valid()
}

func (i *rocksdbIter) First() bool {
	i.iter.SeekToFirst()
	return i.Valid()
}

func (i *rocksdbIter) Last() bool {
	i.iter.SeekToLast()
	return i.Valid()
}

func (i *rocksdbIter) Seek(key []byte) bool {
	i.iter.Seek(key)
	return i.Valid()
}

func (i *rocksdbIter) Next() bool {
	if !i.Valid() {
		return false
	}

	if i.first {
		i.first = false
		return true
	}

	i.iter.Next()
	return i.iter.Valid()
}

func (i *rocksdbIter) Prev() bool {
	if !i.Valid() {
		return false
	}

	if i.first {
		i.first = false
		return true
	}

	i.iter.Prev()
	return true
}

func (i *rocksdbIter) Key() []byte {
	if i.first {
		i.first = false
		return []byte("")
	}
	return i.iter.Key().Data()
}

func (i *rocksdbIter) Value() []byte {
	if i.first {
		i.first = false
		return []byte("")
	}
	return i.iter.Value().Data()
}

func (i *rocksdbIter) Release() {
	i.iter.Close()
}

func (i *rocksdbIter) SetReleaser(releaser util.Releaser) {
}

func (i *rocksdbIter) Error() error {
	return i.iter.Err()
}