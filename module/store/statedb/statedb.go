/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
)

// StateDB provides handle to world state instances
type StateDB interface {

	// CommitBlock commits the state in an atomic operation
	CommitBlock(blockWithRWSet *storePb.BlockWithRWSet) error

	// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
	ReadObject(contractName string, key []byte) ([]byte, error)

	// SelectObject returns an iterator that contains all the key-values between given key ranges.
	// startKey is included in the results and limit is excluded.
	SelectObject(contractName string, startKey []byte, limit []byte) protocol.Iterator

	// GetLastSavepoint returns the last block height
	//GetLastSavepoint() (uint64, error)

	// Close is used to close database
	Close()
}
