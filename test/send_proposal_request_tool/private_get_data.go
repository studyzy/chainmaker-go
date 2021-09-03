/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/spf13/cobra"
)

var (
	key string
)

func GetDataCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getData",
		Short: "get data from blockchain",
		Long:  "get data from blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getData()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&contractName, "contract_name", "x", "", "contract name")
	flags.StringVarP(&key, "key", "u", "", "data key")

	return cmd
}

func getData() error {

	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"contract_name": contractName,
		"key":           key,
	})

	payloadBytes, err := constructQueryPayload(chainId,
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		syscontract.PrivateComputeFunction_GET_DATA.String(),
		pairs,
	)
	if err != nil {
		return fmt.Errorf("marshal get data payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_CONTRACT.String(), err.Error())
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_CONTRACT.String(), err.Error())
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
