/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resultkvdb

import (
	"encoding/binary"
	"fmt"

	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

const (
	txRWSetIdxKeyPrefix  = 'r'
	resultDBSavepointKey = "resultSavepointKey"
)

// ResultKvDB provider a implementation of `historydb.HistoryDB`
// This implementation provides a key-value based data model
type ResultKvDB struct {
	dbHandle protocol.DBHandle
	cache    *cache.StoreCacheMgr
	logger   protocol.Logger
}

func NewResultKvDB(chainId string, handle protocol.DBHandle, logger protocol.Logger) *ResultKvDB {
	return &ResultKvDB{
		dbHandle: handle,
		cache:    cache.NewStoreCacheMgr(chainId, 10, logger),
		logger:   logger,
	}
}

func (h *ResultKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	return h.CommitBlock(genesisBlock)
}

// CommitBlock commits the block rwsets in an atomic operation
func (h *ResultKvDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	batch := types.NewUpdateBatch()
	// 1. last block height
	block := blockInfo.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, uint64(block.Header.BlockHeight))
	batch.Put([]byte(resultDBSavepointKey), lastBlockNumBytes)

	txRWSets := blockInfo.TxRWSets
	rwsetData := blockInfo.SerializedTxRWSets
	for index, txRWSet := range txRWSets {
		// 6. rwset: txID -> txRWSet
		txRWSetBytes := rwsetData[index]
		txRWSetKey := constructTxRWSetIDKey(txRWSet.TxId)
		batch.Put(txRWSetKey, txRWSetBytes)
	}
	err := h.writeBatch(block.Header.BlockHeight, batch)
	if err != nil {
		return err
	}
	h.logger.Debugf("chain[%s]: commit history block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil
}

// ShrinkBlocks archive old blocks rwsets in an atomic operation
func (h *ResultKvDB) ShrinkBlocks(txIdsMap map[uint64][]string) error {
	var err error

	for _, txIds := range txIdsMap {
		batch := types.NewUpdateBatch()
		for _, txId := range txIds {
			txRWSetKey := constructTxRWSetIDKey(txId)
			batch.Delete(txRWSetKey)
		}
		if err = h.dbHandle.WriteBatch(batch, false); err != nil {
			return err
		}
	}

	go h.compactRange()

	return nil
}

func (h *ResultKvDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	startTime := utils.CurrentTimeMillisSeconds()
	for i := len(blockInfos) - 1; i >= 0; i-- {
		blockInfo := blockInfos[i]

		//check whether block can be archived
		if utils.IsConfBlock(blockInfo.Block) {
			h.logger.Infof("skip store conf block: [%d]", blockInfo.Block.Header.BlockHeight)
			continue
		}

		txRWSets := blockInfo.TxRWSets
		rwsetData := blockInfo.SerializedTxRWSets
		batch := types.NewUpdateBatch()
		for index, txRWSet := range txRWSets {
			// rwset: txID -> txRWSet
			batch.Put(constructTxRWSetIDKey(txRWSet.TxId), rwsetData[index])
		}
		if err := h.dbHandle.WriteBatch(batch, false); err != nil {
			return err
		}
	}

	beforeWrite := utils.CurrentTimeMillisSeconds()

	go h.compactRange()

	writeTime := utils.CurrentTimeMillisSeconds() - beforeWrite
	h.logger.Infof("restore block RWSets from [%d] to [%d] time used (prepare_txs:%d write_batch:%d, total:%d)",
		blockInfos[len(blockInfos)-1].Block.Header.BlockHeight, blockInfos[0].Block.Header.BlockHeight,
		beforeWrite-startTime, writeTime, utils.CurrentTimeMillisSeconds()-startTime)

	return nil
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
func (h *ResultKvDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	txRWSetKey := constructTxRWSetIDKey(txId)
	bytes, err := h.get(txRWSetKey)
	if err != nil {
		return nil, err
	} else if bytes == nil {
		return nil, nil
	}

	var txRWSet commonPb.TxRWSet
	err = proto.Unmarshal(bytes, &txRWSet)
	if err != nil {
		return nil, err
	}
	return &txRWSet, nil
}

// GetLastSavepoint returns the last block height
func (h *ResultKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := h.get([]byte(resultDBSavepointKey))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}
	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// Close is used to close database
func (h *ResultKvDB) Close() {
	h.logger.Info("close result kv db")
	h.dbHandle.Close()
	h.cache.Clear()
}

func (h *ResultKvDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {
	//update cache
	h.cache.AddBlock(blockHeight, batch)
	go func() {
		err := h.dbHandle.WriteBatch(batch, false)
		if err != nil {
			panic(fmt.Sprintf("Error writing db: %s", err))
		}
		//db committed, clean cache
		h.cache.DelBlock(blockHeight)
	}()
	return nil
}

func (h *ResultKvDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := h.cache.Get(string(key))
	if exist {
		return value, nil
	}
	//get from database
	return h.dbHandle.Get(key)
}

//
//func (h *ResultKvDB) has(key []byte) (bool, error) {
//	//check has from cache
//	isDelete, exist := h.cache.Has(string(key))
//	if exist {
//		return !isDelete, nil
//	}
//	return h.dbHandle.Has(key)
//}

func constructTxRWSetIDKey(txId string) []byte {
	return append([]byte{txRWSetIdxKeyPrefix}, txId...)
}

func (h *ResultKvDB) compactRange() {
	//trigger level compact
	for i := 1; i <= 1; i++ {
		h.logger.Infof("Do %dst time CompactRange", i)
		if err := h.dbHandle.CompactRange(nil, nil); err != nil {
			h.logger.Warnf("resultdb level compact failed: %v", err)
		}
		//time.Sleep(2 * time.Second)
	}
}
