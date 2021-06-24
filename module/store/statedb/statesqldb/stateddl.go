/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

import (
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/hash"
	"encoding/hex"
	"time"

	"chainmaker.org/chainmaker-go/localconf"

	"chainmaker.org/chainmaker-go/protocol"
)

const (
	expireBlockHeight      = 5
	contractStoreSeparator = "#"
)

// StateRecordSql defines mysql orm model, used to create mysql table 'state_record_sql'
type StateRecordSql struct {
	// id is sql hash
	Id                string    `gorm:"size:64;primaryKey"`
	ContractName      string    `gorm:"size:100"`
	Sql               string    `gorm:"size:4000"`
	SqlType           int       `gorm:"size:1;default:1"`
	BlockHeight       uint64    `gorm:"default:0"`
	ExpireBlockHeight uint64    `gorm:"default:0"`
	UpdatedAt         time.Time `gorm:"default:null"`
}

func (b *StateRecordSql) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqlDbConfig_SqlDbType_MySQL {
		return `CREATE TABLE state_record_sql (
					id varchar(64),
					contract_name varchar(100),
					sql varchar(4000),
					block_height bigint unsigned,
					expire_block_height bigint unsigned,
					updated_at datetime(3) NULL DEFAULT null,
					PRIMARY KEY (id)
				) default character set utf8`
	} else if dbType == localconf.SqlDbConfig_SqlDbType_Sqlite {
		return `CREATE TABLE state_record_sql (
					id text,
					contract_name text,
					sql text,
					block_height integer,
					expire_block_height integer,
					updated_at datetime DEFAULT null,
					PRIMARY KEY (id)
				)`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *StateRecordSql) GetTableName() string {
	return "state_record_sql"
}

func (b *StateRecordSql) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO state_record_sql values(?,?,?,?,?,?,?)",
		[]interface{}{b.Id, b.ContractName, b.Sql, b.SqlType, b.BlockHeight, b.ExpireBlockHeight, b.UpdatedAt}
}

func (b *StateRecordSql) GetUpdateSql() (string, []interface{}) {
	return "UPDATE state_record_sql set block_height=?,expire_block_height=?,updated_at=?" +
			" WHERE id=? ",
		[]interface{}{b.BlockHeight, b.ExpireBlockHeight, b.UpdatedAt, b.Id}
}

func (b *StateRecordSql) GetQuerySql() (string, []interface{}) {
	return "select * FROM state_record_sql WHERE id=? and expire_block_height>=?",
		[]interface{}{b.Id, b.BlockHeight}
}

func (b *StateRecordSql) GetCountSql() (string, []interface{}) {
	return "select count(*) FROM state_record_sql WHERE id=? and expire_block_height>=?",
		[]interface{}{b.Id, b.BlockHeight}
}

// NewStateRecordSql construct a new StateRecordSql
func NewStateRecordSql(contractName string, sql string, sqlType protocol.SqlType, blockHeight uint64) *StateRecordSql {
	bytes, _ := hash.Get(crypto.HASH_TYPE_SHA256, []byte(contractName+contractStoreSeparator+sql))
	id := hex.EncodeToString(bytes)
	return &StateRecordSql{
		Id:                id,
		ContractName:      contractName,
		Sql:               sql,
		SqlType:           int(sqlType),
		BlockHeight:       blockHeight,
		ExpireBlockHeight: blockHeight + expireBlockHeight,
		UpdatedAt:         time.Now(),
	}
}
