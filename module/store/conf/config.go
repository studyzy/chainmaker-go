/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package conf

import "strings"

type StorageConfig struct {
	//默认的Leveldb配置，如果每个DB有不同的设置，可以在自己的DB中进行设置
	StorePath            string `mapstructure:"store_path"`
	DbPrefix             string `mapstructure:"db_prefix"`
	WriteBufferSize      int    `mapstructure:"write_buffer_size"`
	BloomFilterBits      int    `mapstructure:"bloom_filter_bits"`
	BlockWriteBufferSize int    `mapstructure:"block_write_buffer_size"`
	//数据库模式：light只存区块头,normal存储区块头和交易以及生成的State,full存储了区块头、交易、状态和交易收据（读写集、日志等）
	//Mode string `mapstructure:"mode"`
	DisableHistoryDB       bool      `mapstructure:"disable_historydb"`
	DisableResultDB        bool      `mapstructure:"disable_resultdb"`
	DisableContractEventDB bool      `mapstructure:"disable_contract_eventdb"`
	LogDBWriteAsync        bool      `mapstructure:"logdb_write_async"`
	BlockDbConfig          *DbConfig `mapstructure:"blockdb_config"`
	StateDbConfig          *DbConfig `mapstructure:"statedb_config"`
	HistoryDbConfig        *DbConfig `mapstructure:"historydb_config"`
	ResultDbConfig         *DbConfig `mapstructure:"resultdb_config"`
	ContractEventDbConfig  *DbConfig `mapstructure:"contract_eventdb_config"`
	UnArchiveBlockHeight   uint64    `mapstructure:"unarchive_block_height"`
}

func (config *StorageConfig) setDefault() {
	//if config.DbPrefix != "" {
	//	if config.BlockDbConfig != nil && config.BlockDbConfig.SqlDbConfig != nil &&
	//		config.BlockDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.BlockDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.StateDbConfig != nil && config.StateDbConfig.SqlDbConfig != nil &&
	//		config.StateDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.StateDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.HistoryDbConfig != nil && config.HistoryDbConfig.SqlDbConfig != nil &&
	//		config.HistoryDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.HistoryDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.ResultDbConfig != nil && config.ResultDbConfig.SqlDbConfig != nil &&
	//		config.ResultDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.ResultDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//	if config.ContractEventDbConfig != nil && config.ContractEventDbConfig.SqlDbConfig != nil &&
	//		config.ContractEventDbConfig.SqlDbConfig.DbPrefix == "" {
	//		config.ContractEventDbConfig.SqlDbConfig.DbPrefix = config.DbPrefix
	//	}
	//}
}
func (config *StorageConfig) GetBlockDbConfig() *DbConfig {
	if config.BlockDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.BlockDbConfig
}
func (config *StorageConfig) GetStateDbConfig() *DbConfig {
	if config.StateDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.StateDbConfig
}
func (config *StorageConfig) GetHistoryDbConfig() *DbConfig {
	if config.HistoryDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.HistoryDbConfig
}
func (config *StorageConfig) GetResultDbConfig() *DbConfig {
	if config.ResultDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.ResultDbConfig
}
func (config *StorageConfig) GetContractEventDbConfig() *DbConfig {
	if config.ContractEventDbConfig == nil {
		return config.GetDefaultDBConfig()
	}
	config.setDefault()
	return config.ContractEventDbConfig
}
func (config *StorageConfig) GetDefaultDBConfig() *DbConfig {
	lconfig := make(map[string]interface{})
	lconfig["store_path"] = config.StorePath
	//lconfig := &LevelDbConfig{
	//	StorePath:            config.StorePath,
	//	WriteBufferSize:      config.WriteBufferSize,
	//	BloomFilterBits:      config.BloomFilterBits,
	//	BlockWriteBufferSize: config.WriteBufferSize,
	//}

	//bconfig := &BadgerDbConfig{
	//	StorePath: config.StorePath,
	//}

	return &DbConfig{
		Provider:      "leveldb",
		LevelDbConfig: lconfig,
		//BadgerDbConfig: bconfig,
	}
}

//GetActiveDBCount 根据配置的DisableDB的情况，确定当前配置活跃的数据库数量
func (config *StorageConfig) GetActiveDBCount() int {
	count := 5
	if config.DisableContractEventDB {
		count--
	}
	if config.DisableHistoryDB {
		count--
	}
	if config.DisableResultDB {
		count--
	}
	return count
}

type DbConfig struct {
	//leveldb,badgerdb,sql
	Provider       string                 `mapstructure:"provider"`
	LevelDbConfig  map[string]interface{} `mapstructure:"leveldb_config"`
	BadgerDbConfig map[string]interface{} `mapstructure:"badgerdb_config"`
	SqlDbConfig    map[string]interface{} `mapstructure:"sqldb_config"`
}

func (c DbConfig) GetDbConfig() map[string]interface{} {
	switch strings.ToLower(c.Provider) {
	case "leveldb":
		return c.LevelDbConfig
	case "badgerdb":
		return c.BadgerDbConfig
	case "sqldb":
		return c.SqlDbConfig
	default:
		return map[string]interface{}{}
	}
}

//nolint
const (
	DbconfigProviderSql      = "sql"
	DbconfigProviderLeveldb  = "leveldb"
	DbconfigProviderBadgerdb = "badgerdb"
)

func (dbc *DbConfig) IsKVDB() bool {
	return dbc.Provider == DbconfigProviderLeveldb ||
		//dbc.Provider == DbconfigProviderRocksdb ||
		dbc.Provider == DbconfigProviderBadgerdb
}

func (dbc *DbConfig) IsSqlDB() bool {
	return dbc.Provider == DbconfigProviderSql || dbc.Provider == "mysql" || dbc.Provider == "rdbms" //兼容其他配置情况
}

const SqldbconfigSqldbtypeMysql = "mysql"
const SqldbconfigSqldbtypeSqlite = "sqlite"
