/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

import "chainmaker.org/chainmaker/protocol/v2"

// UpdateBatch encloses the details of multiple `updates`
type UpdateBatch struct {
	kvs map[string][]byte
}

// NewUpdateBatch constructs an instance of a Batch
func NewUpdateBatch() protocol.StoreBatcher {
	return &UpdateBatch{kvs: make(map[string][]byte)}
}

// KVs return map
func (batch *UpdateBatch) KVs() map[string][]byte {
	return batch.kvs
}

// Put adds a KV
func (batch *UpdateBatch) Put(key []byte, value []byte) {
	if value == nil {
		panic("Nil value not allowed")
	}
	batch.kvs[string(key)] = value
}

// Delete deletes a Key and associated value
func (batch *UpdateBatch) Delete(key []byte) {
	batch.kvs[string(key)] = nil
}

// Len returns the number of entries in the batch
func (batch *UpdateBatch) Len() int {
	return len(batch.kvs)
}

// Merge merges other kvs to this updateBatch
func (batch *UpdateBatch) Merge(u protocol.StoreBatcher) {
	for key, value := range u.KVs() {
		batch.kvs[key] = value
	}
}
