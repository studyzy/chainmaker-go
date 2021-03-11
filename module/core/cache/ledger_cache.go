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

func CreateNewTestBlock(height int64) *commonpb.Block {
	var hash = []byte("0123456789")
	var version = []byte("0")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "Chain1",
			BlockHeight:    height,
			PreBlockHash:   hash,
			BlockHash:      hash,
			PreConfHeight:  0,
			BlockVersion:   version,
			DagHash:        hash,
			RwSetRoot:      hash,
			TxRoot:         hash,
			BlockTimestamp: 0,
			Proposer:       hash,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}
	tx := CreateNewTestTx()
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}

func CreateNewTestTx() *commonpb.Transaction {
	var hash = []byte("0123456789")
	return &commonpb.Transaction{
		Header: &commonpb.TxHeader{
			ChainId:        "",
			Sender:         nil,
			TxType:         0,
			TxId:           "",
			Timestamp:      0,
			ExpirationTime: 0,
		},
		RequestPayload:   hash,
		RequestSignature: hash,
		Result: &commonpb.Result{
			Code:           commonpb.TxStatusCode_SUCCESS,
			ContractResult: nil,
			RwSetHash:      nil,
		},
	}
}
