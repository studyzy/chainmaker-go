/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/store"
)

var (
	// ConsensusDBName is used to store consensus data
	ConsensusDBName = "consensus"
)

// Iterator allows a chaincode to iteratoe over a set of
// kev/value pairs returned by range query.
type Iterator interface {
	Next() bool
	First() bool
	Error() error
	Key() []byte
	Value() []byte
	Release()
}

// BlockchainStore provides handle to store instances
type BlockchainStore interface {
	//InitGenesis 初始化创世单元到数据库
	InitGenesis(genesisBlock *store.BlockWithRWSet) error
	// PutBlock commits the block and the corresponding rwsets in an atomic operation
	PutBlock(block *common.Block, txRWSets []*common.TxRWSet) error

	// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
	GetBlockByHash(blockHash []byte) (*common.Block, error)

	// BlockExists returns true if the black hash exist, or returns false if none exists.
	BlockExists(blockHash []byte) (bool, error)

	// GetBlock returns a block given it's block height, or returns nil if none exists.
	GetBlock(height int64) (*common.Block, error)

	// GetLastConfigBlock returns the last config block.
	GetLastConfigBlock() (*common.Block, error)

	// GetBlockByTx returns a block which contains a tx.
	GetBlockByTx(txId string) (*common.Block, error)

	// GetBlockWithRWSets returns a block and the corresponding rwsets given
	// it's block height, or returns nil if none exists.
	GetBlockWithRWSets(height int64) (*store.BlockWithRWSet, error)

	// GetTx retrieves a transaction by txid, or returns nil if none exists.
	GetTx(txId string) (*common.Transaction, error)

	// TxExists returns true if the tx exist, or returns false if none exists.
	TxExists(txId string) (bool, error)

	// GetTxConfirmedTime returns the confirmed time for given tx
	GetTxConfirmedTime(txId string) (int64, error)

	// GetLastBlock returns the last block.
	GetLastBlock() (*common.Block, error)

	// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
	ReadObject(contractName string, key []byte) ([]byte, error)

	// SelectObject returns an iterator that contains all the key-values between given key ranges.
	// startKey is included in the results and limit is excluded.
	SelectObject(contractName string, startKey []byte, limit []byte) Iterator

	// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
	GetTxRWSet(txId string) (*common.TxRWSet, error)

	// GetTxRWSetsByHeight returns all the rwsets corresponding to the block,
	// or returns nil if zhe block does not exist
	GetTxRWSetsByHeight(height int64) ([]*common.TxRWSet, error)

	// GetDBHandle returns the database handle for given dbName
	GetDBHandle(dbName string) DBHandle

	// Close closes all the store db instances and releases any resources held by BlockchainStore
	Close() error
}
type SqlDBHandle interface {
	DBHandle
	CreateDatabaseIfNotExist(dbName string) error
	ChangeContextDb(dbName string) error
	CreateTableIfNotExist(obj interface{}) error
	Save(value interface{}) (int64,error)
	ExecSql(sql string, values ...interface{}) (int64, error)
	QuerySql(sql string, values ...interface{}) (SqlRow, error)
	QueryTableSql(sql string, values ...interface{}) (SqlRows, error)
	BeginDbTransaction(txName string) SqlDBTransaction
	GetDbTransaction(txName string) (SqlDBTransaction,error)
	CommitDbTransaction(txName string) error
	RollbackDbTransaction(txName string) error
}
type SqlDBTransaction interface {
	ChangeContextDb(dbName string) error
	Save(value interface{}) (int64,error)
	ExecSql(sql string, values ...interface{}) (int64, error)
	QuerySql(sql string, values ...interface{}) (SqlRow, error)
	QueryTableSql(sql string, values ...interface{}) (SqlRows, error)
	//Commit() error
	//Rollback() error
	BeginDbSavePoint(savePointName string) error
	RollbackDbSavePoint(savePointName string) error
}

type SqlRow interface {
	ScanColumns(dest ...interface{}) error
	ScanObject(dest interface{}) error
}
type SqlRows interface {
	Next() bool
	ScanColumns(dest ...interface{}) error
	ScanObject(dest interface{}) error
	Close() error
}
// DBHandle is an handle to a db
type DBHandle interface {
	// Get returns the value for the given key, or returns nil if none exists
	Get(key []byte) ([]byte, error)

	// Put saves the key-values
	Put(key []byte, value []byte) error

	// Has return true if the given key exist, or return false if none exists
	Has(key []byte) (bool, error)

	// Delete deletes the given key
	Delete(key []byte) error

	// WriteBatch writes a batch in an atomic operation
	WriteBatch(batch StoreBatcher, sync bool) error

	// NewIteratorWithRange returns an iterator that contains all the key-values between given key ranges
	// start is included in the results and limit is excluded.
	NewIteratorWithRange(start []byte, limit []byte) Iterator

	// NewIteratorWithPrefix returns an iterator that contains all the key-values with given prefix
	NewIteratorWithPrefix(prefix []byte) Iterator
	Close() error

}

// StoreBatcher used to cache key-values that commit in a atomic operation
type StoreBatcher interface {
	// Put adds a key-value
	Put(key []byte, value []byte)

	// Delete deletes a key and associated value
	Delete(key []byte)

	// Len retrun the number of key-values
	Len() int

	// Merge used to merge two StoreBatcher
	Merge(batcher StoreBatcher)

	// KVs return the map of key-values
	KVs() map[string][]byte
}
