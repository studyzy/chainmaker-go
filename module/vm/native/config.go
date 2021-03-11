/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
)

func IsNative(contractName string, txType commonPb.TxType) bool {
	return IsNativeContract(contractName) && IsNativeTxType(txType)
}

// IsNativeContract return is native contract name
func IsNativeContract(contractName string) bool {
	switch contractName {
	case commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(),
		commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(),
		commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(),
		commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String(),
		commonPb.ContractName_SYSTEM_CONTRACT_GOVERNANCE.String():
		return true
	default:
		return false
	}
}

// IsNativeTxType return is native contract supported transaction type
func IsNativeTxType(txType commonPb.TxType) bool {
	switch txType {
	case commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		commonPb.TxType_UPDATE_CHAIN_CONFIG:
		return true
	default:
		return false
	}
}
