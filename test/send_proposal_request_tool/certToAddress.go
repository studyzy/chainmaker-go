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

func CertToAddressCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certToAddress",
		Short: "certToAddress",
		Long:  "certToAddress",
		RunE: func(_ *cobra.Command, _ []string) error {
			return certToAddress()
		},
	}

	return cmd
}

func certReturnResult(code commonPb.TxStatusCode, message string, addr *evm.Address) error {
	var result *Result
	result = &Result{
		Code:                  code,
		Message:               message,
		ContractResultCode:    0,
		ContractResultMessage: "",
		ContractQueryResult:   "",
		CertAddress:           addr,
	}

	fmt.Println("addr: ", addr, "result: ", result)

	fmt.Println(result.ToJsonString())
	return nil
}

func certToAddress() error {

	var nulAddr *evm.Address

	ski, err := getSki(userCrtPath)
	fmt.Println("userCrtPath: ", userCrtPath)
	if err != nil {
		output := fmt.Sprintf("getSki failure!: %s, err: %s", userCrtPath, err.Error())
		err = certReturnResult(commonPb.TxStatusCode_INTERNAL_ERROR, output, nulAddr)
		return err
	}

	outAddr, err := getAddr(ski)
	if err != nil {
		output := fmt.Sprintf("getAddr failure!: %s, err: %s", ski, err.Error())
		err = certReturnResult(commonPb.TxStatusCode_INTERNAL_ERROR, output, nulAddr)
		return err
	}
	var outAddrByte []byte
	outAddr.SetBytes(outAddrByte)

	fmt.Println("outAddrByte: ", outAddrByte, ", outAddr: ", outAddr)

	return certReturnResult(commonPb.TxStatusCode_SUCCESS, "SUCCESS", outAddr)
}
