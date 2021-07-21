// +build rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"errors"
	"runtime"
	"strings"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blockkvdb"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/contracteventdb"
	"chainmaker.org/chainmaker-go/store/contracteventdb/eventsqldb"
	"chainmaker.org/chainmaker-go/store/dbprovider/rocksdbprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/historydb/historykvdb"
	"chainmaker.org/chainmaker-go/store/resultdb"
	"chainmaker.org/chainmaker-go/store/resultdb/resultkvdb"
	"chainmaker.org/chainmaker-go/store/statedb"
	"chainmaker.org/chainmaker-go/store/statedb/statekvdb"
	"chainmaker.org/chainmaker-go/store/types"
	"golang.org/x/sync/semaphore"
)

// Factory is a factory function to create an instance of the block store
// which commits block into the ledger.
type Factory struct {
}

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
	if !blocDBConfig.IsKVDB() {
		return nil, errors.New("please use rocksdb in this mode")
	}

	blockDB, err = m.NewBlockKvDB(chainId, parseEngineType(blocDBConfig.Provider),
		blocDBConfig.RocksDbConfig, logger)
	if err != nil {
		return nil, err
	}

	var stateDB statedb.StateDB
	stateDBConfig := storeConfig.GetStateDbConfig()
	stateDB, err = m.NewStateKvDB(chainId, parseEngineType(stateDBConfig.Provider),
		stateDBConfig.RocksDbConfig, logger)
	if err != nil {
		return nil, err
	}
	var historyDB historydb.HistoryDB
	historyDBConfig := storeConfig.GetHistoryDbConfig()
	if !storeConfig.DisableHistoryDB {
		historyDB, err = m.NewHistoryKvDB(chainId, parseEngineType(historyDBConfig.Provider),
			historyDBConfig.RocksDbConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	var resultDB resultdb.ResultDB
	resultDBConfig := storeConfig.GetResultDbConfig()
	if !storeConfig.DisableResultDB {
		resultDB, err = m.NewResultKvDB(chainId, parseEngineType(resultDBConfig.Provider),
			resultDBConfig.RocksDbConfig, logger)
		if err != nil {
			return nil, err
		}
	}
	var contractEventDB contracteventdb.ContractEventDB
	contractEventDBConfig := storeConfig.GetContractEventDbConfig()
	if !storeConfig.DisableContractEventDB {
		if parseEngineType(storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
			storeConfig.ContractEventDbConfig.Provider == "sql" {
			contractEventDB, err = eventsqldb.NewContractEventMysqlDB(chainId, contractEventDBConfig.SqlDbConfig, logger)
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
	return rocksdbprovider.NewRocksDBHandle(chainId, "localdb", config.GetDefaultDBConfig().RocksDbConfig, log)
}

func parseEngineType(dbType string) types.EngineType {
	var storeType types.EngineType
	switch strings.ToLower(dbType) {
	case "rocksdb":
		storeType = types.RocksDb
	default:
		return types.UnknownDb
	}
	return storeType
}

// NewBlockKvDB constructs new `BlockDB`
func (m *Factory) NewBlockKvDB(chainId string, engineType types.EngineType, config *localconf.RocksDbConfig,
	logger protocol.Logger) (blockdb.BlockDB, error) {
	nWorkers := runtime.NumCPU()
	blockDB := &blockkvdb.BlockKvDB{
		WorkersSemaphore: semaphore.NewWeighted(int64(nWorkers)),
		Cache:            cache.NewStoreCacheMgr(chainId, logger),
		Logger:           logger,
	}
	switch engineType {
	case types.RocksDb:
		blockDB.DbHandle = rocksdbprovider.NewRocksDBHandle(chainId, rocksdbprovider.StoreBlockDBDir, config, logger)
	default:
		return nil, nil
	}
	return blockDB, nil
}

// NewStateKvDB constructs new `StabeKvDB`
func (m *Factory) NewStateKvDB(chainId string, engineType types.EngineType, config *localconf.RocksDbConfig,
	logger protocol.Logger) (statedb.StateDB, error) {
	stateDB := &statekvdb.StateKvDB{
		Logger: logger,
		Cache:  cache.NewStoreCacheMgr(chainId, logger),
	}
	switch engineType {
	case types.RocksDb:
		stateDB.DbHandle = rocksdbprovider.NewRocksDBHandle(chainId, rocksdbprovider.StoreStateDBDir, config, logger)
	default:
		return nil, nil
	}
	return stateDB, nil
}

// NewHistoryKvDB constructs new `HistoryKvDB`
func (m *Factory) NewHistoryKvDB(chainId string, engineType types.EngineType, config *localconf.RocksDbConfig,
	logger protocol.Logger) (*historykvdb.HistoryKvDB, error) {
	var db protocol.DBHandle
	switch engineType {
	case types.RocksDb:
		db = rocksdbprovider.NewRocksDBHandle(chainId, rocksdbprovider.StoreHistoryDBDir, config, logger)
	default:
		return nil, errors.New("invalid db type")
	}
	historyDB := historykvdb.NewHistoryKvDB(db, cache.NewStoreCacheMgr(chainId, logger), logger)
	return historyDB, nil
}

func (m *Factory) NewResultKvDB(chainId string, engineType types.EngineType, config *localconf.RocksDbConfig,
	logger protocol.Logger) (*resultkvdb.ResultKvDB, error) {
	var db protocol.DBHandle
	switch engineType {
	case types.RocksDb:
		db = rocksdbprovider.NewRocksDBHandle(chainId, rocksdbprovider.StoreResultDBDir, config, logger)
	default:
		return nil, errors.New("invalid db type")
	}
	resultDB := &resultkvdb.ResultKvDB{
		Cache:    cache.NewStoreCacheMgr(chainId, logger),
		Logger:   logger,
		DbHandle: db,
	}
	return resultDB, nil
}