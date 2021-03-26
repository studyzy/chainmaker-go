/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

import "time"

// StateInfo defines mysql orm model, used to create mysql table 'state_infos'
type StateInfo struct {
	//ID           uint   `gorm:"primarykey"`
	ContractName string `gorm:"size:128;primaryKey"`
	ObjectKey    []byte `gorm:"size:128;primaryKey;default:''"`
	ObjectValue  []byte `gorm:"type:longblob"`
	BlockHeight  int64  `gorm:"index:idx_height"`
	//CreatedAt    time.Time `gorm:"default:null"`
	UpdatedAt time.Time `gorm:"default:null"`
}

// NewStateInfo construct a new StateInfo
func NewStateInfo(contractName string, objectKey []byte, objectValue []byte, blockHeight int64) *StateInfo {
	return &StateInfo{
		ContractName: contractName,
		ObjectKey:    objectKey,
		ObjectValue:  objectValue,
		BlockHeight:  blockHeight,
	}
}
