/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resultsqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/sqldbprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"github.com/gogo/protobuf/proto"
)

// ResultSqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type ResultSqlDB struct {
	db     protocol.SqlDBHandle
	Logger protocol.Logger
}

// NewHistoryMysqlDB construct a new `HistoryDB` for given chainId
func NewResultSqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*ResultSqlDB, error) {
	db := sqldbprovider.NewSqlDBHandle(chainId, dbConfig, logger)
	return newResultSqlDB(chainId, db, logger)
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *ResultSqlDB) initDb(dbName string) {
	db.Logger.Debugf("create result database %s to save transaction reciept", dbName)
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		panic("init state sql db fail")
	}
	db.Logger.Debug("create table result_infos")
	err = db.db.CreateTableIfNotExist(&ResultInfo{})
	if err != nil {
		panic("init state sql db table `state_history_infos` fail")
	}

}
func getDbName(chainId string) string {
	return chainId + "_result"
}
func newResultSqlDB(chainId string, db protocol.SqlDBHandle, logger protocol.Logger) (*ResultSqlDB, error) {
	rdb := &ResultSqlDB{
		db:     db,
		Logger: logger,
	}
	return rdb, nil
}
func (h *ResultSqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	h.initDb(getDbName(genesisBlock.Block.Header.ChainId))
	return h.CommitBlock(genesisBlock)
}
func (h *ResultSqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	block := blockInfo.Block
	txRWSets := blockInfo.TxRWSets
	blockHashStr := block.GetBlockHashStr()
	dbtx, err := h.db.BeginDbTransaction(blockHashStr)
	if err != nil {
		return err
	}
	for i, txRWSet := range txRWSets {
		tx := block.Txs[i]

		resultInfo := NewResultInfo(tx.Header.TxId, block.Header.BlockHeight, i, tx.Result.ContractResult, txRWSet)
		_, err := dbtx.Save(resultInfo)
		if err != nil {
			h.db.RollbackDbTransaction(blockHashStr)
			return err
		}

	}
	h.db.CommitDbTransaction(blockHashStr)

	h.Logger.Debugf("chain[%s]: commit result db, block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil

}

func (h *ResultSqlDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	sql := "select rwset from result_infos where tx_id=?"
	result, err := h.db.QuerySql(sql, txId)
	if err != nil {
		return nil, err
	}
	var b []byte
	err = result.ScanColumns(&b)
	if err != nil {
		return nil, err
	}
	var rwSet commonPb.TxRWSet
	proto.Unmarshal(b, &rwSet)
	return &rwSet, nil
}

func (h *ResultSqlDB) GetLastSavepoint() (uint64, error) {
	row, err := h.db.QuerySql("select max(block_height) from result_infos")
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

func (h *ResultSqlDB) Close() {
	h.db.Close()
}
