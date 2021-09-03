/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
)

//nolint
const (
	PrefixContractInfo     = "Contract:"
	PrefixContractByteCode = "ContractByteCode:"
)

func GetContractDbKey(contractName string) []byte {
	return []byte(PrefixContractInfo + contractName)
}
func GetContractByteCodeDbKey(contractName string) []byte {
	return []byte(PrefixContractByteCode + contractName)
}
func GetContractByName(readObject func(contractName string, key []byte) ([]byte, error), name string) (
	*commonPb.Contract, error) {
	key := GetContractDbKey(name)
	value, err := readObject(syscontract.SystemContract_CONTRACT_MANAGE.String(), key)
	if err != nil {
		return nil, err
	}
	contract := &commonPb.Contract{}
	err = contract.Unmarshal(value)
	if err != nil {
		return nil, err
	}
	return contract, nil
}
func GetContractBytecode(readObject func(contractName string, key []byte) ([]byte, error), name string) (
	[]byte, error) {
	key := GetContractByteCodeDbKey(name)
	return readObject(syscontract.SystemContract_CONTRACT_MANAGE.String(), key)
}
