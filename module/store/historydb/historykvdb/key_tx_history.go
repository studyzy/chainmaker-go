/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package historykvdb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chainmaker.org/chainmaker-go/store/historydb"
)

//k+ContractName+StateKey+BlockHeight+TxId
func constructKey(contractName string, key []byte, blockHeight uint64, txId string) []byte {
	dbkey := fmt.Sprintf(keyHistoryPrefix+"%s"+splitChar+"%s"+splitChar+"%d"+splitChar+"%s",
		contractName, key, blockHeight, txId)
	return []byte(dbkey)
}
func constructKeyPrefix(contractName string, key []byte) []byte {
	dbkey := fmt.Sprintf(keyHistoryPrefix+"%s"+splitChar+"%s"+splitChar, contractName, key)
	return []byte(dbkey)
}
func splitKey(dbKey []byte) (contractName string, key []byte, blockHeight uint64, txId string, err error) {
	if len(dbKey) == 0 {
		return "", nil, 0, "", errors.New("empty dbKey")
	}
	array := strings.Split(string(dbKey[1:]), splitChar)
	if len(array) != 4 {
		return "", nil, 0, "", errors.New("invalid dbKey format")
	}
	contractName = array[0]
	key = []byte(array[1])
	var height int
	height, err = strconv.Atoi(array[2])
	blockHeight = uint64(height)
	txId = array[3]
	return
}
func (h *HistoryKvDB) GetHistoryForKey(contractName string, key []byte) (historydb.HistoryIterator, error) {
	iter, erro := h.dbHandle.NewIteratorWithPrefix(constructKeyPrefix(contractName, key))
	if erro != nil {
		return nil, erro
	}
	splitKeyFunc := func(key []byte) (*historydb.BlockHeightTxId, error) {
		_, _, height, txId, err := splitKey(key)
		if err != nil {
			return nil, err
		}
		return &historydb.BlockHeightTxId{
			BlockHeight: height,
			TxId:        txId,
		}, nil
	}
	return &historyKeyIterator{dbIter: iter, buildFunc: splitKeyFunc}, nil
}
