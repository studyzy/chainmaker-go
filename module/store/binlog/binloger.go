/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package binlog

type BinLoger interface {
	Close() error
	TruncateFront(index uint64) error
	Read(index uint64) (data []byte, err error)
	LastIndex() (index uint64, err error)
	Write(index uint64, data []byte) error
}
