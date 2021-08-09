/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package historykvdb

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"chainmaker.org/chainmaker-go/store/historydb"
)

func constructAcctTxHistKey(accountId []byte, blockHeight uint64, txId string) []byte {
	key := fmt.Sprintf(accountTxHistoryPrefix+"%x_%d_%s", accountId, blockHeight, txId)
	return []byte(key)
}
func constructAcctTxHistKeyPrefix(accountId []byte) []byte {
	key := fmt.Sprintf(accountTxHistoryPrefix+"%x_", accountId)
	return []byte(key)
}
func splitAcctTxHistKey(key []byte) (accountId []byte, blockHeight uint64, txId string, err error) {
	if len(key) == 0 {
		err = errors.New("empty dbKey")
		return
	}
	array := strings.Split(string(key[1:]), "_")
	if len(array) != 3 {
		err = errors.New("invalid dbKey format")
		return
	}
	accountId, err = hex.DecodeString(array[0])
	if err != nil {
		return
	}
	var height int
	height, err = strconv.Atoi(array[1])
	blockHeight = uint64(height)
	txId = array[2]
	return
}

//GetAccountTxHistory AccountId+BlockHeight+ TxId
func (h *HistoryKvDB) GetAccountTxHistory(account []byte) (historydb.HistoryIterator, error) {
	iter, err := h.dbHandle.NewIteratorWithPrefix(constructAcctTxHistKeyPrefix(account))
	if err != nil {
		return nil, err
	}
	splitKeyFunc := func(key []byte) (*historydb.BlockHeightTxId, error) {
		_, height, txId, err := splitAcctTxHistKey(key)
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
