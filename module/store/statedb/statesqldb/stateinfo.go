/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

import (
	"chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"time"
)

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

type kvIterator struct {
	rows protocol.SqlRows
}

func newKVIterator(rows protocol.SqlRows) *kvIterator {
	return &kvIterator{
		rows: rows,
	}
}
func (kvi *kvIterator) Next() bool {
	return kvi.rows.Next()
}

func (kvi *kvIterator) Value() (*store.KV, error) {
	var kv StateInfo
	err := kvi.rows.ScanObject(&kv)
	if err != nil {
		return nil, err
	}
	return &store.KV{
		ContractName: kv.ContractName,
		Key:          kv.ObjectKey,
		Value:        kv.ObjectValue,
	}, nil
}
func (kvi *kvIterator) Release() {
	kvi.rows.Close()
}
