/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historymysqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/store/dbprovider/mysqldbprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"gorm.io/gorm"
)

// HistoryMysqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type HistoryMysqlDB struct {
	db     *gorm.DB
	Logger *logImpl.CMLogger
}

// NewHistoryMysqlDB construct a new `HistoryDB` for given chainId
func NewHistoryMysqlDB(chainId string) (historydb.HistoryDB, error) {
	db := mysqldbprovider.NewProvider(chainId).GetDB(chainId, localconf.ChainMakerConfig)
	if err := db.AutoMigrate(&HistoryInfo{}); err != nil {
		panic(fmt.Sprintf("failed to migrate blockinfo:%s", err))
	}
	historyDB := &HistoryMysqlDB{
		db:     db,
		Logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
	}
	return historyDB, nil
}

func (h *HistoryMysqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	block := blockInfo.Block
	txRWSets := blockInfo.TxRWSets

	historyInfos := make([]*HistoryInfo, 0, len(txRWSets))
	for _, txRWSet := range txRWSets {
		// rwset: txID -> txRWSet
		txRWSetBytes, err := proto.Marshal(txRWSet)
		if err != nil {
			return err
		}
		historyInfo := NewHistoryInfo(txRWSet.TxId, txRWSetBytes, block.Header.BlockHeight)
		historyInfos = append(historyInfos, historyInfo)
	}

	return h.db.Transaction(func(tx *gorm.DB) error {
		for _, historyInfo := range historyInfos {
			//res := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(historyInfo)
			res := tx.Save(historyInfo)
			if res.Error != nil {
				h.Logger.Errorf("failed to set history, txid:%s, err:%s",
					historyInfo.TxId, res.Error)
				return res.Error
			}
		}
		h.Logger.Debugf("chain[%s]: commit history db, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return nil
	})

}

func (h *HistoryMysqlDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	var historyInfo HistoryInfo
	historyInfo.TxId = txId
	res := h.db.Find(&historyInfo)
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		h.Logger.Errorf("failed to read state, txid:%s, err:%s", txId, res.Error)
		return nil, res.Error
	}
	var txRWSet commonPb.TxRWSet
	err := proto.Unmarshal(historyInfo.RwSets, &txRWSet)
	if err != nil {
		return nil, err
	}
	return &txRWSet, nil
}

func (h *HistoryMysqlDB) GetLastSavepoint() (uint64, error) {
	var historyInfo HistoryInfo
	res := h.db.Order("block_height desc").Limit(1).Find(&historyInfo)
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		h.Logger.Errorf("failed to get last savepoint")
		return 0, res.Error
	}
	return uint64(historyInfo.BlockHeight), nil
}

func (h *HistoryMysqlDB) Close() {
	sqlDB, err := h.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}
