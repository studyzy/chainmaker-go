/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historykvdb

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"encoding/binary"
	"fmt"
	"github.com/gogo/protobuf/proto"
)

const (
	historyDBName         = ""
	txRWSetIdxKeyPrefix   = 'r'
	historyDBSavepointKey = "historySavepointKey"
)

// HistoryKvDB provider a implementation of `historydb.HistoryDB`
// This implementation provides a key-value based data model
type HistoryKvDB struct {
	DbHandle protocol.DBHandle
	Cache    *cache.StoreCacheMgr

	Logger protocol.Logger
}

// CommitBlock commits the block rwsets in an atomic operation
func (h *HistoryKvDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	batch := types.NewUpdateBatch()
	// 1. last block height
	block := blockInfo.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, uint64(block.Header.BlockHeight))
	batch.Put([]byte(historyDBSavepointKey), lastBlockNumBytes)

	txRWSets := blockInfo.TxRWSets
	for index, txRWSet := range txRWSets {
		// 6. rwset: txID -> txRWSet
		txRWSetBytes := blockInfo.SerializedTxRWSets[index]
		txRWSetKey := constructTxRWSetIDKey(txRWSet.TxId)
		batch.Put(txRWSetKey, txRWSetBytes)
	}
	err := h.writeBatch(block.Header.BlockHeight, batch)
	if err != nil {
		return err
	}
	h.Logger.Debugf("chain[%s]: commit history block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
func (h *HistoryKvDB) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
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
func (h *HistoryKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := h.get([]byte(historyDBSavepointKey))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}
	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// Close is used to close database
func (h *HistoryKvDB) Close() {
	h.DbHandle.Close()
}

func (h *HistoryKvDB) writeBatch(blockHeight int64, batch protocol.StoreBatcher) error {
	//update cache
	h.Cache.AddBlock(blockHeight, batch)
	go func() {
		err := h.DbHandle.WriteBatch(batch, false)
		if err != nil {
			panic(fmt.Sprintf("Error writting db: %s", err))
		}
		//db committed, clean cache
		h.Cache.DelBlock(blockHeight)
	}()
	return nil
}

func (h *HistoryKvDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := h.Cache.Get(string(key))
	if exist {
		return value, nil
	}
	//get from database
	return h.DbHandle.Get(key)
}

func (h *HistoryKvDB) has(key []byte) (bool, error) {
	//check has from cache
	isDelete, exist := h.Cache.Has(string(key))
	if exist {
		return !isDelete, nil
	}
	return h.DbHandle.Has(key)
}

func constructTxRWSetIDKey(txId string) []byte {
	return append([]byte{txRWSetIdxKeyPrefix}, txId...)
}
