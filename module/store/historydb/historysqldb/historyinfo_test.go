package historysqldb

import (
	"testing"

	"chainmaker.org/chainmaker-go/localconf"
	"github.com/stretchr/testify/assert"
)

func TestAccountTxHistoryInfo_GetCreateTableSql(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, err.(string), "Unsupported db type:test")
	}()
	accountTxHistoryInfo := &AccountTxHistoryInfo{}
	sql := accountTxHistoryInfo.GetCreateTableSql(localconf.SqldbconfigSqldbtypeMysql)
	assert.Equal(t, sql, `CREATE TABLE account_tx_history_infos (
    account_id varbinary(2048),block_height bigint unsigned,tx_id varchar(128),
    PRIMARY KEY (account_id,block_height,tx_id)
    ) default character set utf8`)

	// Unsupported db should panic
	sql = accountTxHistoryInfo.GetCreateTableSql("test")
}

func TestAccountTxHistoryInfo_GetUpdateSql(t *testing.T) {
	accountTxHistoryInfo := &AccountTxHistoryInfo{}
	sql, _ := accountTxHistoryInfo.GetUpdateSql()
	assert.Equal(t, sql, "UPDATE account_tx_history_infos set account_id=?"+
		" WHERE account_id=? and block_height=? and tx_id=?")
}

func TestContractTxHistoryInfo_GetCreateTableSql(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, err.(string), "Unsupported db type:test")
	}()
	contractTxHistoryInfo := &ContractTxHistoryInfo{}
	sql := contractTxHistoryInfo.GetCreateTableSql(localconf.SqldbconfigSqldbtypeMysql)
	assert.Equal(t, sql, `CREATE TABLE contract_tx_history_infos (
    contract_name varchar(128),block_height bigint unsigned,tx_id varchar(128),
    account_id varbinary(2048),PRIMARY KEY (contract_name,block_height,tx_id)
    ) default character set utf8`)

	// Unsupported db should panic
	sql = contractTxHistoryInfo.GetCreateTableSql("test")
}

func TestContractTxHistoryInfo_GetUpdateSql(t *testing.T) {
	contractTxHistoryInfo := &ContractTxHistoryInfo{}
	sql, _ := contractTxHistoryInfo.GetUpdateSql()
	assert.Equal(t, sql, `UPDATE contract_tx_history_infos 
set account_id=?
WHERE contract_name=? and block_height=? and tx_id=?`)
}

func TestStateHistoryInfo_GetCreateTableSql(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, err.(string), "Unsupported db type:test")
	}()
	stateHistoryInfo := &StateHistoryInfo{}
	sql := stateHistoryInfo.GetCreateTableSql(localconf.SqldbconfigSqldbtypeMysql)
	assert.Equal(t, sql, `CREATE TABLE state_history_infos (
    contract_name varchar(128),state_key varbinary(128),tx_id varchar(128),block_height bigint unsigned,
    PRIMARY KEY (contract_name,state_key,tx_id,block_height)
    ) default character set utf8`)

	// Unsupported db should panic
	sql = stateHistoryInfo.GetCreateTableSql("test")
}

func TestStateHistoryInfo_GetUpdateSql(t *testing.T) {
	stateHistoryInfo := &StateHistoryInfo{}
	sql, _ := stateHistoryInfo.GetUpdateSql()
	assert.Equal(t, sql, `UPDATE state_history_infos set contract_name=?
			 WHERE contract_name=? and state_key=? and tx_id=? and block_height=?`)
}
