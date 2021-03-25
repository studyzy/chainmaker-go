/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockmysqldb

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"fmt"
	"runtime"

	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/dbprovider/mysqldbprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/utils"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

// BlockMysqlDB provider a implementation of `blockdb.BlockDB`
// This implementation provides a mysql based data model
type BlockMysqlDB struct {
	db               *gorm.DB
	workersSemaphore *semaphore.Weighted
	Logger           protocol.Logger
}

// NewBlockMysqlDB constructs a new `BlockMysqlDB` given an chainId and engine type
func NewBlockMysqlDB(chainId string, logger protocol.Logger) (blockdb.BlockDB, error) {
	nWorkers := runtime.NumCPU()
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	db := mysqldbprovider.NewProvider().GetDB(chainId, localconf.ChainMakerConfig)

	if err := db.AutoMigrate(&BlockInfo{}); err != nil {
		panic(fmt.Sprintf("failed to migrate blockinfo:%s", err))
	}
	if err := db.AutoMigrate(&TxInfo{}); err != nil {
		panic(fmt.Sprintf("failed to migrate txinfo:%s", err))
	}
	blockDB := &BlockMysqlDB{
		db:               db,
		workersSemaphore: semaphore.NewWeighted(int64(nWorkers)),
		Logger:           logger,
	}
	return blockDB, nil
}

// CommitBlock commits the block and the corresponding rwsets in an atomic operation
func (b *BlockMysqlDB) CommitBlock(blockWithSerializedInfo *serialization.BlockWithSerializedInfo) error {
	block := blockWithSerializedInfo.Block
	startCommitTxs := utils.CurrentTimeMillisSeconds()
	//save txs
	txInfos := make([]*TxInfo, 0, len(block.Txs))
	for index, tx := range block.Txs {
		txinfo, err := NewTxInfo(tx, block.Header.BlockHeight, int32(index))
		if err != nil {
			b.Logger.Errorf("failed to init txinfo, err:%s", err)
			return err
		}
		txInfos = append(txInfos, txinfo)
	}

	err := b.db.Transaction(func(tx *gorm.DB) error {
		for _, txInfo := range txInfos {
			//res := b.db.Clauses(clause.OnConflict{DoNothing: true}).Create(txInfo)
			res := tx.Save(txInfo)
			if res.Error != nil {
				b.Logger.Errorf("faield to commit txinfo info, height:%d, tx:%s,err:%s",
					block.Header.BlockHeight, txInfo.TxId, res.Error)
				return res.Error
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	elapsedCommitTxs := utils.CurrentTimeMillisSeconds() - startCommitTxs

	//save block info
	startCommitBlockInfo := utils.CurrentTimeMillisSeconds()
	blockInfo, err := NewBlockInfo(block)
	if err != nil {
		b.Logger.Errorf("failed to init blockinfo, err:%s", err)
		return err
	}
	response := b.db.Save(blockInfo)
	if response.Error != nil {
		b.Logger.Errorf("faield to commit block info, height:%d, err:%s",
			block.Header.BlockHeight, response.Error)
		return response.Error
	}
	elapsedCommitBlockInfos := utils.CurrentTimeMillisSeconds() - startCommitBlockInfo
	b.Logger.Infof("chain[%s]: commit block[%d] time used (commit_txs:%d commit_block:%d, total:%d)",
		block.Header.ChainId, block.Header.BlockHeight,
		elapsedCommitTxs, elapsedCommitBlockInfos,
		utils.CurrentTimeMillisSeconds()-startCommitTxs)
	return nil
}

// HasBlock returns true if the block hash exist, or returns false if none exists.
func (b *BlockMysqlDB) BlockExists(blockHash []byte) (bool, error) {
	var count int64
	res := b.db.Model(&BlockInfo{}).Where("block_hash = ?", blockHash).Count(&count)
	if res.Error == gorm.ErrRecordNotFound {
		return false, nil
	} else if res.Error != nil {
		return false, res.Error
	}
	if count > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

// GetBlock returns a block given it's hash, or returns nil if none exists.
func (b *BlockMysqlDB) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	//get block info from mysql
	var blockInfo BlockInfo
	res := b.db.Where("block_hash = ?", blockHash).First(&blockInfo)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		return nil, res.Error
	}
	return b.getBlockByInfo(&blockInfo)
}

// GetBlockAt returns a block given it's block height, or returns nil if none exists.
func (b *BlockMysqlDB) GetBlock(height int64) (*commonPb.Block, error) {
	//get block info from mysql
	var blockInfo BlockInfo
	res := b.db.Find(&blockInfo, height)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		return nil, res.Error
	}
	return b.getBlockByInfo(&blockInfo)
}

// GetLastBlock returns the last block.
func (b *BlockMysqlDB) GetLastBlock() (*commonPb.Block, error) {
	var blockInfo BlockInfo
	res := b.db.Last(&blockInfo)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		return nil, res.Error
	}
	return b.getBlockByInfo(&blockInfo)
}

// GetLastConfigBlock returns the last config block.
func (b *BlockMysqlDB) GetLastConfigBlock() (*commonPb.Block, error) {
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
func (b *BlockMysqlDB) GetFilteredBlock(height int64) (*storePb.SerializedBlock, error) {
	var blockInfo BlockInfo
	res := b.db.Find(&blockInfo, height)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		return nil, res.Error
	}
	return blockInfo.GetFilterdBlock()
}

// GetLastSavepoint reurns the last block height
func (b *BlockMysqlDB) GetLastSavepoint() (uint64, error) {
	var block_height uint64
	res := b.db.Model(&BlockInfo{}).Select("block_height").Last(&block_height)
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		return 0, res.Error
	}
	return block_height, nil
}

// GetBlockByTx returns a block which contains a tx.
func (b *BlockMysqlDB) GetBlockByTx(txId string) (*commonPb.Block, error) {
	//var txInfo TxInfo
	var blockHeight int64
	res := b.db.Model(&TxInfo{}).Select("block_height").Where("tx_id = ?", txId).Find(&blockHeight)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		return nil, res.Error
	}
	return b.GetBlock(blockHeight)
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
func (b *BlockMysqlDB) GetTx(txId string) (*commonPb.Transaction, error) {
	var txInfo TxInfo
	res := b.db.Where("tx_id = ?", txId).First(&txInfo)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		return nil, res.Error
	}
	return txInfo.GetTx()
}

// HasTx returns true if the tx exist, or returns false if none exists.
func (b *BlockMysqlDB) TxExists(txId string) (bool, error) {
	var count int64
	res := b.db.Model(&TxInfo{}).Where("tx_id = ?", txId).Count(&count)
	if res.Error == gorm.ErrRecordNotFound {
		return false, nil
	} else if res.Error != nil {
		return false, res.Error
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (b *BlockMysqlDB) GetTxConfirmedTime(txId string) (int64, error) {
	panic("implement me")
}

// Close is used to close database
func (b *BlockMysqlDB) Close() {
	sqlDB, err := b.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}

// GetDB returns the db handler of `gorm.DB`
func (b *BlockMysqlDB) GetDB() *gorm.DB {
	return b.db
}

func (b *BlockMysqlDB) getBlockByInfo(blockInfo *BlockInfo) (*commonPb.Block, error) {
	//get txinfos form mysql
	var txInfos []TxInfo
	//res = b.db.Debug().Find(&txInfos, txList)
	res := b.db.Where("block_height = ?",
		blockInfo.BlockHeight).Order("offset asc").Find(&txInfos)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		b.Logger.Errorf("failed to get tx from tx_info, height:%s, err:%s", blockInfo.BlockHeight, res.Error)
		return nil, res.Error
	}

	block, err := blockInfo.GetBlock()
	if err != nil {
		b.Logger.Errorf("failed to transform blockinfo to block, chain:%s, block:%d, err:%s",
			blockInfo.ChainId, blockInfo.BlockHeight, err)
		return nil, err
	}
	for _, txInfo := range txInfos {
		tx, err := txInfo.GetTx()
		if err != nil {
			b.Logger.Errorf("failed to transform txinfo to tx, chain:%s, txid:%s, err:%s",
				block.Header.ChainId, txInfo.TxId, err)
		}
		block.Txs = append(block.Txs, tx)
	}
	return block, nil
}
