/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

import (
	"encoding/hex"
	"time"

	"chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/hash"

	"chainmaker.org/chainmaker-go/localconf"

	"chainmaker.org/chainmaker/protocol"
)

const (
	contractStoreSeparator = "#"
)

// StateRecordSql defines mysql orm model, used to create mysql table 'state_record_sql'
type StateRecordSql struct {
	// id is sql hash
	Id           string    `gorm:"size:64;primaryKey"`
	ContractName string    `gorm:"size:100"`
	SqlString    string    `gorm:"size:4000"`
	SqlType      int       `gorm:"size:1;default:1"`
	Version      string    `gorm:"size:20"`
	Status       int       `gorm:"default:0"` //0: start process, 1:success 2:fail
	UpdatedAt    time.Time `gorm:"default:null"`
}

func (b *StateRecordSql) GetCreateTableSql(dbType string) string {
	if dbType == localconf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE state_record_sql (
					id varchar(64),
					contract_name varchar(100),
					sql_string varchar(4000),
					version varchar(20),
					status int,
					updated_at datetime(3) NULL DEFAULT null,
					PRIMARY KEY (id)
				) default character set utf8`
	} else if dbType == localconf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE state_record_sql (
					id text,
					contract_name text,
					sql_string text,
					version text,
					status integer,
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
		[]interface{}{b.Id, b.ContractName, b.SqlString, b.SqlType, b.Version, b.Status, b.UpdatedAt}
}

func (b *StateRecordSql) GetUpdateSql() (string, []interface{}) {
	return "UPDATE state_record_sql set updated_at=?,status=?" +
			" WHERE id=?",
		[]interface{}{b.UpdatedAt, b.Status, b.Id}
}

func (b *StateRecordSql) GetQueryStatusSql() (string, interface{}) {
	return "select status FROM state_record_sql WHERE id=?",
		b.Id
}

func (b *StateRecordSql) GetCountSql() (string, interface{}) {
	return "select count(*) FROM state_record_sql WHERE id=?",
		b.Id
}

// NewStateRecordSql construct a new StateRecordSql
func NewStateRecordSql(contractName string, sql string, sqlType protocol.SqlType,
	version string, status int) *StateRecordSql {
	rawId := []byte(contractName + contractStoreSeparator + version + contractStoreSeparator + sql)
	bytes, _ := hash.Get(crypto.HASH_TYPE_SHA256, rawId)
	id := hex.EncodeToString(bytes)
	return &StateRecordSql{
		Id:           id,
		ContractName: contractName,
		SqlString:    sql,
		SqlType:      int(sqlType),
		Version:      version,
		Status:       status,
		UpdatedAt:    time.Now(),
	}
}
