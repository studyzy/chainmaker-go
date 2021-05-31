/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLedger(t *testing.T) {
	ledgerCache := NewLedgerCache("Chain1")
	ledgerCache.SetLastCommittedBlock(CreateNewTestBlock(0))
	modulea := &moduleA{}
	moduleb := &moduleB{
		ledgerCache: ledgerCache,
	}
	modulea.setLedgerCache(ledgerCache)
	b := ledgerCache.GetLastCommittedBlock()
	b.Header.BlockHeight = 100
	ledgerCache.SetLastCommittedBlock(b)
	require.Equal(t, int64(100), modulea.getBlock().Header.BlockHeight)
	b = modulea.getBlock()
	b.Header.BlockHeight = 200
	modulea.updateBlock(b)
	require.Equal(t, int64(200), moduleb.getBlock().Header.BlockHeight)
}

type moduleA struct {
	ledgerCache protocol.LedgerCache
}

func (m *moduleA) setLedgerCache(cache protocol.LedgerCache) {
	m.ledgerCache = cache
}

func (m *moduleA) updateBlock(block *commonpb.Block) {
	m.ledgerCache.SetLastCommittedBlock(block)
}

func (m *moduleA) getBlock() *commonpb.Block {
	return m.ledgerCache.GetLastCommittedBlock()
}

type moduleB struct {
	ledgerCache protocol.LedgerCache
}

func (m *moduleB) updateBlock(block *commonpb.Block) {
	m.ledgerCache.SetLastCommittedBlock(block)
}

func (m *moduleB) getBlock() *commonpb.Block {
	return m.ledgerCache.GetLastCommittedBlock()
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
			TxCount:        1,
			Signature:      []byte(""),
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