/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dpos

import (
	"bytes"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

func (impl *DPoSImpl) getState(
	contractName string, key []byte, block *common.Block, blockTxRwSet map[string]*common.TxRWSet) ([]byte, error) {
	if len(block.Txs) > 0 {
		for i := len(block.Txs) - 1; i >= 0; i-- {
			rwSets := blockTxRwSet[block.Txs[i].Payload.TxId]
			for _, txWrite := range rwSets.TxWrites {
				if txWrite.ContractName == contractName && bytes.Equal(txWrite.Key, key) {
					return txWrite.Value, nil
				}
			}
		}
	}

	val, err := impl.stateDB.ReadObject(contractName, key)
	if err != nil {
		impl.log.Errorf("query user balance failed, reason: %s", err)
		return nil, err
	}
	return val, nil
}
