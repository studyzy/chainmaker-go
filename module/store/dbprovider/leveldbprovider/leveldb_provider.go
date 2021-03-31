/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package leveldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"

	"sync"
)

const defaultBloomFilterBits = 10

const (
	StoreBlockDBDir   = "store_block"
	StoreStateDBDir   = "store_state"
	StoreHistoryDBDir = "store_history"
	StoreResultDBDir  = "store_result"
)

var DbNameKeySep = []byte{0x00}

// LevelDBProvider provides handle to db instances
type LevelDBProvider struct {
	dbConfig  *localconf.LevelDbConfig
	dbHandles map[string]protocol.DBHandle
	mutex     sync.Mutex
	dbDir     string
	logger    protocol.Logger
}

// NewBlockProvider construct a new LevelDBProvider for block operation with given chainId
func NewBlockProvider(chainId string, dbconfig *localconf.LevelDbConfig, logger protocol.Logger) *LevelDBProvider {
	return NewLevelDBProvider(chainId, StoreBlockDBDir, dbconfig, logger)
}

// NewStateProvider construct a new LevelDBProvider for state operation with given chainId
func NewStateProvider(chainId string, dbconfig *localconf.LevelDbConfig, logger protocol.Logger) *LevelDBProvider {
	return NewLevelDBProvider(chainId, StoreStateDBDir, dbconfig, logger)
}

// NewStateProvider construct a new LevelDBProvider for state operation with given chainId
func NewHistoryProvider(chainId string, dbconfig *localconf.LevelDbConfig, logger protocol.Logger) *LevelDBProvider {
	return NewLevelDBProvider(chainId, StoreHistoryDBDir, dbconfig, logger)
}

// NewLevelDBProvider construct a new db LevelDBProvider for given chainId and dir
func NewLevelDBProvider(chainId string, dbDir string, dbconfig *localconf.LevelDbConfig, logger protocol.Logger) *LevelDBProvider {
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}

	return &LevelDBProvider{
		dbConfig:  dbconfig,
		dbHandles: make(map[string]protocol.DBHandle),
		mutex:     sync.Mutex{},
		dbDir:     dbDir,
		logger:    logger,
	}
}

// GetDBHandle returns a DBHandle for given dbname
func (p *LevelDBProvider) GetDBHandle(dbName string) protocol.DBHandle {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	dbHandle, exist := p.dbHandles[dbName]
	if exist {
		return dbHandle
	}
	dbHandle = NewLevelDBHandle(dbName, p.dbDir, p.dbConfig, p.logger)
	p.dbHandles[dbName] = dbHandle

	return dbHandle
}

// Close is used to close database
func (p *LevelDBProvider) Close() error {
	for _, h := range p.dbHandles {
		err := h.Close()
		if err != nil {
			p.logger.Errorf("close leveldbprovider, err:%s", err.Error())
		}
	}
	return nil
}
