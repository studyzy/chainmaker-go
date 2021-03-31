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

type kvIterator struct {
	keyValues []*StateInfo
	idx       int
	count     int
}

func newKVIterator() *kvIterator {
	return &kvIterator{
		keyValues: make([]*StateInfo, 0),
		idx:       0,
		count:     0,
	}
}
func (kvi *kvIterator) append(kv *StateInfo) {
	kvi.keyValues = append(kvi.keyValues, kv)
	kvi.count++
}
func (kvi *kvIterator) Next() bool {
	kvi.idx++
	return kvi.idx < kvi.count
}
func (kvi *kvIterator) First() bool {
	return kvi.idx == 0
}
func (kvi *kvIterator) Error() error {
	return nil
}
func (kvi *kvIterator) Key() []byte {
	return kvi.keyValues[kvi.idx].ObjectKey
}
func (kvi *kvIterator) Value() []byte {
	return kvi.keyValues[kvi.idx].ObjectValue
}
func (kvi *kvIterator) Release() {
	kvi.idx = 0
	kvi.count = 0
	kvi.keyValues = make([]*StateInfo, 0)
}
