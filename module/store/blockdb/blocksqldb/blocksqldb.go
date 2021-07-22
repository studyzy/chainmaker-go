/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	"errors"
	"runtime"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	storePb "chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/protocol"
	"golang.org/x/sync/semaphore"
)

var NotImplementError = errors.New("implement me")

// BlockSqlDB provider a implementation of `blockdb.BlockDB`
// This implementation provides a mysql based data model
type BlockSqlDB struct {
	db               protocol.SqlDBHandle
	workersSemaphore *semaphore.Weighted
	logger           protocol.Logger
	dbName           string
}

func (db *BlockSqlDB) GetHeightByHash(blockHash []byte) (uint64, error) {
	sql := "SELECT block_height FROM block_infos WHERE block_hash=?"
	var height uint64
	res, err := db.db.QuerySingle(sql, blockHash)
	if err != nil {
		return 0, err
	}
	err = res.ScanColumns(&height)
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (db *BlockSqlDB) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	sql := "SELECT * from block_infos WHERE block_height=?"
	blockInfo, err := db.getBlockInfoBySql(sql, height)
	if err != nil {
		return nil, err
	}
	if blockInfo == nil && err == nil {
		return nil, nil
	}
	return blockInfo.GetBlockHeader(), nil
}

func (db *BlockSqlDB) GetTxHeight(txId string) (uint64, error) {
	sql := "SELECT block_height FROM tx_infos WHERE tx_id=?"
	var height uint64
	res, err := db.db.QuerySingle(sql, txId)
	if err != nil {
		return 0, err
	}
	err = res.ScanColumns(&height)
	if err != nil {
		return 0, err
	}
	return height, nil

}

func (db *BlockSqlDB) TxArchived(txId string) (bool, error) {
	return false, nil
}

func (db *BlockSqlDB) GetArchivedPivot() (uint64, error) {
	return 0, nil
}

func (db *BlockSqlDB) ShrinkBlocks(startHeight uint64, endHeight uint64) (map[uint64][]string, error) {
	return nil, NotImplementError
}

func (db *BlockSqlDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	return NotImplementError
}

// NewBlockSqlDB constructs a new `BlockSqlDB` given an chainId and engine type
func NewBlockSqlDB(chainId string, dbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*BlockSqlDB, error) {
	dbName := getDbName(dbConfig, chainId)
	db := rawsqlprovider.NewSqlDBHandle(dbName, dbConfig, logger)
	return newBlockSqlDB(dbName, db, logger)
}

//如果数据库不存在，则创建数据库，然后切换到这个数据库，创建表
//如果数据库存在，则切换数据库，检查表是否存在，不存在则创建表。
func (db *BlockSqlDB) initDb(dbName string) {
	err := db.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		panic("init state sql db fail")
	}

	err = db.db.CreateTableIfNotExist(&BlockInfo{})
	if err != nil {
		panic("init state sql db table `block_infos` fail" + err.Error())
	}
	err = db.db.CreateTableIfNotExist(&TxInfo{})
	if err != nil {
		panic("init state sql db table `tx_infos` fail")
	}
}
func getDbName(dbConfig *localconf.SqlDbConfig, chainId string) string {
	return dbConfig.DbPrefix + "blockdb_" + chainId
}
func newBlockSqlDB(dbName string, db protocol.SqlDBHandle, logger protocol.Logger) (*BlockSqlDB, error) {
	nWorkers := runtime.NumCPU()

	blockDB := &BlockSqlDB{
		db:               db,
		workersSemaphore: semaphore.NewWeighted(int64(nWorkers)),
		logger:           logger,
		dbName:           dbName,
	}
	return blockDB, nil
}

func (b *BlockSqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	b.initDb(b.dbName)
	return b.CommitBlock(genesisBlock)
}

// CommitBlock commits the block and the corresponding rwsets in an atomic operation
func (b *BlockSqlDB) CommitBlock(blocksInfo *serialization.BlockWithSerializedInfo) error {
	block := blocksInfo.Block
	dbTxKey := block.GetTxKey()
	startCommitTxs := utils.CurrentTimeMillisSeconds()
	dbtx, err := b.db.BeginDbTransaction(dbTxKey)
	if err != nil {
		return err
	}
	//save txs
	for index, tx := range block.Txs {
		var txInfo *TxInfo
		txInfo, err = NewTxInfo(tx, uint64(block.Header.BlockHeight), block.Header.BlockHash, uint32(index))
		if err != nil {
			b.logger.Errorf("failed to init txinfo, err:%s", err)
			if err2 := b.db.RollbackDbTransaction(dbTxKey); err2 != nil {
				b.logger.Errorf("failed to rollback db transaction[%s],error:%s", dbTxKey, err2.Error())
				return err2
			}
			return err
		}
		_, err = dbtx.Save(txInfo)
		if err != nil {
			b.logger.Errorf("failed to commit txinfo info, height:%d, tx:%s,err:%s",
				block.Header.BlockHeight, txInfo.TxId, err)
			if err2 := b.db.RollbackDbTransaction(dbTxKey); err2 != nil {
				b.logger.Errorf("failed to rollback db transaction[%s],error:%s", dbTxKey, err2.Error())
				return err2
			}
			return err
		}
	}

	elapsedCommitTxs := utils.CurrentTimeMillisSeconds() - startCommitTxs
	//save block info
	startCommitBlockInfo := utils.CurrentTimeMillisSeconds()
	blockInfo, err := NewBlockInfo(block)
	if err != nil {
		b.logger.Errorf("failed to init blockinfo, err:%s", err)
		if err2 := b.db.RollbackDbTransaction(dbTxKey); err2 != nil {
			b.logger.Errorf("failed to rollback db transaction[%s],error:%s", dbTxKey, err2.Error())
			return err2
		}
		return err
	}
	_, err = dbtx.Save(blockInfo)
	if err != nil {
		b.logger.Errorf("failed to commit block info, height:%d, err:%s",
			block.Header.BlockHeight, err)
		_ = b.db.RollbackDbTransaction(dbTxKey) //rollback tx
		return err
	}
	err = b.db.CommitDbTransaction(dbTxKey)
	if err != nil {
		b.logger.Errorf("failed to commit tx, err:%s", err)
		return err
	}
	elapsedCommitBlockInfos := utils.CurrentTimeMillisSeconds() - startCommitBlockInfo
	b.logger.Infof("chain[%s]: commit block[%d] time used (commit_txs:%d commit_block:%d, total:%d)",
		block.Header.ChainId, block.Header.BlockHeight,
		elapsedCommitTxs, elapsedCommitBlockInfos,
		utils.CurrentTimeMillisSeconds()-startCommitTxs)
	return nil
}

// BlockExists returns true if the block hash exist, or returns false if none exists.
func (b *BlockSqlDB) BlockExists(blockHash []byte) (bool, error) {
	var count int64
	sql := "select count(*) from block_infos where block_hash = ?"
	res, err := b.db.QuerySingle(sql, blockHash)
	if err != nil {
		return false, err
	}
	err = res.ScanColumns(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
func (b *BlockSqlDB) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {

	return b.getFullBlockBySql("select * from block_infos where block_hash = ?", blockHash)
}
func (b *BlockSqlDB) getBlockInfoBySql(sql string, values ...interface{}) (*BlockInfo, error) {
	//get block info from mysql
	var blockInfo BlockInfo
	res, err := b.db.QuerySingle(sql, values...)
	if err != nil {
		return nil, err
	}
	if res.IsEmpty() {
		b.logger.Infof("sql[%s] %v return empty result", sql, values)
		return nil, nil
	}
	err = blockInfo.ScanObject(res.ScanColumns)
	if err != nil {
		return nil, err
	}
	return &blockInfo, nil
}
func (b *BlockSqlDB) getFullBlockBySql(sql string, values ...interface{}) (*commonPb.Block, error) {
	blockInfo, err := b.getBlockInfoBySql(sql, values...)
	if err != nil {
		return nil, err
	}
	if blockInfo == nil && err == nil {
		return nil, nil
	}
	block, err := blockInfo.GetBlock()
	if err != nil {
		return nil, err
	}
	txs, err := b.getTxsByBlockHeight(blockInfo.BlockHeight)
	if err != nil {
		return nil, err
	}
	block.Txs = txs
	return block, nil
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
func (b *BlockSqlDB) GetBlock(height uint64) (*commonPb.Block, error) {
	return b.getFullBlockBySql("select * from block_infos where block_height =?", height)
}

// GetLastBlock returns the last block.
func (b *BlockSqlDB) GetLastBlock() (*commonPb.Block, error) {
	return b.getFullBlockBySql(`select * 
from block_infos 
where block_height = (select max(block_height) from block_infos)`)
}

// GetLastConfigBlock returns the last config block.
func (b *BlockSqlDB) GetLastConfigBlock() (*commonPb.Block, error) {
	lastBlock, err := b.GetLastBlock()
	if err != nil {
		return nil, err
	}
	if utils.IsConfBlock(lastBlock) {
		return lastBlock, nil
	}
	return b.GetBlock(lastBlock.Header.PreConfHeight)
}

// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
func (b *BlockSqlDB) GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error) {
	blockInfo, err := b.getBlockInfoBySql("select * from block_infos where block_height = ?", height)
	if err != nil {
		return nil, err
	}
	if blockInfo == nil && err == nil {
		return nil, nil
	}
	return blockInfo.GetFilteredBlock()
}

// GetLastSavepoint reurns the last block height
func (b *BlockSqlDB) GetLastSavepoint() (uint64, error) {
	sql := "select max(block_height) from block_infos"
	row, err := b.db.QuerySingle(sql)
	if err != nil {
		b.logger.Errorf("get block sqldb save point error:%s", err.Error())
		return 0, err
	}
	if row.IsEmpty() {
		return 0, nil
	}
	var height uint64
	err = row.ScanColumns(&height)
	if err != nil {
		return 0, err
	}

	return height, nil
}

// GetBlockByTx returns a block which contains a tx.
func (b *BlockSqlDB) GetBlockByTx(txId string) (*commonPb.Block, error) {
	sql := "select * from block_infos where block_height=(select block_height from tx_infos where tx_id=?)"
	return b.getFullBlockBySql(sql, txId)
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
func (b *BlockSqlDB) GetTx(txId string) (*commonPb.Transaction, error) {
	if len(txId) == 0 {
		return nil, errors.New("parameter is null")
	}
	var txInfo TxInfo
	res, err := b.db.QuerySingle("select * from tx_infos where tx_id = ?", txId)
	if err != nil {
		return nil, err
	}
	if res.IsEmpty() {
		b.logger.Infof("tx[%s] not found in db", txId)
		return nil, nil
	}

	err = txInfo.ScanObject(res.ScanColumns)
	if err != nil {
		return nil, err
	}
	if len(txInfo.TxId) > 0 {
		return txInfo.GetTx()
	}
	b.logger.Errorf("tx data not found by txid:%s", txId)
	return nil, errors.New("data not found")
}
func (b *BlockSqlDB) GetTxWithBlockInfo(txId string) (*commonPb.TransactionInfo, error) {
	var txInfo TxInfo
	res, err := b.db.QuerySingle("select * from tx_infos where tx_id = ?", txId)
	if err != nil {
		return nil, err
	}
	if res.IsEmpty() {
		b.logger.Infof("tx[%s] not found in db", txId)
		return nil, nil
	}
	err = txInfo.ScanObject(res.ScanColumns)
	if err != nil {
		return nil, err
	}
	if len(txInfo.TxId) > 0 {
		return txInfo.GetTxInfo()
	}
	b.logger.Errorf("tx data not found by txid:%s", txId)
	return nil, errors.New("data not found")
}

// TxExists returns true if the tx exist, or returns false if none exists.
func (b *BlockSqlDB) TxExists(txId string) (bool, error) {
	var count int64
	sql := "select count(*) from tx_infos where tx_id = ?"
	res, err := b.db.QuerySingle(sql, txId)
	if err != nil {
		return false, err
	}
	err = res.ScanColumns(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

//获得某个区块高度下的所有交易
func (b *BlockSqlDB) getTxsByBlockHeight(blockHeight uint64) ([]*commonPb.Transaction, error) {
	res, err := b.db.QueryMulti("select * from tx_infos where block_height = ? order by offset", blockHeight)
	if err != nil {
		return nil, err
	}
	result := []*commonPb.Transaction{}
	for res.Next() {
		var txInfo TxInfo
		err = txInfo.ScanObject(res.ScanColumns)
		if err != nil {
			return nil, err
		}
		tx, err := txInfo.GetTx()
		if err != nil {
			return nil, err
		}
		result = append(result, tx)
	}
	return result, nil
}
func (b *BlockSqlDB) GetTxConfirmedTime(txId string) (int64, error) {
	return 0, NotImplementError
}

// Close is used to close database
func (b *BlockSqlDB) Close() {
	b.logger.Info("close block sql db")
	b.db.Close()
}
