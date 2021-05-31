/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"bytes"

	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/resultdb"
	storePb "chainmaker.org/chainmaker/pb-go/store"
)

type HistoryIteratorImpl struct {
	contractName string
	key          []byte
	dbItr        historydb.HistoryIterator
	resultStore  resultdb.ResultDB
	blockStore   blockdb.BlockDB
}

func NewHistoryIterator(contractName string, key []byte, dbItr historydb.HistoryIterator,
	resultStore resultdb.ResultDB, blockStore blockdb.BlockDB) *HistoryIteratorImpl {
	return &HistoryIteratorImpl{
		contractName: contractName,
		key:          key,
		dbItr:        dbItr,
		resultStore:  resultStore,
		blockStore:   blockStore,
	}
}
func (hs *HistoryIteratorImpl) Next() bool {
	return hs.dbItr.Next()
}

func (hs *HistoryIteratorImpl) Value() (*storePb.KeyModification, error) {

	txId, _ := hs.dbItr.Value()
	result := storePb.KeyModification{
		TxId:     txId.TxId,
		IsDelete: false,
	}
	rwset, _ := hs.resultStore.GetTxRWSet(txId.TxId)
	for _, wset := range rwset.TxWrites {
		if bytes.Equal(wset.Key, hs.key) && wset.ContractName == hs.contractName {
			result.Value = wset.Value
		}
	}
	if len(result.Value) == 0 {
		result.IsDelete = true
	}
	tx, _ := hs.blockStore.GetTxWithBlockInfo(txId.TxId)
	result.Timestamp = tx.Transaction.Header.Timestamp
	return &result, nil
}
func (hs *HistoryIteratorImpl) Release() {
	hs.dbItr.Release()
}

type TxHistoryIteratorImpl struct {
	dbItr      historydb.HistoryIterator
	blockStore blockdb.BlockDB
}

func NewTxHistoryIterator(dbItr historydb.HistoryIterator, blockStore blockdb.BlockDB) *TxHistoryIteratorImpl {
	return &TxHistoryIteratorImpl{
		dbItr:      dbItr,
		blockStore: blockStore,
	}
}
func (hs *TxHistoryIteratorImpl) Next() bool {
	return hs.dbItr.Next()
}

func (hs *TxHistoryIteratorImpl) Value() (*storePb.TxHistory, error) {
	txId, _ := hs.dbItr.Value()
	result := storePb.TxHistory{
		TxId:        txId.TxId,
		BlockHeight: txId.BlockHeight,
	}
	tx, _ := hs.blockStore.GetTxWithBlockInfo(txId.TxId)
	result.Timestamp = tx.Transaction.Header.Timestamp
	result.BlockHash = tx.BlockHash
	return &result, nil
}
func (hs *TxHistoryIteratorImpl) Release() {
	hs.dbItr.Release()
}
