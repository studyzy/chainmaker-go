/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemysqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/sqldbprovider"
	"chainmaker.org/chainmaker-go/store/statedb"
	"fmt"
	"gorm.io/gorm"
)

// StateMysqlDB provider a implementation of `statedb.StateDB`
// This implementation provides a mysql based data model
type StateMysqlDB struct {
	db     *gorm.DB
	Logger protocol.Logger
}

// NewStateMysqlDB construct a new `StateDB` for given chainId
func NewStateMysqlDB(chainId string, logger protocol.Logger) (statedb.StateDB, error) {
	db := sqldbprovider.NewProvider().GetDB(chainId, localconf.ChainMakerConfig)
	if logger == nil {
		logger = logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId)
	}
	if err := db.AutoMigrate(&StateInfo{}); err != nil {
		panic(fmt.Sprintf("failed to migrate blockinfo:%s", err))
	}
	stateDB := &StateMysqlDB{
		db:     db,
		Logger: logger,
	}
	return stateDB, nil
}

// CommitBlock commits the state in an atomic operation
func (s *StateMysqlDB) CommitBlock(blockWithRWSet *storePb.BlockWithRWSet) error {
	block := blockWithRWSet.Block
	txRWSets := blockWithRWSet.TxRWSets
	stateInfos := make([]*StateInfo, 0, len(txRWSets))
	for _, txRWSet := range txRWSets {
		for _, txWrite := range txRWSet.TxWrites {
			stateInfo := NewStateInfo(txWrite.ContractName, txWrite.Key, txWrite.Value, block.Header.BlockHeight)
			if txWrite.Key == nil {
				s.Logger.Warnf("object_key cannot be null, block[%d], txid[%s] ",
					block.Header.BlockHeight, txRWSet.TxId)
			}
			stateInfos = append(stateInfos, stateInfo)
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, stateInfo := range stateInfos {
			var res *gorm.DB
			if stateInfo.ObjectValue == nil {
				res = tx.Delete(stateInfo)
			} else {
				res = tx.Save(stateInfo)
			}
			if res.Error != nil {
				s.Logger.Errorf("failed to set state, contract:%s, key:%s, err:%s",
					stateInfo.ContractName, stateInfo.ObjectKey, res.Error)
				return res.Error
			}
		}
		s.Logger.Debugf("chain[%s]: commit state block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return nil
	})

}

// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
func (s *StateMysqlDB) ReadObject(contractName string, key []byte) ([]byte, error) {
	var stateInfo StateInfo
	res := s.db.Find(&stateInfo, &StateInfo{ContractName: contractName, ObjectKey: key})
	if res.Error == gorm.ErrRecordNotFound {
		return nil, nil
	} else if res.Error != nil {
		s.Logger.Errorf("failed to read state, contract:%s, key:%s", contractName, key)
		return nil, res.Error
	}
	return stateInfo.ObjectValue, nil
}

// SelectObject returns an iterator that contains all the key-values between given key ranges.
// startKey is included in the results and limit is excluded.
func (s *StateMysqlDB) SelectObject(contractName string, startKey []byte, limit []byte) protocol.Iterator {
	//todo
	panic("selectObject not implemented!")
}

// GetLastSavepoint returns the last block height
func (s *StateMysqlDB) GetLastSavepoint() (uint64, error) {
	var stateInfo StateInfo
	res := s.db.Order("block_height desc").Limit(1).Find(&stateInfo)
	if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
		s.Logger.Errorf("failed to get last savepoint")
		return 0, res.Error
	}
	return uint64(stateInfo.BlockHeight), nil
}

// Close is used to close database, there is no need for gorm to close db
func (s *StateMysqlDB) Close() {
	sqlDB, err := s.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}
