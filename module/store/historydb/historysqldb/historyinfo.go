/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historysqldb

// StateHistoryInfo defines mysql orm model, used to create mysql table 'state_history_infos'
type StateHistoryInfo struct {
	ContractName string `gorm:"size:128;primaryKey"`
	StateKey     []byte `gorm:"size:128;primaryKey"`
	TxId         string `gorm:"size:128;primaryKey"`
	BlockHeight  int64  `gorm:"primaryKey"`
}

// NewHistoryInfo construct a new HistoryInfo
func NewStateHistoryInfo(contractName, txid string, StateKey []byte, blockHeight int64) *StateHistoryInfo {
	return &StateHistoryInfo{
		TxId:         txid,
		ContractName: contractName,
		StateKey:     StateKey,
		BlockHeight:  blockHeight,
	}
}
