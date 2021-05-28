/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package ledger is cache for current block and proposal blocks
package cache

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"errors"
	"sync"
)

// LedgerCache is used for cache current block info
type LedgerCache struct {
	chainId            string
	lastCommittedBlock *commonpb.Block
	rwMu               sync.RWMutex
}

// NewLedgerCache get a ledger cache.
// One ledger cache for one chain.
func NewLedgerCache(chainId string) protocol.LedgerCache {
	return &LedgerCache{
		chainId: chainId,
	}
}

// GetLastCommittedBlock get the latest committed block
func (lc *LedgerCache) GetLastCommittedBlock() *commonpb.Block {
	lc.rwMu.RLock()
	defer lc.rwMu.RUnlock()
	return lc.lastCommittedBlock
}

// SetLastCommittedBlock set the latest committed block
func (lc *LedgerCache) SetLastCommittedBlock(b *commonpb.Block) {
	lc.rwMu.Lock()
	defer lc.rwMu.Unlock()
	lc.lastCommittedBlock = b
}

// CurrentHeight get current block height
func (lc *LedgerCache) CurrentHeight() (int64, error) {
	lc.rwMu.RLock()
	defer lc.rwMu.RUnlock()
	if lc.lastCommittedBlock == nil {
		return -1, errors.New("last committed block == nil")
	}
	return lc.lastCommittedBlock.Header.BlockHeight, nil
}

