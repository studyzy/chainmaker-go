/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/sqldbprovider"
)

// StateSqlDB provider a implementation of `statedb.StateDB`
// This implementation provides a mysql based data model
type StateSqlDB struct {
	db     protocol.SqlDBHandle
	Logger protocol.Logger
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *StateSqlDB) initDb(dbName string) {
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		panic("init state sql db fail")
	}
	err = db.db.CreateTableIfNotExist(&StateInfo{})
	if err != nil {
		panic("init state sql db table fail")
	}
}

// NewStateMysqlDB construct a new `StateDB` for given chainId
func NewStateSqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*StateSqlDB, error) {
	db := sqldbprovider.NewSqlDBHandle(chainId, dbConfig, logger)
	return newStateSqlDB(chainId, db, logger)
}
func getDbName(chainId string) string {
	return chainId + "_statedb"
}
func newStateSqlDB(chainId string, db protocol.SqlDBHandle, logger protocol.Logger) (*StateSqlDB, error) {
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	stateDB := &StateSqlDB{
		db:     db,
		Logger: logger,
	}

	return stateDB, nil
}
func (s *StateSqlDB) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	s.initDb(getDbName(genesisBlock.Block.Header.ChainId))
	return s.CommitBlock(genesisBlock)
}
func getContractDbName(chainId, contractName string) string {
	return chainId + "_" + contractName
}

// CommitBlock commits the state in an atomic operation
func (s *StateSqlDB) CommitBlock(blockWithRWSet *storePb.BlockWithRWSet) error {
	block := blockWithRWSet.Block
	txRWSets := blockWithRWSet.TxRWSets
	blockHash := block.GetBlockHashStr()
	if len(txRWSets) == 0 {
		s.Logger.Warnf("block[%d] don't have any read write set data", block.Header.BlockHeight)
		return nil
	}
	if block.IsContractMgmtBlock() {
		//创建对应合约的数据库
		payload := &commonPb.ContractMgmtPayload{}
		payload.Unmarshal(block.Txs[0].RequestPayload)
		dbName := getContractDbName(block.Header.ChainId, payload.ContractId.ContractName)
		s.initDb(dbName) //创建KV表
		writes := txRWSets[0].TxWrites
		for _, write := range writes {
			if len(write.Key) == 0 { //这是SQL语句
				_, err := s.db.ExecSql(string(write.Value)) //运行用户自定义的建表语句
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	tx, err := s.db.BeginDbTransaction(blockHash)
	if err != nil {
		return err
	}
	currentDb := ""
	for _, txRWSet := range txRWSets {
		for _, txWrite := range txRWSet.TxWrites {
			if txWrite.ContractName != "" && (txWrite.ContractName != currentDb || currentDb == "") { //切换DB
				tx.ChangeContextDb(txWrite.ContractName)
				currentDb = txWrite.ContractName
			}
			if txWrite.Key == nil { //sql
				sql := string(txWrite.Value)
				if _, err := tx.ExecSql(sql); err != nil {
					s.Logger.Errorf("execute sql[%s] get error:%s", txWrite.Value, err.Error())
					s.db.RollbackDbTransaction(blockHash)
					return err
				}
			} else {
				stateInfo := NewStateInfo(txWrite.ContractName, txWrite.Key, txWrite.Value, block.Header.BlockHeight)
				if _, err := tx.Save(stateInfo); err != nil {
					s.Logger.Errorf("save state key[%s] get error:%s", txWrite.Key, err.Error())
					s.db.RollbackDbTransaction(blockHash)
					return err
				}
			}
		}
	}
	err = s.db.CommitDbTransaction(blockHash)
	if err != nil {
		s.Logger.Error(err.Error())
		return err
	}
	s.Logger.Debugf("chain[%s]: commit state block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil
}

// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
func (s *StateSqlDB) ReadObject(contractName string, key []byte) ([]byte, error) {
	if contractName != "" {
		if err := s.db.ChangeContextDb(contractName); err != nil {
			return nil, err
		}
	}
	var stateInfo StateInfo
	sql := "select * from state_infos where object_key=?"

	res, err := s.db.QuerySql(sql, key)
	if err != nil {
		s.Logger.Errorf("failed to read state, contract:%s, key:%s", contractName, key)
		return nil, err
	}
	err = res.ScanObject(&stateInfo)
	if err != nil {
		s.Logger.Errorf("failed to read state, contract:%s, key:%s", contractName, key)
		return nil, err
	}
	return stateInfo.ObjectValue, nil
}

// SelectObject returns an iterator that contains all the key-values between given key ranges.
// startKey is included in the results and limit is excluded.
func (s *StateSqlDB) SelectObject(contractName string, startKey []byte, limit []byte) protocol.Iterator {
	sql := "select * from state_infos where object_key between ? and ?"
	rows, err := s.db.QueryTableSql(sql, startKey, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := &kvIterator{}
	for rows.Next() {
		var kv StateInfo
		rows.ScanObject(&kv)
		result.append(&kv)
	}
	return result
}

// GetLastSavepoint returns the last block height
func (s *StateSqlDB) GetLastSavepoint() (uint64, error) {
	sql := "select max(block_height) from state_infos"
	row, err := s.db.QuerySql(sql)
	if err != nil {
		return 0, err
	}
	var height *uint64
	err = row.ScanColumns(&height)
	if err != nil {
		return 0, err
	}
	if height == nil {
		return 0, nil
	}
	return *height, nil
}

// Close is used to close database, there is no need for gorm to close db
func (s *StateSqlDB) Close() {
	s.db.Close()
}
func (s *StateSqlDB) ExecSql(sql string, values ...interface{}) (int64, error) {
	return s.db.ExecSql(sql, values...)
}
func (s *StateSqlDB) QuerySql(sql string, values ...interface{}) (protocol.SqlRow, error) {
	return s.db.QuerySql(sql, values...)
}
func (s *StateSqlDB) QueryTableSql(sql string, values ...interface{}) (protocol.SqlRows, error) {
	return s.db.QueryTableSql(sql, values...)

}
func (s *StateSqlDB) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return s.db.BeginDbTransaction(txName)

}
func (s *StateSqlDB) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return s.db.GetDbTransaction(txName)

}
func (s *StateSqlDB) CommitDbTransaction(txName string) error {
	return s.db.CommitDbTransaction(txName)

}
func (s *StateSqlDB) RollbackDbTransaction(txName string) error {
	return s.db.RollbackDbTransaction(txName)

}
