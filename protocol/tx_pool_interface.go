/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/common"
)

// TxPool Manage pending transactions and update the current status of
// transactions (pending packages, pending warehousing, pending retries, etc.)
type TxPool interface {
	// Start start the txPool service
	Start() error
	// Stop stop the txPool service
	Stop() error

	// AddTx Add a transaction to the txPool
	// There are three types of Source (RPC/P2P/INTERNAL), which different checks
	// are performed for different types of cases
	AddTx(tx *common.Transaction, source TxSource) error
	// GetTxByTxId Retrieve the transaction by the txId from the txPool
	GetTxByTxId(txId string) (tx *common.Transaction, inBlockHeight int64)
	// IsTxExistInPool verifies whether the transaction exists in the tx_pool
	TxExists(tx *common.Transaction) bool
	// GetTxsByTxIds Retrieves the tx by the txIds from the tx pool.
	// txsRet if the transaction is in the tx pool, it will be returned in txsRet.
	// txsHeightRet if the transaction is in the pending queue of the tx pool,
	// the corresponding block height when the transaction entered the block is returned,
	// if the transaction is in the normal queue of the tx pool, the tx height is 0,
	// if the transaction is not in the transaction pool, the tx height is -1.
	GetTxsByTxIds(txIds []string) (txsRet map[string]*common.Transaction, txsHeightRet map[string]int64)
	// RetryAndRemove Process transactions within multiple proposed blocks at the same height to
	// ensure that these transactions are not lost, re-add valid txs which that are not on local node.
	// remove txs in the commit block.
	RetryAndRemoveTxs(retryTxs []*common.Transaction, removeTxs []*common.Transaction)
	// FetchTxBatch Get the batch of transactions from the tx pool to generate new block
	FetchTxBatch(blockHeight int64) []*common.Transaction
	// AddTxsToPendingCache These transactions will be added to the cache to avoid the transactions
	// are fetched again and re-filled into the new block. Because Because of the chain confirmation
	// rule in the HotStuff consensus algorithm.
	AddTxsToPendingCache(txs []*common.Transaction, blockHeight int64)
}

type TxSource int

const (
	RPC TxSource = iota
	P2P
	INTERNAL
)
