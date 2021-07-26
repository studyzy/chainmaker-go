/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historysqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker/protocol"
)

// HistorySqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type HistorySqlDB struct {
	db     protocol.SqlDBHandle
	logger protocol.Logger
	dbName string
}

//NewHistorySqlDB construct a new `HistoryDB` for given chainId
func NewHistorySqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*HistorySqlDB, error) {
	dbName := getDbName(dbConfig, chainId)
	db := rawsqlprovider.NewSqlDBHandle(dbName, dbConfig, logger)
	return newHistorySqlDB(dbName, db, logger)
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *HistorySqlDB) initDb(dbName string) {
	db.logger.Debugf("create history database:%s", dbName)
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		db.logger.Panicf("init state sql db fail,error:%s", err)
	}
	db.logger.Debug("create table[state_history_infos] to save history")
	err = db.db.CreateTableIfNotExist(&StateHistoryInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `state_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&AccountTxHistoryInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `account_tx_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&ContractTxHistoryInfo{})
	if err != nil {
		db.logger.Panicf("init state sql db table `contract_tx_history_infos` fail, error:%s", err)
	}
	err = db.db.CreateTableIfNotExist(&types.SavePoint{})
	if err != nil {
		db.logger.Panicf("init state sql db table `save_points` fail, error:%s", err)
	}
	_, err = db.db.Save(&types.SavePoint{})
	if err != nil {
		db.logger.Panicf("insert new SavePoint get an error:%s", err)
	}

}
func getDbName(dbConfig *localconf.SqlDbConfig, chainId string) string {
	return dbConfig.DbPrefix + "historydb_" + chainId
}
func newHistorySqlDB(dbName string, db protocol.SqlDBHandle, logger protocol.Logger) (*HistorySqlDB, error) {

	historyDB := &HistorySqlDB{
		db:     db,
		logger: logger,
		dbName: dbName,
	}
	return historyDB, nil
}
func (h *HistorySqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	h.initDb(h.dbName)
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
			if len(w.Key) == 0 {
				continue
			}
			historyInfo := NewStateHistoryInfo(w.ContractName, txRWSet.TxId, w.Key, uint64(block.Header.BlockHeight))
			_, err = dbtx.Save(historyInfo)
			if err != nil {
				h.logger.Errorf("save tx[%s] state key[%s] history info fail,rollback history save transaction,%s",
					txRWSet.TxId, w.Key, err.Error())
				err2 := h.db.RollbackDbTransaction(blockHashStr)
				if err2 != nil {
					return err2
				}
				return err
			}

		}
	}
	for _, tx := range blockInfo.Block.Txs {
		txSender := tx.GetSenderAccountId()
		if len(txSender) == 0 {
			continue //genesis block tx don't have sender
		}
		accountTxInfo := &AccountTxHistoryInfo{
			AccountId:   txSender,
			BlockHeight: uint64(block.Header.BlockHeight),
			TxId:        tx.Payload.TxId,
		}
		_, err = dbtx.Save(accountTxInfo)
		if err != nil {
			h.logger.Errorf("save account[%s] and tx[%s] info fail,rollback history save transaction,%s",
				txSender, tx.Payload.TxId, err.Error())
			err2 := h.db.RollbackDbTransaction(blockHashStr)
			if err2 != nil {
				return err2
			}
			return err
		}

		contractName := tx.Payload.ContractName

		contractTxInfo := &ContractTxHistoryInfo{
			ContractName: contractName,
			BlockHeight:  uint64(block.Header.BlockHeight),
			TxId:         tx.Payload.TxId,
			AccountId:    txSender,
		}
		_, err = dbtx.Save(contractTxInfo)
		if err != nil {
			h.logger.Errorf("save contract[%s] and tx[%s] history info fail,rollback history save transaction,%s",
				contractName, tx.Payload.TxId, err.Error())
			err2 := h.db.RollbackDbTransaction(blockHashStr)
			if err2 != nil {
				return err2
			}
			return err
		}
	}
	//save last point
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

	h.logger.Debugf("chain[%s]: commit history db, block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil

}

func (s *HistorySqlDB) GetLastSavepoint() (uint64, error) {
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

func (h *HistorySqlDB) Close() {
	h.logger.Info("close history sql db")
	h.db.Close()
}

type hisIter struct {
	rows protocol.SqlRows
}

func (hi *hisIter) Next() bool {
	return hi.rows.Next()
}
func (hi *hisIter) Value() (*historydb.BlockHeightTxId, error) {
	var txId string
	var blockHeight uint64
	err := hi.rows.ScanColumns(&txId, &blockHeight)
	if err != nil {
		return nil, err
	}
	return &historydb.BlockHeightTxId{TxId: txId, BlockHeight: blockHeight}, nil
}
func (hi *hisIter) Release() {
	hi.rows.Close()
}
func newHisIter(rows protocol.SqlRows) *hisIter {
	return &hisIter{rows: rows}
}
func (h *HistorySqlDB) GetHistoryForKey(contractName string, key []byte) (historydb.HistoryIterator, error) {
	sql := `select tx_id,block_height 
from state_history_infos 
where contract_name=? and state_key=? 
order by block_height desc`
	rows, err := h.db.QueryMulti(sql, contractName, key)
	if err != nil {
		return nil, err
	}
	return newHisIter(rows), nil
}
func (h *HistorySqlDB) GetAccountTxHistory(account []byte) (historydb.HistoryIterator, error) {
	sql := `select tx_id,block_height 
from account_tx_history_infos 
where account_id=? 
order by block_height desc`
	rows, err := h.db.QueryMulti(sql, account)
	if err != nil {
		return nil, err
	}
	return newHisIter(rows), nil
}
func (h *HistorySqlDB) GetContractTxHistory(contractName string) (historydb.HistoryIterator, error) {
	sql := `select tx_id,block_height 
from contract_tx_history_infos 
where contract_name=? 
order by block_height desc`
	rows, err := h.db.QueryMulti(sql, contractName)
	if err != nil {
		return nil, err
	}
	return newHisIter(rows), nil
}
