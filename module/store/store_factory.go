// +build !rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blockkvdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blocksqldb"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/dbprovider/leveldbprovider"
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
	"golang.org/x/sync/semaphore"
	"runtime"
	"strings"
)

// Factory is a factory function to create an instance of the block store
// which commits block into the ledger.
type Factory struct {
}

// NewStore constructs new BlockStore
func (m *Factory) NewStore(chainId string, storeConfig *localconf.StorageConfig, logger protocol.Logger) (protocol.BlockchainStore, error) {
	return m.newStore(chainId, storeConfig, nil, logger)
}

func (m *Factory) newStore(chainId string, storeConfig *localconf.StorageConfig, binLog binlog.BinLoger, logger protocol.Logger) (protocol.BlockchainStore, error) {
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	var blockDB blockdb.BlockDB
	var err error
	if storeConfig.BlockDbConfig.IsKVDB() {
		blockDB, err = m.NewBlockKvDB(chainId, parseEngineType(storeConfig.BlockDbConfig.DbType),
			storeConfig.BlockDbConfig.LevelDbConfig, logger)
		if err != nil {
			return nil, err
		}
	} else {
		blockDB, err = blocksqldb.NewBlockSqlDB(chainId, storeConfig.BlockDbConfig.SqlDbConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	var stateDB statedb.StateDB
	if storeConfig.StateDbConfig.IsKVDB() {
		stateDB, err = m.NewStateKvDB(chainId, parseEngineType(storeConfig.StateDbConfig.DbType),
			storeConfig.StateDbConfig.LevelDbConfig, logger)
		if err != nil {
			return nil, err
		}
	} else {
		stateDB, err = statesqldb.NewStateSqlDB(chainId, storeConfig.StateDbConfig.SqlDbConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	var historyDB historydb.HistoryDB
	if !storeConfig.DisableHistoryDB {
		if storeConfig.HistoryDbConfig.IsKVDB() {
			historyDB, err = m.NewHistoryKvDB(chainId, parseEngineType(storeConfig.HistoryDbConfig.DbType),
				storeConfig.HistoryDbConfig.LevelDbConfig, logger)
			if err != nil {
				return nil, err
			}
		} else {
			historyDB, err = historysqldb.NewHistorySqlDB(chainId, storeConfig.HistoryDbConfig.SqlDbConfig, logger)
			if err != nil {
				return nil, err
			}
		}
	}
	var resultDB resultdb.ResultDB
	if !storeConfig.DisableResultDB {
		if storeConfig.ResultDbConfig.IsKVDB() {
			resultDB, err = m.NewHistoryKvDB(chainId, parseEngineType(storeConfig.ResultDbConfig.DbType),
				storeConfig.ResultDbConfig.LevelDbConfig, logger)
			if err != nil {
				return nil, err
			}
		} else {
			resultDB, err = resultsqldb.NewResultSqlDB(chainId, storeConfig.ResultDbConfig.SqlDbConfig, logger)
			if err != nil {
				return nil, err
			}
		}
	}
	return NewBlockStoreImpl(chainId, blockDB, stateDB, historyDB, resultDB,
		leveldbprovider.NewBlockProvider(chainId, storeConfig.HistoryDbConfig.LevelDbConfig, logger),
		storeConfig, binLog, logger)

}

func parseEngineType(dbType string) types.EngineType {
	var storeType types.EngineType
	switch strings.ToLower(dbType) {
	case "leveldb":
		storeType = types.LevelDb
	case "rocksdb":
		storeType = types.RocksDb
	case "mysql":
		storeType = types.MySQL
	case "sqlite":
		storeType = types.Sqlite
	default:
		return types.UnknowDb
	}
	return storeType
}

// NewBlockKvDB constructs new `BlockDB`
func (m *Factory) NewBlockKvDB(chainId string, engineType types.EngineType, config *localconf.LevelDbConfig, logger protocol.Logger) (blockdb.BlockDB, error) {
	nWorkers := runtime.NumCPU()
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	blockDB := &blockkvdb.BlockKvDB{
		WorkersSemaphore: semaphore.NewWeighted(int64(nWorkers)),
		Cache:            cache.NewStoreCacheMgr(chainId, logger),

		Logger: logger,
	}
	switch engineType {
	case types.LevelDb:
		blockDB.DbHandle = leveldbprovider.NewLevelDBHandle(chainId, leveldbprovider.StoreBlockDBDir, config, logger)
	default:
		return nil, nil
	}
	return blockDB, nil
}

// NewStateKvDB constructs new `StabeKvDB`
func (m *Factory) NewStateKvDB(chainId string, engineType types.EngineType, config *localconf.LevelDbConfig, logger protocol.Logger) (statedb.StateDB, error) {
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	stateDB := &statekvdb.StateKvDB{
		Logger: logger,
		Cache:  cache.NewStoreCacheMgr(chainId, logger),
	}
	switch engineType {
	case types.LevelDb:
		stateDB.DbHandle = leveldbprovider.NewLevelDBHandle(chainId, leveldbprovider.StoreBlockDBDir, config, logger)
	default:
		return nil, nil
	}
	return stateDB, nil
}

// NewHistoryKvDB constructs new `HistoryKvDB`
func (m *Factory) NewHistoryKvDB(chainId string, engineType types.EngineType, config *localconf.LevelDbConfig, logger protocol.Logger) (*historykvdb.HistoryKvDB, error) {
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	historyDB := &historykvdb.HistoryKvDB{
		Cache:  cache.NewStoreCacheMgr(chainId, logger),
		Logger: logger,
	}
	switch engineType {
	case types.LevelDb:
		historyDB.DbHandle = leveldbprovider.NewLevelDBHandle(chainId, leveldbprovider.StoreHistoryDBDir, config, logger)
	default:
		return nil, nil
	}
	return historyDB, nil
}
func (m *Factory) NewResultKvDB(chainId string, engineType types.EngineType, config *localconf.LevelDbConfig, logger protocol.Logger) (*resultkvdb.ResultKvDB, error) {
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	resultDB := &resultkvdb.ResultKvDB{
		Cache:  cache.NewStoreCacheMgr(chainId, logger),
		Logger: logger,
	}
	switch engineType {
	case types.LevelDb:
		resultDB.DbHandle = leveldbprovider.NewLevelDBHandle(chainId, leveldbprovider.StoreHistoryDBDir, config, logger)
	default:
		return nil, nil
	}
	return resultDB, nil
}

//
//// NewKvDBProvider constructs new kv database
//func NewKvDBProvider(chainId string, dbDir string, engineType types.EngineType,config *localconf.LevelDbConfig, logger protocol.Logger) dbprovider.Provider {
//	switch engineType {
//	case types.LevelDb:
//		return leveldbprovider.NewLevelDBHandle(chainId,leveldbprovider.StoreHistoryDBDir, config, logger)
//	}
//	return nil
//}
