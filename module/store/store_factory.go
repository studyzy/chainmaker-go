//go:build !rocksdb
// +build !rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blockkvdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blocksqldb"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/conf"
	"chainmaker.org/chainmaker-go/store/contracteventdb"
	"chainmaker.org/chainmaker-go/store/contracteventdb/eventsqldb"
	"chainmaker.org/chainmaker-go/store/dbprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/historydb/historykvdb"
	"chainmaker.org/chainmaker-go/store/historydb/historysqldb"
	"chainmaker.org/chainmaker-go/store/resultdb"
	"chainmaker.org/chainmaker-go/store/resultdb/resultkvdb"
	"chainmaker.org/chainmaker-go/store/resultdb/resultsqldb"
	"chainmaker.org/chainmaker-go/store/statedb"
	"chainmaker.org/chainmaker-go/store/statedb/statekvdb"
	"chainmaker.org/chainmaker-go/store/statedb/statesqldb"
	"chainmaker.org/chainmaker/protocol/v2"
)

const (
	//StoreBlockDBDir blockdb folder name
	StoreBlockDBDir = "store_block"
	//StoreStateDBDir statedb folder name
	StoreStateDBDir = "store_state"
	//StoreHistoryDBDir historydb folder name
	StoreHistoryDBDir = "store_history"
	//StoreResultDBDir resultdb folder name
	StoreResultDBDir = "store_result"
	StoreLocalDBDir  = "localdb"
)

// Factory is a factory function to create an instance of the block store
// which commits block into the ledger.
type Factory struct {
}

var dbFactory = &dbprovider.DBFactory{}

// NewStore constructs new BlockStore
func (m *Factory) NewStore(chainId string, storeConfig *conf.StorageConfig,
	logger protocol.Logger) (protocol.BlockchainStore, error) {
	return m.newStore(chainId, storeConfig, nil, logger)
}

func newBlockDB(chainId string, blockDBConfig *conf.DbConfig, logger protocol.Logger) (
	blockdb.BlockDB, error) {
	if blockDBConfig.IsKVDB() {
		config := blockDBConfig.GetDbConfig()
		dbHandle, err := dbFactory.NewKvDB(chainId, blockDBConfig.Provider, StoreBlockDBDir, config, logger)
		if err != nil {
			return nil, err
		}
		blockDB := blockkvdb.NewBlockKvDB(chainId, dbHandle, logger)
		//Get and update archive pivot
		if _, err := blockDB.GetArchivedPivot(); err != nil {
			return nil, err
		}

		return blockDB, nil
	}
	db, err := dbFactory.NewSqlDB(chainId, blockDBConfig.Provider, "blockdb", blockDBConfig.SqlDbConfig, logger)
	if err != nil {
		return nil, err
	}
	return blocksqldb.NewBlockSqlDB(chainId, db, logger), nil

}

func newStateDB(chainId, dbPrefix string, stateDBConfig *conf.DbConfig, logger protocol.Logger) (
	statedb.StateDB, error) {
	if stateDBConfig.IsKVDB() {
		config := stateDBConfig.GetDbConfig()
		dbHandle, err := dbFactory.NewKvDB(chainId, stateDBConfig.Provider, StoreStateDBDir, config, logger)
		if err != nil {
			return nil, err
		}
		blockDB := statekvdb.NewStateKvDB(chainId, dbHandle, logger)
		return blockDB, nil
	}
	db, err := dbFactory.NewSqlDB(chainId, stateDBConfig.Provider, "statedb", stateDBConfig.SqlDbConfig, logger)
	if err != nil {
		return nil, err
	}
	newDbFunc := func(dbName string) (protocol.SqlDBHandle, error) {
		return dbFactory.NewSqlDB(chainId, stateDBConfig.Provider, dbName, stateDBConfig.SqlDbConfig, logger)
	}

	return statesqldb.NewStateSqlDB(dbPrefix, chainId, db, newDbFunc, logger)

}

func newHistoryDB(chainId string, historyDBConfig *conf.DbConfig, logger protocol.Logger) (
	historydb.HistoryDB, error) {
	if historyDBConfig.IsKVDB() {
		config := historyDBConfig.GetDbConfig()
		dbHandle, err := dbFactory.NewKvDB(chainId, historyDBConfig.Provider, StoreHistoryDBDir, config, logger)
		if err != nil {
			return nil, err
		}
		cache1 := cache.NewStoreCacheMgr(chainId, 10, logger)
		blockDB := historykvdb.NewHistoryKvDB(dbHandle, cache1, logger)
		return blockDB, nil
	}
	db, err := dbFactory.NewSqlDB(chainId, historyDBConfig.Provider, "historydb", historyDBConfig.SqlDbConfig, logger)
	if err != nil {
		return nil, err
	}
	return historysqldb.NewHistorySqlDB(chainId, db, logger)

}

func newResultDB(chainId string, resultDBConfig *conf.DbConfig, logger protocol.Logger) (
	resultdb.ResultDB, error) {
	if resultDBConfig.IsKVDB() {
		config := resultDBConfig.GetDbConfig()
		dbHandle, err := dbFactory.NewKvDB(chainId, resultDBConfig.Provider, StoreResultDBDir, config, logger)
		if err != nil {
			return nil, err
		}
		resultdb := resultkvdb.NewResultKvDB(chainId, dbHandle, logger)
		return resultdb, nil
	}
	db, err := dbFactory.NewSqlDB(chainId, resultDBConfig.Provider, "resultdb", resultDBConfig.SqlDbConfig, logger)
	if err != nil {
		return nil, err
	}
	return resultsqldb.NewResultSqlDB(chainId, db, logger), nil

}
func newEventDB(chainId string, eventDBConfig *conf.DbConfig, logger protocol.Logger) (
	contracteventdb.ContractEventDB, error) {
	db, err := dbFactory.NewSqlDB(chainId, eventDBConfig.Provider, "eventdb", eventDBConfig.SqlDbConfig, logger)
	if err != nil {
		return nil, err
	}
	return eventsqldb.NewContractEventDB(chainId, db, logger)
}
func (m *Factory) newStore(chainId string, storeConfig *conf.StorageConfig, binLog binlog.BinLoger,
	logger protocol.Logger) (protocol.BlockchainStore, error) {
	//new blockdb
	blockDBConfig := storeConfig.GetBlockDbConfig()
	blockDB, err := newBlockDB(chainId, blockDBConfig, logger)
	if err != nil {
		return nil, err
	}
	//new statedb
	stateDBConfig := storeConfig.GetStateDbConfig()
	stateDB, err := newStateDB(chainId, storeConfig.DbPrefix, stateDBConfig, logger)
	if err != nil {
		return nil, err
	}
	//new historydb
	var historyDB historydb.HistoryDB
	historyDBConfig := storeConfig.GetHistoryDbConfig()
	if !storeConfig.DisableHistoryDB {
		historyDB, err = newHistoryDB(chainId, historyDBConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	//new result db
	var resultDB resultdb.ResultDB
	resultDBConfig := storeConfig.GetResultDbConfig()
	if !storeConfig.DisableResultDB {
		resultDB, err = newResultDB(chainId, resultDBConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	//new contract event db
	var contractEventDB contracteventdb.ContractEventDB
	contractEventDBConfig := storeConfig.GetContractEventDbConfig()
	if !storeConfig.DisableContractEventDB {
		contractEventDB, err = newEventDB(chainId, contractEventDBConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	return NewBlockStoreImpl(chainId, blockDB, stateDB, historyDB, contractEventDB, resultDB,
		getLocalCommonDB(chainId, storeConfig, logger),
		storeConfig, binLog, logger)
}

func getLocalCommonDB(chainId string, config *conf.StorageConfig, log protocol.Logger) protocol.DBHandle {

	//storeType := parseEngineType(config.BlockDbConfig.Provider)
	//if storeType == types.BadgerDb {
	//	return badgerdbprovider.NewBadgerDBHandle(chainId, StoreLocalDBDir,
	//		config.GetDefaultDBConfig().BadgerDbConfig, log)
	//}
	dbHandle, _ := dbFactory.NewKvDB(chainId, "leveldb", StoreLocalDBDir,
		config.GetDefaultDBConfig().LevelDbConfig, log)
	return dbHandle
}

//func parseEngineType(dbType string) types.EngineType {
//	var storeType types.EngineType
//	switch strings.ToLower(dbType) {
//	case "leveldb":
//		storeType = types.LevelDb
//	case "badgerdb":
//		storeType = types.BadgerDb
//	case "mysql":
//		storeType = types.MySQL
//	case "sqlite":
//		storeType = types.Sqlite
//	default:
//		return types.UnknownDb
//	}
//	return storeType
//}

// NewBlockKvDB constructs new `BlockDB`
//func (m *Factory) NewBlockKvDB(chainId string, provider string, dbConfig *conf.DbConfig,
//	logger protocol.Logger) (blockdb.BlockDB, error) {
//
//	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreBlockDBDir, dbConfig.LevelDbConfig, logger)
//	if err != nil {
//		return nil, err
//	}
//	blockDB := blockkvdb.NewBlockKvDB(chainId, dbHandle, logger)
//
//	//Get and update archive pivot
//	if _, err := blockDB.GetArchivedPivot(); err != nil {
//		return nil, err
//	}
//
//	return blockDB, nil
//}
//
//// NewStateKvDB constructs new `StabeKvDB`
//func (m *Factory) NewStateKvDB(chainId string, provider string, dbConfig *conf.DbConfig,
//	logger protocol.Logger) (statedb.StateDB, error) {
//
//	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreStateDBDir, dbConfig.LevelDbConfig, logger)
//	if err != nil {
//		return nil, err
//	}
//	stateDB := statekvdb.NewStateKvDB(chainId, dbHandle, logger)
//	return stateDB, nil
//}
//
//// NewHistoryKvDB constructs new `HistoryKvDB`
//func (m *Factory) NewHistoryKvDB(chainId string, provider string, dbConfig *conf.DbConfig,
//	logger protocol.Logger) (*historykvdb.HistoryKvDB, error) {
//	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreHistoryDBDir, dbConfig.LevelDbConfig, logger)
//	if err != nil {
//		return nil, err
//	}
//
//	historyDB := historykvdb.NewHistoryKvDB(dbHandle,
//		cache.NewStoreCacheMgr(chainId, 10, logger), logger)
//	return historyDB, nil
//}
//
//func (m *Factory) NewResultKvDB(chainId string, provider string, dbConfig *conf.DbConfig,
//	logger protocol.Logger) (*resultkvdb.ResultKvDB, error) {
//	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreResultDBDir, dbConfig.LevelDbConfig, logger)
//	if err != nil {
//		return nil, err
//	}
//	resultDB := resultkvdb.NewResultKvDB(chainId, dbHandle, logger)
//	return resultDB, nil
//}
