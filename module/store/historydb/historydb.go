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

	// CommitBlock commits the block rwsets in an atomic operation
	CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error

	// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
	//GetTxRWSet(contractName string,key []byte) ([], error)

	// GetLastSavepoint returns the last block height
	GetLastSavepoint() (uint64, error)

	// Close is used to close database
	Close()
}
