/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/sha256"
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/spf13/cobra"
)

var (
	codeHash string
	//contractCode string
)

func GetContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getContract",
		Short: "get contract from blockchain",
		Long:  "get contract from blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getContract()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&contractName, "contract_name", "x", "", "contract name")
	flags.StringVarP(&contractCode, "", "r", "", "contract")

	return cmd
}

func getContract() error {

	codeHashArr := sha256.Sum256([]byte(contractCode))
	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"contract_name": contractName,
		"code_hash":     string(codeHashArr[:]),
	})

	payloadBytes, err := constructQueryPayload(chainId,
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		syscontract.PrivateComputeFunction_GET_CONTRACT.String(),
		pairs,
	)
	if err != nil {
		return fmt.Errorf("marshal get contract payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	contractInfo := &common.PrivateGetContract{}
	if err = contractInfo.Unmarshal(resp.ContractResult.Result); err != nil {
		return fmt.Errorf("GetContract unmarshal contract info payload failed, %s", err.Error())
	}

	resultStruct := &Result{
		Code:    resp.Code,
		Message: resp.Message,
	}

	if resp.ContractResult != nil {
		resultStruct.ContractResultCode = resp.ContractResult.Code
		resultStruct.ContractResultMessage = resp.ContractResult.Message
		resultStruct.ContractQueryResult = string(resp.ContractResult.Result)
	} else {
		fmt.Println("resp.ContractResult is nil ")
	}

	fmt.Println(resultStruct.ToJsonString())

	return nil
}
