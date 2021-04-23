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
type HistoryIterator interface {
	Next() bool
	Value() (*store.KeyModification, error)
	Release()
}

// BlockchainStore provides handle to store instances
type BlockchainStore interface {
	StateSqlOperation
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
type StateSqlOperation interface {
	//不在事务中，直接查询状态数据库，返回一行结果
	QuerySingle(contractName, sql string, values ...interface{}) (SqlRow, error)
	//不在事务中，直接查询状态数据库，返回多行结果
	QueryMulti(contractName, sql string, values ...interface{}) (SqlRows, error)
	//执行建表、修改表等DDL语句，不得在事务中运行
	ExecDdlSql(contractName, sql string) error
	//启用一个事务
	BeginDbTransaction(txName string) (SqlDBTransaction, error)
	//根据事务名，获得一个已经启用的事务
	GetDbTransaction(txName string) (SqlDBTransaction, error)
	//提交一个事务
	CommitDbTransaction(txName string) error
	//回滚一个事务
	RollbackDbTransaction(txName string) error
}

//SqlDBHandle 对SQL数据库的操作方法
type SqlDBHandle interface {
	DBHandle
	//CreateDatabaseIfNotExist 如果数据库不存在则创建对应的数据库，创建后将当前数据库设置为新数据库
	CreateDatabaseIfNotExist(dbName string) error
	//ChangeContextDb 改变当前上下文所使用的数据库
	ChangeContextDb(dbName string) error
	//CreateTableIfNotExist 根据一个对象struct，自动构建对应的sql数据库表
	CreateTableIfNotExist(obj interface{}) error
	//Save 直接保存一个对象到SQL数据库中
	Save(value interface{}) (int64, error)
	//ExecSql 执行指定的SQL语句，返回受影响的行数
	ExecSql(sql string, values ...interface{}) (int64, error)
	//QuerySingle 执行指定的SQL语句，查询单条数据记录，如果查询到0条，则返回nil,nil，如果查询到多条，则返回第一条
	QuerySingle(sql string, values ...interface{}) (SqlRow, error)
	//QueryMulti 执行指定的SQL语句，查询多条数据记录，如果查询到0条，则SqlRows.Next()直接返回false
	QueryMulti(sql string, values ...interface{}) (SqlRows, error)
	//BeginDbTransaction 开启一个数据库事务，并指定该事务的名字，并缓存其句柄，如果之前已经开启了同名的事务，则返回错误
	BeginDbTransaction(txName string) (SqlDBTransaction, error)
	//GetDbTransaction 根据事务的名字，获得事务的句柄,如果事务不存在，则返回错误
	GetDbTransaction(txName string) (SqlDBTransaction, error)
	//CommitDbTransaction 提交一个事务，并从缓存中清除该事务，如果找不到对应的事务，则返回错误
	CommitDbTransaction(txName string) error
	//RollbackDbTransaction 回滚一个事务，并从缓存中清除该事务，如果找不到对应的事务，则返回错误
	RollbackDbTransaction(txName string) error
}

//SqlDBTransaction开启一个事务后，能在这个事务中进行的操作
type SqlDBTransaction interface {
	//ChangeContextDb 改变当前上下文所使用的数据库
	ChangeContextDb(dbName string) error
	//Save 直接保存一个对象到SQL数据库中
	Save(value interface{}) (int64, error)
	//ExecSql 执行指定的SQL语句，返回受影响的行数
	ExecSql(sql string, values ...interface{}) (int64, error)
	//QuerySingle 执行指定的SQL语句，查询单条数据记录，如果查询到0条，则返回nil,nil，如果查询到多条，则返回第一条
	QuerySingle(sql string, values ...interface{}) (SqlRow, error)
	//QueryMulti 执行指定的SQL语句，查询多条数据记录，如果查询到0条，则SqlRows.Next()直接返回false
	QueryMulti(sql string, values ...interface{}) (SqlRows, error)
	//BeginDbSavePoint 创建一个新的保存点
	BeginDbSavePoint(savePointName string) error
	//回滚事务到指定的保存点
	RollbackDbSavePoint(savePointName string) error
}

//运行SQL查询后返回的一行数据，在获取这行数据时提供了ScanColumns，ScanObject和Data三种方法，但是三选一，调用其中一个就别再调另外一个。
type SqlRow interface {
	//将这个数据的每个列赋值到dest指针对应的对象中
	ScanColumns(dest ...interface{}) error
	//将这个数据赋值到dest对象的属性中
	ScanObject(dest interface{}) error
	//将这个数据转换为ColumnName为Key，Data为Value的Map中
	Data() (map[string]string, error)
	//判断返回的SqlRow是否为空
	IsEmpty() bool
}

//运行SQL查询后返回的多行数据
type SqlRows interface {
	//还有下一行
	Next() bool
	//将当前行这个数据的每个列赋值到dest指针对应的对象中
	ScanColumns(dest ...interface{}) error
	//将当前行这个数据赋值到dest对象的属性中
	ScanObject(dest interface{}) error
	//将当前行这个数据转换为ColumnName为Key，Data为Value的Map中
	Data() (map[string]string, error)
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

//SqlVerifier 在支持SQL语句操作状态数据库模式下，对合约中输入的SQL语句进行规则校验
type SqlVerifier interface {
	//VerifyDDLSql 验证输入语句是不是DDL语句，是DDL则返回nil，不是则返回error
	VerifyDDLSql(sql string) error
	//VerifyDMLSql 验证输入的SQL语句是不是更新语句（insert、update、delete），是则返回nil，不是则返回error
	VerifyDMLSql(sql string) error
	//VerifyDQLSql 验证输入的语句是不是查询语句，是则返回nil，不是则返回error
	VerifyDQLSql(sql string) error
}
