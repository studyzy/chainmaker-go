/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import commonPb "chainmaker.org/chainmaker/pb-go/common"

const (
	PrefixContractInfo     = "Contract:"
	PrefixContractByteCode = "ContractByteCode:"
)

func GetContractByName(readObject func(contractName string, key []byte) ([]byte, error), name string) (*commonPb.Contract, error) {
	key := []byte(PrefixContractInfo + name)
	value, err := readObject(commonPb.SystemContract_CONTRACT_MANAGE.String(), key)
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
func GetContractBytecode(readObject func(contractName string, key []byte) ([]byte, error), name string) ([]byte, error) {
	key := []byte(PrefixContractByteCode + name)
	return readObject(commonPb.SystemContract_CONTRACT_MANAGE.String(), key)
}
