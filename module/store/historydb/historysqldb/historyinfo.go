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
	BlockHeight  uint64 `gorm:"primaryKey"`
}

// NewHistoryInfo construct a new HistoryInfo
func NewStateHistoryInfo(contractName, txid string, StateKey []byte, blockHeight uint64) *StateHistoryInfo {
	return &StateHistoryInfo{
		TxId:         txid,
		ContractName: contractName,
		StateKey:     StateKey,
		BlockHeight:  blockHeight,
	}
}

type AccountTxHistoryInfo struct {
	AccountId   []byte `gorm:"size:128;primaryKey"`
	BlockHeight uint64 `gorm:"primaryKey"`
	TxId        string `gorm:"size:128;primaryKey"`
}
type ContractTxHistoryInfo struct {
	ContractName string `gorm:"size:128;primaryKey"`
	BlockHeight  uint64 `gorm:"primaryKey"`
	TxId         string `gorm:"size:128;primaryKey"`
	AccountId    []byte `gorm:"size:128"`
}
