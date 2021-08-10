/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import "regexp"

var (
	contractNameReg = regexp.MustCompile("^[a-zA-Z0-9_]{1,128}$")
	evmAddressReg   = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	chainIdReg      = regexp.MustCompile("^[a-zA-Z0-9_]{1,30}$")
	txIDReg         = regexp.MustCompile("^\\S{1,64}$")
)

func CheckChainIdFormat(chainId string) bool {
	return chainIdReg.MatchString(chainId)
}
func CheckContractNameFormat(name string) bool {
	return contractNameReg.MatchString(name)
}
func CheckEvmAddressFormat(addr string) bool {
	return evmAddressReg.MatchString(addr)
}
func CheckTxIDFormat(txID string) bool {
	return txIDReg.MatchString(txID)
}
