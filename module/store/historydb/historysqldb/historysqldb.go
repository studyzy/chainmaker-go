/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historysqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/sqldbprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
)

// HistorySqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type HistorySqlDB struct {
	db     protocol.SqlDBHandle
	Logger protocol.Logger
}

// NewHistoryMysqlDB construct a new `HistoryDB` for given chainId
func NewHistorySqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*HistorySqlDB, error) {
	db := sqldbprovider.NewSqlDBHandle(chainId, dbConfig, logger)
	return newHistorySqlDB(chainId, db, logger)
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *HistorySqlDB) initDb(dbName string) {
	db.Logger.Debugf("create history database:%s", dbName)
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		panic("init state sql db fail")
	}
	db.Logger.Debug("create table[state_history_infos] to save history")
	err = db.db.CreateTableIfNotExist(&StateHistoryInfo{})
	if err != nil {
		panic("init state sql db table `state_history_infos` fail")
	}

}
func getDbName(chainId string) string {
	return "historydb_" + chainId
}
func newHistorySqlDB(chainId string, db protocol.SqlDBHandle, logger protocol.Logger) (*HistorySqlDB, error) {

	historyDB := &HistorySqlDB{
		db:     db,
		Logger: logger,
	}
	return historyDB, nil
}
func (h *HistorySqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	h.initDb(getDbName(genesisBlock.Block.Header.ChainId))
	return h.CommitBlock(genesisBlock)
}
func (h *HistorySqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	block := blockInfo.Block
	txRWSets := blockInfo.TxRWSets
	blockHashStr := block.GetBlockHashStr()
	dbtx, err := h.db.BeginDbTransaction(blockHashStr)
	if err != nil {
		return err
	}
	for _, txRWSet := range txRWSets {
		for _, w := range txRWSet.TxWrites {
			historyInfo := NewStateHistoryInfo(w.ContractName, txRWSet.TxId, w.Key, block.Header.BlockHeight)
			_, err := dbtx.Save(historyInfo)
			if err != nil {
				h.db.RollbackDbTransaction(blockHashStr)
				return err
			}
		}

	}
	h.db.CommitDbTransaction(blockHashStr)

	h.Logger.Debugf("chain[%s]: commit history db, block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil

}

func (h *HistorySqlDB) GetLastSavepoint() (uint64, error) {
	row, err := h.db.QuerySingle("select max(block_height) from state_history_infos")
	if err != nil {
		return 0, err
	}
	var height *uint64
	err = row.ScanColumns(&height)
	if err != nil {
		h.Logger.Error(err.Error())
		return 0, err
	}
	if height == nil {
		return 0, nil
	}
	return *height, nil
}

func (h *HistorySqlDB) Close() {
	h.Logger.Info("close history sql db")
	h.db.Close()
}
