/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	evm "chainmaker.org/chainmaker/common/v2/evmutils"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/spf13/cobra"
)

func ContractNameToAddressCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contractNameToAddress",
		Short: "contractNameToAddress",
		Long:  "contractNameToAddress",
		RunE: func(_ *cobra.Command, _ []string) error {
			return contractNameToAddress()
		},
	}

	return cmd
}

func contractNameToAddress() error {

	var nulAddr *evm.Address

	outAddr, err := getContractAddress(contractName)
	if err != nil {
		output := fmt.Sprintf("getContractAddress failure!: %s, err: %s", contractName, err.Error())
		err = certReturnResult(commonPb.TxStatusCode_INTERNAL_ERROR, output, nulAddr)
		return err
	}
	var outAddrByte []byte
	outAddr.SetBytes(outAddrByte)

	fmt.Println("outAddrByte: ", outAddrByte, ", outAddr: ", outAddr)

	return certReturnResult(commonPb.TxStatusCode_SUCCESS, "SUCCESS", outAddr)
}
