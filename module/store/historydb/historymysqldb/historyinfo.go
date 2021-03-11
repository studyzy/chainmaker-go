/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historymysqldb

// HistoryInfo defines mysql orm model, used to create mysql table 'hisgory_infos'
type HistoryInfo struct {
	TxId        string `gorm:"size:128;primaryKey"`
	RwSets      []byte `gorm:"type:longblob"`
	BlockHeight int64  `gorm:"index:idx_height"`
}

// NewHistoryInfo construct a new HistoryInfo
func NewHistoryInfo(txid string, rwSets []byte, blockHeight int64) *HistoryInfo {
	return &HistoryInfo{
		TxId:        txid,
		RwSets:      rwSets,
		BlockHeight: blockHeight,
	}
}
