/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statedb

import (
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker/protocol"
)

// StateDB provides handle to world state instances
type StateDB interface {
	InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error
	// CommitBlock commits the state in an atomic operation
	CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo) error

	// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
	ReadObject(contractName string, key []byte) ([]byte, error)

	// SelectObject returns an iterator that contains all the key-values between given key ranges.
	// startKey is included in the results and limit is excluded.
	SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error)

	// GetLastSavepoint returns the last block height
	GetLastSavepoint() (uint64, error)

	// Close is used to close database
	Close()
	//不在事务中，直接查询状态数据库，返回一行结果
	QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error)
	//不在事务中，直接查询状态数据库，返回多行结果
	QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error)
	//执行DDL语句
	ExecDdlSql(contractName, sql string) error
	//启用一个事务
	BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error)
	//根据事务名，获得一个已经启用的事务
	GetDbTransaction(txName string) (protocol.SqlDBTransaction, error)
	//提交一个事务
	CommitDbTransaction(txName string) error
	//回滚一个事务
	RollbackDbTransaction(txName string) error
}
