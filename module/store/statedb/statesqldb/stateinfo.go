/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

import (
	"time"

	"chainmaker.org/chainmaker-go/localconf"

	"chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
)

const (
	contractStoreSeparator = '#'
)

// StateInfo defines mysql orm model, used to create mysql table 'state_infos'
type StateInfo struct {
	//ID           uint   `gorm:"primarykey"`
	ContractName string `gorm:"size:128;primaryKey"`
	ObjectKey    []byte `gorm:"size:128;primaryKey;default:''"`
	ObjectValue  []byte `gorm:"type:longblob"`
	BlockHeight  uint64 `gorm:"index:idx_height"`
	//CreatedAt    time.Time `gorm:"default:null"`
	UpdatedAt time.Time `gorm:"default:null"`
}

func (b *StateInfo) ScanObject(scan func(dest ...interface{}) error) error {
	return scan(&b.ContractName, &b.ObjectKey, &b.ObjectValue, &b.BlockHeight, &b.UpdatedAt)
}
func (b *StateInfo) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqlDbConfig_SqlDbType_MySQL {
		return `CREATE TABLE state_infos (
    contract_name varchar(128),object_key varbinary(128) DEFAULT '',
    object_value longblob,block_height bigint unsigned,
    updated_at datetime(3) NULL DEFAULT null,
    PRIMARY KEY (contract_name,object_key),
    INDEX idx_height (block_height)
    ) default character set utf8`
	} else if dbType == localconf.SqlDbConfig_SqlDbType_Sqlite {
		return `CREATE TABLE state_infos (
    contract_name text,object_key blob DEFAULT '',
    object_value longblob,block_height integer,updated_at datetime DEFAULT null,
    PRIMARY KEY (contract_name,object_key)
    )`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *StateInfo) GetTableName() string {
	return "state_infos"
}
func (b *StateInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO state_infos values(?,?,?,?,?)",
		[]interface{}{b.ContractName, b.ObjectKey, b.ObjectValue, b.BlockHeight, b.UpdatedAt}
}
func (b *StateInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE state_infos set object_value=?,block_height=?,updated_at=?" +
			" WHERE contract_name=? and object_key=?",
		[]interface{}{b.ObjectValue, b.BlockHeight, b.UpdatedAt, b.ContractName, b.ObjectKey}
}
func (b *StateInfo) GetCountSql() (string, []interface{}) {
	return "select count(*) FROM state_infos WHERE contract_name=? and object_key=?",
		[]interface{}{b.ContractName, b.ObjectKey}
}

// NewStateInfo construct a new StateInfo
func NewStateInfo(contractName string, objectKey []byte, objectValue []byte, blockHeight uint64,
	t time.Time) *StateInfo {
	return &StateInfo{
		ContractName: contractName,
		ObjectKey:    objectKey,
		ObjectValue:  objectValue,
		BlockHeight:  blockHeight,
		UpdatedAt:    t,
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
	err := kv.ScanObject(kvi.rows.ScanColumns)
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
