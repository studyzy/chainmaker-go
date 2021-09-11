/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historysqldb

import "chainmaker.org/chainmaker-go/store/conf"

// StateHistoryInfo defines mysql orm model, used to create mysql table 'state_history_infos'
type StateHistoryInfo struct {
	ContractName string `gorm:"size:128;primaryKey"`
	StateKey     []byte `gorm:"size:128;primaryKey"`
	TxId         string `gorm:"size:128;primaryKey"`
	BlockHeight  uint64 `gorm:"primaryKey"`
}

func (b *StateHistoryInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE state_history_infos (
    contract_name varchar(128),state_key varbinary(128),tx_id varchar(128),block_height bigint unsigned,
    PRIMARY KEY (contract_name,state_key,tx_id,block_height)
    ) default character set utf8`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE state_history_infos (
    contract_name text,state_key blob,tx_id text,block_height integer,
    PRIMARY KEY (contract_name,state_key,tx_id,block_height))`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *StateHistoryInfo) GetTableName() string {
	return "state_history_infos"
}
func (b *StateHistoryInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO state_history_infos values(?,?,?,?)",
		[]interface{}{b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}
func (b *StateHistoryInfo) GetUpdateSql() (string, []interface{}) {
	return `UPDATE state_history_infos set contract_name=?
			 WHERE contract_name=? and state_key=? and tx_id=? and block_height=?`,
		[]interface{}{b.ContractName, b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}
func (b *StateHistoryInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM state_history_infos" +
			" WHERE contract_name=? and state_key=? and tx_id=? and block_height=?",
		[]interface{}{b.ContractName, b.StateKey, b.TxId, b.BlockHeight}
}

// NewStateHistoryInfo construct a new HistoryInfo
func NewStateHistoryInfo(contractName, txid string, stateKey []byte, blockHeight uint64) *StateHistoryInfo {
	return &StateHistoryInfo{
		TxId:         txid,
		ContractName: contractName,
		StateKey:     stateKey,
		BlockHeight:  blockHeight,
	}
}

type AccountTxHistoryInfo struct {
	AccountId   []byte `gorm:"size:2048;primaryKey"` //primary key size max=3072
	BlockHeight uint64 `gorm:"primaryKey"`
	TxId        string `gorm:"size:128;primaryKey"`
}

func (b *AccountTxHistoryInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE account_tx_history_infos (
    account_id varbinary(2048),block_height bigint unsigned,tx_id varchar(128),
    PRIMARY KEY (account_id,block_height,tx_id)
    ) default character set utf8`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE account_tx_history_infos (
account_id blob,block_height integer,tx_id text,
PRIMARY KEY (account_id,block_height,tx_id))`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *AccountTxHistoryInfo) GetTableName() string {
	return "account_tx_history_infos"
}
func (b *AccountTxHistoryInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO account_tx_history_infos values(?,?,?)", []interface{}{b.AccountId, b.BlockHeight, b.TxId}
}
func (b *AccountTxHistoryInfo) GetUpdateSql() (string, []interface{}) {
	return "UPDATE account_tx_history_infos set account_id=?" +
		" WHERE account_id=? and block_height=? and tx_id=?", []interface{}{b.AccountId, b.AccountId, b.BlockHeight, b.TxId}
}
func (b *AccountTxHistoryInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM account_tx_history_infos" +
			" WHERE account_id=? and block_height=? and tx_id=?",
		[]interface{}{b.AccountId, b.BlockHeight, b.TxId}
}

type ContractTxHistoryInfo struct {
	ContractName string `gorm:"size:128;primaryKey"`
	BlockHeight  uint64 `gorm:"primaryKey"`
	TxId         string `gorm:"size:128;primaryKey"`
	AccountId    []byte `gorm:"size:2048"`
}

func (b *ContractTxHistoryInfo) GetCreateTableSql(dbType string) string {
	if dbType == conf.SqldbconfigSqldbtypeMysql {
		return `CREATE TABLE contract_tx_history_infos (
    contract_name varchar(128),block_height bigint unsigned,tx_id varchar(128),
    account_id varbinary(2048),PRIMARY KEY (contract_name,block_height,tx_id)
    ) default character set utf8`
	} else if dbType == conf.SqldbconfigSqldbtypeSqlite {
		return `CREATE TABLE contract_tx_history_infos (
    contract_name text,block_height integer,tx_id text,account_id blob,
    PRIMARY KEY (contract_name,block_height,tx_id)
    )`
	}
	panic("Unsupported db type:" + dbType)
}
func (b *ContractTxHistoryInfo) GetTableName() string {
	return "contract_tx_history_infos"
}
func (b *ContractTxHistoryInfo) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO contract_tx_history_infos values(?,?,?,?)",
		[]interface{}{b.ContractName, b.BlockHeight, b.TxId, b.AccountId}
}
func (b *ContractTxHistoryInfo) GetUpdateSql() (string, []interface{}) {
	return `UPDATE contract_tx_history_infos 
set account_id=?
WHERE contract_name=? and block_height=? and tx_id=?`,
		[]interface{}{b.AccountId, b.ContractName, b.BlockHeight, b.TxId}
}
func (b *ContractTxHistoryInfo) GetCountSql() (string, []interface{}) {
	return "SELECT count(*) FROM contract_tx_history_infos" +
			" WHERE contract_name=? and block_height=? and tx_id=?",
		[]interface{}{b.ContractName, b.BlockHeight, b.TxId}
}
