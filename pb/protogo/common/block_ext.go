/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

func (b *Block) Hash() []byte {
	return b.Header.BlockHash
}
func (b *Block) GetBlockHashStr() string {
	return hex.EncodeToString(b.Header.BlockHash)
}
func (b *Block) IsContractMgmtBlock() bool {
	return b.Txs[0].Header.TxType == TxType_MANAGE_USER_CONTRACT
}
func (b *Block) GetTimestamp() time.Time {
	return time.Unix(b.Header.BlockTimestamp, 0)
}

// GetTxKey get transaction key
func (b *Block) GetTxKey() string {
	return GetTxKewWith(b.Header.Proposer, b.Header.BlockHeight)
}

func (b *Block) IsConfigBlock() bool {
	return b.Header.BlockHeight == 0 || b.Txs[0].Header.TxType == TxType_UPDATE_CHAIN_CONFIG
}
func GetTxKewWith(propose []byte, blockHeight int64) string {
	if propose == nil {
		return ""
	}
	f := sha256.New()
	f.Write(propose)
	f.Write([]byte(strconv.Itoa(int(blockHeight))))
	return hex.EncodeToString(f.Sum(nil))
}
