/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
)

// IsConfBlock is it a configuration block
func IsConfBlock(block *common.Block) bool {
	if block == nil || len(block.Txs) == 0 {
		return false
	}
	tx := block.Txs[0]
	return isValidConfigTx(tx)
}

// isValidConfigTx the transaction is a valid config transaction or not
func isValidConfigTx(tx *common.Transaction) bool {
	if tx.Result == nil || tx.Result.ContractResult == nil || tx.Result.ContractResult.Result == nil {
		return false
	}
	if !isConfigTx(tx) {
		return false
	}
	if tx.Result.Code != common.TxStatusCode_SUCCESS {
		return false
	}
	return true
}

// IsConfigTx the transaction is a config transaction or not
func isConfigTx(tx *common.Transaction) bool {
	if tx == nil {
		return false
	}
	return tx.Payload.ContractName == syscontract.SystemContract_CHAIN_CONFIG.String()
}
