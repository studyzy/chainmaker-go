/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historydb

import (
	"chainmaker.org/chainmaker-go/store/serialization"
)

// HistoryDB provides handle to rwSets instances
type HistoryDB interface {
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error
	// CommitBlock commits the block rwsets in an atomic operation
	CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error

	//GetHistoryForKey 获得Key的交易历史
	GetHistoryForKey(contractName string, key []byte) (HistoryIterator, error)
	GetAccountTxHistory(account []byte) (HistoryIterator, error)
	GetContractTxHistory(contractName string) (HistoryIterator, error)
	// GetLastSavepoint returns the last block height
	GetLastSavepoint() (uint64, error)

	// Close is used to close database
	Close()
}
type BlockHeightTxId struct {
	BlockHeight uint64
	TxId        string
}
type HistoryIterator interface {
	Next() bool
	Value() (*BlockHeightTxId, error)
	Release()
}
