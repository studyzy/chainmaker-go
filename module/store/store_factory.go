// +build !rocksdb

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	logImpl "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blockkvdb"
	"chainmaker.org/chainmaker-go/store/blockdb/blockmysqldb"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/dbprovider"
	"chainmaker.org/chainmaker-go/store/dbprovider/leveldbprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/historydb/historykvdb"
	"chainmaker.org/chainmaker-go/store/historydb/historymysqldb"
	"chainmaker.org/chainmaker-go/store/statedb"
	"chainmaker.org/chainmaker-go/store/statedb/statekvdb"
	"chainmaker.org/chainmaker-go/store/statedb/statemysqldb"
	"chainmaker.org/chainmaker-go/store/types"
	"golang.org/x/sync/semaphore"
	"runtime"
)

// Factory is a factory function to create an instance of the block store
// which commits block into the ledger.
type Factory struct {
}

// NewStore constructs new BlockStore
func (m *Factory) NewStore(engineType types.EngineType, chainId string) (protocol.BlockchainStore, error) {
	switch engineType {
	case types.LevelDb:
		blockDB, err := m.NewBlockKvDB(chainId, engineType)
		if err != nil {
			return nil, err
		}
		stateDB, err := m.NewStateKvDB(chainId, engineType)
		if err != nil {
			return nil, err
		}
		historyDB, err := m.NewHistoryKvDB(chainId, engineType)
		if err != nil {
			return nil, err
		}
		return NewBlockStoreImpl(chainId, blockDB, stateDB, historyDB, NewKvDBProvider(chainId, types.CommonDBDir, engineType))
	case types.MySQL:
		blockDB, err := blockmysqldb.NewBlockMysqlDB(chainId)
		if err != nil {
			return nil, err
		}
		stateDB, err := statemysqldb.NewStateMysqlDB(chainId)
		if err != nil {
			return nil, err
		}
		historyDB, err := historymysqldb.NewHistoryMysqlDB(chainId)
		if err != nil {
			return nil, err
		}
		return NewBlockStoreImpl(chainId, blockDB, stateDB, historyDB, NewKvDBProvider(chainId, types.CommonDBDir, types.LevelDb))
	default:
		return nil, nil
	}
}

// NewBlockKvDB constructs new `BlockDB`
func (m *Factory) NewBlockKvDB(chainId string, engineType types.EngineType) (blockdb.BlockDB, error) {
	nWorkers := runtime.NumCPU()
	blockDB := &blockkvdb.BlockKvDB{
		WorkersSemaphore: semaphore.NewWeighted(int64(nWorkers)),
		Cache:            cache.NewStoreCacheMgr(chainId),

		Logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
	}
	switch engineType {
	case types.LevelDb:
		blockDB.DbProvider = leveldbprovider.NewBlockProvider(chainId)
	default:
		return nil, nil
	}
	return blockDB, nil
}

// NewStateKvDB constructs new `StabeKvDB`
func (m *Factory) NewStateKvDB(chainId string, engineType types.EngineType) (statedb.StateDB, error) {
	stateDB := &statekvdb.StateKvDB{
		Logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
		Cache:  cache.NewStoreCacheMgr(chainId),
	}
	switch engineType {
	case types.LevelDb:
		stateDB.DbProvider = leveldbprovider.NewStateProvider(chainId)
	default:
		return nil, nil
	}
	return stateDB, nil
}

// NewHistoryKvDB constructs new `HistoryKvDB`
func (m *Factory) NewHistoryKvDB(chainId string, engineType types.EngineType) (historydb.HistoryDB, error) {
	historyDB := &historykvdb.HistoryKvDB{
		Cache:  cache.NewStoreCacheMgr(chainId),
		Logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
	}
	switch engineType {
	case types.LevelDb:
		historyDB.DbProvider = leveldbprovider.NewHistoryProvider(chainId)
	default:
		return nil, nil
	}
	return historyDB, nil
}

// NewKvDBProvider constructs new kv database
func NewKvDBProvider(chainId string, dbDir string, engineType types.EngineType) dbprovider.Provider {
	switch engineType {
	case types.LevelDb:
		return leveldbprovider.NewProvider(chainId, dbDir)
	}
	return nil
}
