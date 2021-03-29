/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historysqldb

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/serialization"
)

// HistorySqlDB provider a implementation of `history.HistoryDB`
// This implementation provides a mysql based data model
type HistorySqlDB struct {
	db     protocol.SqlDBHandle
	Logger protocol.Logger
}

// NewHistoryMysqlDB construct a new `HistoryDB` for given chainId
func NewHistorySqlDB(chainId string, db protocol.SqlDBHandle, logger protocol.Logger) (*HistorySqlDB, error) {
	//db := sqldbprovider.NewProvider().GetDB(chainId, localconf.ChainMakerConfig)
	//if logger == nil {
	//	logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	//}
	//if err := db.AutoMigrate(&HistoryInfo{}); err != nil {
	//	panic(fmt.Sprintf("failed to migrate blockinfo:%s", err))
	//}
	historyDB := &HistorySqlDB{
		db:     db,
		Logger: logger,
	}
	return historyDB, nil
}

func (h *HistorySqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	block := blockInfo.Block
	txRWSets := blockInfo.TxRWSets
	blockHashStr := block.GetBlockHashStr()
	dbtx := h.db.BeginDbTransaction(blockHashStr)
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

func (h *HistorySqlDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	return nil, nil
}

func (h *HistorySqlDB) GetLastSavepoint() (uint64, error) {
	row, err := h.db.QuerySql("select max(block_height) from state_history_infos")
	if err != nil {
		return 0, err
	}
	var height uint64
	err = row.ScanColumns(&height)
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (h *HistorySqlDB) Close() {
	h.db.Close()
}
