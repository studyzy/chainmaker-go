//go:build !rocksdb
// +build !rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"errors"
	"strings"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blockkvdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blocksqldb"
	"chainmaker.org/chainmaker-go/store/cache"
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
	"chainmaker.org/chainmaker-go/store/types"
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
func (m *Factory) NewStore(chainId string, storeConfig *localconf.StorageConfig,
	logger protocol.Logger) (protocol.BlockchainStore, error) {
	return m.newStore(chainId, storeConfig, nil, logger)
}

func (m *Factory) newStore(chainId string, storeConfig *localconf.StorageConfig, binLog binlog.BinLoger,
	logger protocol.Logger) (protocol.BlockchainStore, error) {

	var blockDB blockdb.BlockDB
	var err error
	blocDBConfig := storeConfig.GetBlockDbConfig()
	if blocDBConfig.IsKVDB() {
		blockDB, err = m.NewBlockKvDB(chainId, blocDBConfig.Provider,
			blocDBConfig, logger)
		if err != nil {
			return nil, err
		}
	} else {
		blockDB, err = blocksqldb.NewBlockSqlDB(chainId, blocDBConfig.SqlDbConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	var stateDB statedb.StateDB
	stateDBConfig := storeConfig.GetStateDbConfig()
	if stateDBConfig.IsKVDB() {
		stateDB, err = m.NewStateKvDB(chainId, stateDBConfig.Provider,
			stateDBConfig, logger)
		if err != nil {
			return nil, err
		}
	} else {
		stateDB, err = statesqldb.NewStateSqlDB(chainId, stateDBConfig.SqlDbConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	var historyDB historydb.HistoryDB
	historyDBConfig := storeConfig.GetHistoryDbConfig()
	if !storeConfig.DisableHistoryDB {
		if historyDBConfig.IsKVDB() {
			historyDB, err = m.NewHistoryKvDB(chainId, historyDBConfig.Provider,
				historyDBConfig, logger)
			if err != nil {
				return nil, err
			}
		} else {
			historyDB, err = historysqldb.NewHistorySqlDB(chainId, historyDBConfig.SqlDbConfig, logger)
			if err != nil {
				return nil, err
			}
		}
	}
	var resultDB resultdb.ResultDB
	resultDBConfig := storeConfig.GetResultDbConfig()
	if !storeConfig.DisableResultDB {
		if resultDBConfig.IsKVDB() {
			resultDB, err = m.NewResultKvDB(chainId, resultDBConfig.Provider,
				resultDBConfig, logger)
			if err != nil {
				return nil, err
			}
		} else {
			resultDB, err = resultsqldb.NewResultSqlDB(chainId, resultDBConfig.SqlDbConfig, logger)
			if err != nil {
				return nil, err
			}
		}
	}
	var contractEventDB contracteventdb.ContractEventDB
	contractEventDBConfig := storeConfig.GetContractEventDbConfig()
	if !storeConfig.DisableContractEventDB {
		if parseEngineType(storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
			storeConfig.ContractEventDbConfig.Provider == "sql" {
			contractEventDB, err = eventsqldb.NewContractEventMysqlDB(chainId,
				contractEventDBConfig.SqlDbConfig, logger)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("contract event db config err")
		}
	}
	return NewBlockStoreImpl(chainId, blockDB, stateDB, historyDB, contractEventDB, resultDB,
		getLocalCommonDB(chainId, storeConfig, logger),
		storeConfig, binLog, logger)
}

func getLocalCommonDB(chainId string, config *localconf.StorageConfig, log protocol.Logger) protocol.DBHandle {

	//storeType := parseEngineType(config.BlockDbConfig.Provider)
	//if storeType == types.BadgerDb {
	//	return badgerdbprovider.NewBadgerDBHandle(chainId, StoreLocalDBDir,
	//		config.GetDefaultDBConfig().BadgerDbConfig, log)
	//}
	dbHandle, _ := dbFactory.NewKvDB(chainId, "leveldb", StoreLocalDBDir,
		config.GetDefaultDBConfig().LevelDbConfig, log)
	return dbHandle
}

func parseEngineType(dbType string) types.EngineType {
	var storeType types.EngineType
	switch strings.ToLower(dbType) {
	case "leveldb":
		storeType = types.LevelDb
	case "badgerdb":
		storeType = types.BadgerDb
	case "mysql":
		storeType = types.MySQL
	case "sqlite":
		storeType = types.Sqlite
	default:
		return types.UnknownDb
	}
	return storeType
}

// NewBlockKvDB constructs new `BlockDB`
func (m *Factory) NewBlockKvDB(chainId string, provider string, dbConfig *localconf.DbConfig,
	logger protocol.Logger) (blockdb.BlockDB, error) {

	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreBlockDBDir, dbConfig.LevelDbConfig, logger)
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

// NewStateKvDB constructs new `StabeKvDB`
func (m *Factory) NewStateKvDB(chainId string, provider string, dbConfig *localconf.DbConfig,
	logger protocol.Logger) (statedb.StateDB, error) {

	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreStateDBDir, dbConfig.LevelDbConfig, logger)
	if err != nil {
		return nil, err
	}
	stateDB := statekvdb.NewStateKvDB(chainId, dbHandle, logger)
	return stateDB, nil
}

// NewHistoryKvDB constructs new `HistoryKvDB`
func (m *Factory) NewHistoryKvDB(chainId string, provider string, dbConfig *localconf.DbConfig,
	logger protocol.Logger) (*historykvdb.HistoryKvDB, error) {
	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreHistoryDBDir, dbConfig.LevelDbConfig, logger)
	if err != nil {
		return nil, err
	}

	historyDB := historykvdb.NewHistoryKvDB(dbHandle,
		cache.NewStoreCacheMgr(chainId, 10, logger), logger)
	return historyDB, nil
}

func (m *Factory) NewResultKvDB(chainId string, provider string, dbConfig *localconf.DbConfig,
	logger protocol.Logger) (*resultkvdb.ResultKvDB, error) {
	dbHandle, err := dbFactory.NewKvDB(chainId, provider, StoreResultDBDir, dbConfig.LevelDbConfig, logger)
	if err != nil {
		return nil, err
	}
	resultDB := resultkvdb.NewResultKvDB(chainId, dbHandle, logger)
	return resultDB, nil
}
