/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resultsqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"errors"
	"github.com/gogo/protobuf/proto"
)

// ResultSqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type ResultSqlDB struct {
	db     protocol.SqlDBHandle
	logger protocol.Logger
	dbName string
}

// NewResultSqlDB construct a new `HistoryDB` for given chainId
func NewResultSqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*ResultSqlDB, error) {
	dbName := getDbName(dbConfig, chainId)
	db := rawsqlprovider.NewSqlDBHandle(dbName, dbConfig, logger)
	return newResultSqlDB(dbName, db, logger)
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *ResultSqlDB) initDb(dbName string) {
	db.logger.Debugf("create result database %s to save transaction receipt", dbName)
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		db.logger.Panicf("init state sql db fail,error:%s", err)
	}
	db.logger.Debug("create table result_infos")
	err = db.db.CreateTableIfNotExist(&ResultInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `state_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&types.SavePoint{})
	if err != nil {
		db.logger.Panicf("init state sql db table `save_points` fail,error:%s", err)
	}
	_, err = db.db.Save(&types.SavePoint{})
	if err != nil {
		db.logger.Panicf("insert new SavePoint to table get an error:%s", err)
	}
}
func getDbName(dbConfig *localconf.SqlDbConfig, chainId string) string {
	return dbConfig.DbPrefix + "resultdb_" + chainId
}
func newResultSqlDB(dbName string, db protocol.SqlDBHandle, logger protocol.Logger) (*ResultSqlDB, error) {
	rdb := &ResultSqlDB{
		db:     db,
		logger: logger,
		dbName: dbName,
	}
	return rdb, nil
}
func (h *ResultSqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	h.initDb(h.dbName)
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
		_, err = dbtx.Save(resultInfo)
		if err != nil {
			err2 := h.db.RollbackDbTransaction(blockHashStr)
			if err2 != nil {
				return err2
			}
			return err
		}

	}
	_, err = dbtx.ExecSql("update save_points set block_height=?", block.Header.BlockHeight)
	if err != nil {
		h.logger.Errorf("update save point error:%s", err)
		err2 := h.db.RollbackDbTransaction(blockHashStr)
		if err2 != nil {
			return err2
		}
		return err
	}
	err = h.db.CommitDbTransaction(blockHashStr)
	if err != nil {
		return err
	}

	h.logger.Debugf("chain[%s]: commit result db, block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil

}

// ShrinkBlocks archive old blocks rwsets in an atomic operation
func (h *ResultSqlDB) ShrinkBlocks(txIdsMap map[uint64][]string) error {
	return errors.New("implement me")
}

// RestoreBlocks restore blocks from outside serialized block data
func (h *ResultSqlDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	return errors.New("implement me")
}

func (h *ResultSqlDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	sql := "select rwset from result_infos where tx_id=?"
	result, err := h.db.QuerySingle(sql, txId)
	if err != nil {
		return nil, err
	}
	if result.IsEmpty() {
		h.logger.Infof("cannot query rwset by txid=%s", txId)
		return nil, nil
	}
	var b []byte
	err = result.ScanColumns(&b)
	if err != nil {
		return nil, err
	}
	var rwSet commonPb.TxRWSet
	err = proto.Unmarshal(b, &rwSet)
	if err != nil {
		return nil, err
	}
	return &rwSet, nil
}

func (s *ResultSqlDB) GetLastSavepoint() (uint64, error) {
	sql := "select block_height from save_points"
	row, err := s.db.QuerySingle(sql)
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

func (h *ResultSqlDB) Close() {
	h.logger.Info("close result sql db")
	h.db.Close()
}
