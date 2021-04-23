/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"bytes"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/resultdb"
)

type HistoryIteratorImpl struct {
	contractName string
	key          []byte
	dbItr        []*historydb.BlockHeightTxId
	resultStore  resultdb.ResultDB
	blockStore   blockdb.BlockDB
	current      int
}

func NewHistoryIterator(contractName string, key []byte, dbItr []*historydb.BlockHeightTxId, resultStore resultdb.ResultDB, blockStore blockdb.BlockDB) *HistoryIteratorImpl {
	return &HistoryIteratorImpl{
		contractName: contractName,
		key:          key,
		dbItr:        dbItr,
		resultStore:  resultStore,
		blockStore:   blockStore,
		current:      0,
	}
}
func (hs *HistoryIteratorImpl) Next() bool {
	hs.current++
	return hs.current < len(hs.dbItr)
}

//func (hs *HistoryIteratorImpl)Key() []byte{
//	return []byte(hs.contractName+"_"+string(hs.key)+"_"+hs.dbItr[hs.current].TxId)
//}
//
func (hs *HistoryIteratorImpl) Value() (*storePb.KeyModification, error) {

	txId := hs.dbItr[hs.current].TxId
	result := storePb.KeyModification{
		TxId:     txId,
		IsDelete: false,
	}
	rwset, _ := hs.resultStore.GetTxRWSet(txId)
	for _, wset := range rwset.TxWrites {
		if bytes.Equal(wset.Key, hs.key) && wset.ContractName == hs.contractName {
			result.Value = wset.Value
		}
	}
	if len(result.Value) == 0 {
		result.IsDelete = true
	}
	tx, _ := hs.blockStore.GetTxWithBlockInfo(txId)
	result.Timestamp = tx.Transaction.Header.Timestamp
	return &result, nil
}
func (hs *HistoryIteratorImpl) Release() {

}
