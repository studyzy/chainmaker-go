/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockdb

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
)

// BlockDB provides handle to block and tx instances
type BlockDB interface {
	//
	SaveBlockHeader(header *commonPb.BlockHeader) error
	//GetBlockHeaderByHash(blockHash []byte) (*commonPb.BlockHeader, error)
	//GetBlockHeaderByHeight(blockHash []byte) (*commonPb.BlockHeader, error)

	// CommitBlock commits the block and the corresponding rwsets in an atomic operation
	CommitBlock(block *commonPb.Block) error

	// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
	GetBlockByHash(blockHash []byte) (*commonPb.Block, error)

	// BlockExists returns true if the block hash exist, or returns false if none exists.
	BlockExists(blockHash []byte) (bool, error)

	// GetBlock returns a block given it's block height, or returns nil if none exists.
	GetBlock(height uint64) (*commonPb.Block, error)

	// GetTx retrieves a transaction by txid, or returns nil if none exists.
	GetTx(txId string) (*commonPb.Transaction, error)

	// TxExists returns true if the tx exist, or returns false if none exists.
	TxExists(txId string) (bool, error)

	// GetTxConfirmedTime retrieves time of the tx confirmed in the blockChain
	GetTxConfirmedTime(txId string) (int64, error)

	// GetLastBlock returns the last block.
	GetLastBlock() (*commonPb.Block, error)

	// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
	GetFilteredBlock(height int64) (*storePb.SerializedBlock, error)

	// GetLastSavepoint reurns the last block height
	//GetLastSavepoint() (uint64, error)

	// GetLastConfigBlock returns the last config block.
	GetLastConfigBlock() (*commonPb.Block, error)

	// GetBlockByTx returns a block which contains a tx.
	GetBlockByTx(txId string) (*commonPb.Block, error)

	// Close is used to close database
	Close()
}
