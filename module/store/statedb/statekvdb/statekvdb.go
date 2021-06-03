/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statekvdb

import (
	"encoding/binary"
	"errors"
	"fmt"

	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"

	"chainmaker.org/chainmaker-go/utils"

	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"

	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
)

const (
	contractStoreSeparator = '#'
	stateDBSavepointKey    = "stateDBSavePointKey"
)

// StateKvDB provider a implementation of `statedb.StateDB`
// This implementation provides a key-value based data model
type StateKvDB struct {
	DbHandle protocol.DBHandle
	Cache    *cache.StoreCacheMgr
	Logger   protocol.Logger
}

func (s *StateKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	s.Logger.Debug("initial genesis state data into leveldb")
	return s.CommitBlock(genesisBlock)
}

// CommitBlock commits the state in an atomic operation
func (s *StateKvDB) CommitBlock(blockWithRWSet *serialization.BlockWithSerializedInfo) error {
	batch := types.NewUpdateBatch()
	// 1. last block height
	block := blockWithRWSet.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, uint64(block.Header.BlockHeight))
	batch.Put([]byte(stateDBSavepointKey), lastBlockNumBytes)

	txRWSets := blockWithRWSet.TxRWSets
	for _, txRWSet := range txRWSets {
		for _, txWrite := range txRWSet.TxWrites {

			s.operateDbByWriteSet(batch, txWrite)
		}
	}
	//process consensusArgs
	if len(block.Header.ConsensusArgs) > 0 {
		err := s.updateConsensusArgs(batch, block)
		if err != nil {
			return err
		}
	}
	err := s.writeBatch(block.Header.BlockHeight, batch)
	if err != nil {
		return err
	}
	s.Logger.Debugf("chain[%s]: commit state block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil
}

func (s *StateKvDB) updateConsensusArgs(batch protocol.StoreBatcher, block *commonPb.Block) error {
	//try to add consensusArgs
	consensusArgs, err := utils.GetConsensusArgsFromBlock(block)
	if err != nil {
		s.Logger.Errorf("parse header.ConsensusArgs get an error:%s", err)
		return err
	}
	if consensusArgs.ConsensusData != nil {
		s.Logger.Debugf("add consensusArgs ConsensusData to statedb")
		for _, write := range consensusArgs.ConsensusData.TxWrites {
			s.operateDbByWriteSet(batch, write)
		}
	}
	return nil
}

// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
func (s *StateKvDB) ReadObject(contractName string, key []byte) ([]byte, error) {
	objectKey := constructStateKey(contractName, key)
	return s.get(objectKey)
}

// SelectObject returns an iterator that contains all the key-values between given key ranges.
// startKey is included in the results and limit is excluded.
func (s *StateKvDB) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	objectStartKey := constructStateKey(contractName, startKey)
	objectLimitKey := constructStateKey(contractName, limit)
	//todo combine cache and database
	s.Cache.LockForFlush()
	defer s.Cache.UnLockFlush()
	//logger.Debugf("start[%s], limit[%s]", objectStartKey, objectLimitKey)
	iter := s.DbHandle.NewIteratorWithRange(objectStartKey, objectLimitKey)
	return &kvi{
		iter:         iter,
		contractName: contractName,
	}, nil
}

type kvi struct {
	iter         protocol.Iterator
	contractName string
}

func (i *kvi) Next() bool {
	return i.iter.Next()
}
func (i *kvi) Value() (*storePb.KV, error) {
	err := i.iter.Error()
	if err != nil {
		return nil, err
	}
	return &storePb.KV{
		ContractName: i.contractName,
		Key:          i.iter.Key(),
		Value:        i.iter.Value(),
	}, nil
}
func (i *kvi) Release() {
	i.iter.Release()
}

// GetLastSavepoint returns the last block height
func (b *StateKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := b.get([]byte(stateDBSavepointKey))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}
	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// Close is used to close database
func (s *StateKvDB) Close() {
	s.Logger.Info("close state kv db")
	s.DbHandle.Close()
}

func (s *StateKvDB) writeBatch(blockHeight int64, batch protocol.StoreBatcher) error {
	//update cache
	s.Cache.AddBlock(blockHeight, batch)
	go func() {
		err := s.DbHandle.WriteBatch(batch, false)
		if err != nil {
			panic(fmt.Sprintf("Error writing leveldb: %s", err))
		}
		//db committed, clean cache
		s.Cache.DelBlock(blockHeight)
	}()
	return nil
}

func (s *StateKvDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := s.Cache.Get(string(key))
	if exist {
		return value, nil
	}
	//get from database
	return s.DbHandle.Get(key)
}

//func (s *StateKvDB) has(key []byte) (bool, error) {
//	//check has from cache
//	isDelete, exist := s.Cache.Has(string(key))
//	if exist {
//		return !isDelete, nil
//	}
//	return s.DbHandle.Has(key)
//}

func constructStateKey(contractName string, key []byte) []byte {
	return append(append([]byte(contractName), contractStoreSeparator), key...)
}

var errorSqldbOnly = errors.New("leveldb don't support this operation, please change to sql db")

func (s *StateKvDB) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	return nil, errorSqldbOnly
}
func (s *StateKvDB) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	return nil, errorSqldbOnly

}
func (s *StateKvDB) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return nil, errorSqldbOnly

}
func (s *StateKvDB) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return nil, errorSqldbOnly

}
func (s *StateKvDB) CommitDbTransaction(txName string) error {
	return errorSqldbOnly

}
func (s *StateKvDB) RollbackDbTransaction(txName string) error {
	return errorSqldbOnly

}
func (s *StateKvDB) ExecDdlSql(contractName, sql string) error {
	return errorSqldbOnly

}

func (s *StateKvDB) operateDbByWriteSet(batch protocol.StoreBatcher, txWrite *commonPb.TxWrite) {
	// 5. state: contractID + stateKey
	txWriteKey := constructStateKey(txWrite.ContractName, txWrite.Key)
	if txWrite.Value == nil {
		batch.Delete(txWriteKey)
	} else {
		batch.Put(txWriteKey, txWrite.Value)
	}
}
func (s *StateKvDB) GetChainConfig() (*configPb.ChainConfig, error) {
	val, err := s.ReadObject(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		[]byte(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()))
	if err != nil {
		return nil, err
	}
	conf := &configPb.ChainConfig{}
	err = conf.Unmarshal(val)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
