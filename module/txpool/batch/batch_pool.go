/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package batch

import (
	"sync"

	"chainmaker.org/chainmaker-go/common/sortedmap"
	txpoolPb "chainmaker.org/chainmaker-go/pb/protogo/txpool"
)

type nodeBatchPool struct {
	pool *sortedmap.IntKeySortedMap
}

func newNodeBatchPool() *nodeBatchPool {
	return &nodeBatchPool{pool: sortedmap.NewIntKeySortedMap()}
}

func (p *nodeBatchPool) PutIfNotExist(batch *txpoolPb.TxBatch) bool {
	batchId := int(batch.BatchId)
	ok := p.pool.Contains(batchId)
	if ok {
		return false
	}
	p.pool.Put(batchId, batch)
	return true
}

func (p *nodeBatchPool) RemoveIfExist(batch *txpoolPb.TxBatch) bool {
	batchId := int(batch.BatchId)
	_, ok := p.pool.Remove(batchId)
	return ok
}

func (p *nodeBatchPool) currentSize() int {
	return p.pool.Length()
}

type pendingBatchPool struct {
	l    sync.RWMutex
	pool map[int32]*txpoolPb.TxBatch
}

func newPendingBatchPool() *pendingBatchPool {
	return &pendingBatchPool{pool: make(map[int32]*txpoolPb.TxBatch)}
}

func (p *pendingBatchPool) PutIfNotExist(batch *txpoolPb.TxBatch) bool {
	p.l.Lock()
	defer p.l.Unlock()
	batchId := batch.BatchId
	_, ok := p.pool[batchId]
	if !ok {
		p.pool[batchId] = batch
		return true
	}
	return false
}

func (p *pendingBatchPool) RemoveIfExist(batch *txpoolPb.TxBatch) bool {
	p.l.Lock()
	defer p.l.Unlock()
	batchId := batch.BatchId
	_, ok := p.pool[batchId]
	if ok {
		delete(p.pool, batchId)
		return true
	}
	return false
}

func (p *pendingBatchPool) Range(f func(batch *txpoolPb.TxBatch) (isContinue bool)) {
	p.l.RLock()
	defer p.l.RUnlock()
	for _, batch := range p.pool {
		if !f(batch) {
			break
		}
	}
}

func (p *pendingBatchPool) currentSize() int {
	p.l.RLock()
	defer p.l.RUnlock()
	return len(p.pool)
}

type cfgBatchPool struct {
	pool *sortedmap.IntKeySortedMap
}

func newCfgBatchPool() *cfgBatchPool {
	return &cfgBatchPool{pool: sortedmap.NewIntKeySortedMap()}
}

func (p *cfgBatchPool) PutIfNotExist(batch *txpoolPb.TxBatch) bool {
	batchId := int(batch.BatchId)
	ok := p.pool.Contains(batchId)
	if ok {
		return false
	}
	p.pool.Put(batchId, batch)
	return true
}

func (p *cfgBatchPool) RemoveIfExist(batch *txpoolPb.TxBatch) bool {
	batchId := int(batch.BatchId)
	_, ok := p.pool.Remove(batchId)
	return ok
}

func (p *cfgBatchPool) currentSize() int {
	return p.pool.Length()
}
