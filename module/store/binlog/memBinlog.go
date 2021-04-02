/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package binlog

import "errors"

type MemBinlog struct {
	mem  map[uint64][]byte
	last uint64
}

func NewMemBinlog() *MemBinlog {
	return &MemBinlog{
		mem:  make(map[uint64][]byte),
		last: 0,
	}
}

func (l *MemBinlog) Close() error {
	return nil
}
func (l *MemBinlog) TruncateFront(index uint64) error {
	return nil
}
func (l *MemBinlog) Read(index uint64) ([]byte, error) {
	return l.mem[index], nil
}
func (l *MemBinlog) LastIndex() (uint64, error) {
	return l.last, nil
}
func (l *MemBinlog) Write(index uint64, data []byte) error {
	if index != l.last+1 {
		return errors.New("binlog out of order")
	}
	l.mem[index] = data
	l.last = index
	return nil
}
