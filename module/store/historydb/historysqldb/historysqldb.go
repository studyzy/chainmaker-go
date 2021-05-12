/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historysqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
)

// HistorySqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type HistorySqlDB struct {
	db     protocol.SqlDBHandle
	logger protocol.Logger
	dbName string
}

// NewHistoryMysqlDB construct a new `HistoryDB` for given chainId
func NewHistorySqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*HistorySqlDB, error) {
	db := rawsqlprovider.NewSqlDBHandle(getDbName(chainId), dbConfig, logger)
	return newHistorySqlDB(chainId, db, logger)
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *HistorySqlDB) initDb(dbName string) {
	db.logger.Debugf("create history database:%s", dbName)
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		panic("init state sql db fail")
	}
	db.logger.Debug("create table[state_history_infos] to save history")
	err = db.db.CreateTableIfNotExist(&StateHistoryInfo{})
	if err != nil {
		panic("init state sql db table `state_history_infos` fail")
	}
	err = db.db.CreateTableIfNotExist(&AccountTxHistoryInfo{})
	if err != nil {
		panic("init state sql db table `account_tx_history_infos` fail")
	}
	err = db.db.CreateTableIfNotExist(&ContractTxHistoryInfo{})
	if err != nil {
		panic("init state sql db table `contract_tx_history_infos` fail")
	}
	err = db.db.CreateTableIfNotExist(&types.SavePoint{})
	if err != nil {
		panic("init state sql db table `save_points` fail")
	}
	db.db.Save(&types.SavePoint{0})

}
func getDbName(chainId string) string {
	return "historydb_" + chainId
}
func newHistorySqlDB(chainId string, db protocol.SqlDBHandle, logger protocol.Logger) (*HistorySqlDB, error) {

	historyDB := &HistorySqlDB{
		db:     db,
		logger: logger,
		dbName: getDbName(chainId),
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
			historyInfo := NewStateHistoryInfo(w.ContractName, txRWSet.TxId, w.Key, uint64(block.Header.BlockHeight))
			_, err := dbtx.Save(historyInfo)
			if err != nil {
				h.logger.Errorf("save tx[%s] state key[%s] history info fail,rollback history save transaction,%s", txRWSet.TxId, w.Key, err.Error())
				h.db.RollbackDbTransaction(blockHashStr)
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
			TxId:        tx.Header.TxId,
		}
		_, err := dbtx.Save(accountTxInfo)
		if err != nil {
			h.logger.Errorf("save account[%s] and tx[%s] info fail,rollback history save transaction,%s", txSender, tx.Header.TxId, err.Error())
			h.db.RollbackDbTransaction(blockHashStr)
			return err
		}
		contractName, err := tx.GetContractName()
		if err != nil {
			h.logger.Warnf("Tx[%s] don't have contract name since:%s", tx.Header.TxId, err.Error())
			continue
		}
		contractTxInfo := &ContractTxHistoryInfo{
			ContractName: contractName,
			BlockHeight:  uint64(block.Header.BlockHeight),
			TxId:         tx.Header.TxId,
			AccountId:    txSender,
		}
		_, err = dbtx.Save(contractTxInfo)
		if err != nil {
			h.logger.Errorf("save contract[%s] and tx[%s] history info fail,rollback history save transaction,%s", contractName, tx.Header.TxId, err.Error())
			h.db.RollbackDbTransaction(blockHashStr)
			return err
		}
	}
	//save last point
	_, err = dbtx.ExecSql("update save_points set block_height=?", block.Header.BlockHeight)
	if err != nil {
		h.logger.Errorf("update save point error:%s", err)
		h.db.RollbackDbTransaction(blockHashStr)
		return err
	}
	h.db.CommitDbTransaction(blockHashStr)

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
func NewHisIter(rows protocol.SqlRows) *hisIter {
	return &hisIter{rows: rows}
}
func (h *HistorySqlDB) GetHistoryForKey(contractName string, key []byte) (historydb.HistoryIterator, error) {
	sql := "select tx_id,block_height from state_history_infos where contract_name=? and state_key=? order by block_height desc"
	rows, err := h.db.QueryMulti(sql, contractName, key)
	if err != nil {
		return nil, err
	}
	return NewHisIter(rows), nil
}
func (h *HistorySqlDB) GetAccountTxHistory(account []byte) (historydb.HistoryIterator, error) {
	sql := "select tx_id,block_height from account_tx_history_infos where account_id=? order by block_height desc"
	rows, err := h.db.QueryMulti(sql, account)
	if err != nil {
		return nil, err
	}
	return NewHisIter(rows), nil
}
func (h *HistorySqlDB) GetContractTxHistory(contractName string) (historydb.HistoryIterator, error) {
	sql := "select tx_id,block_height from contract_tx_history_infos where contract_name=? order by block_height desc"
	rows, err := h.db.QueryMulti(sql, contractName)
	if err != nil {
		return nil, err
	}
	return NewHisIter(rows), nil
}
