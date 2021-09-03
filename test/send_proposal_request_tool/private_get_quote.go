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

func GetQuoteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getQuote",
		Short: "get quote from blockchain",
		Long:  "get quote from blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getQuote()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&quoteId, "quote_id", "v", "", "quote id")

	return cmd
}

func getQuote() error {
	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"quote_id": quoteId,
	})

	payloadBytes, err := constructQueryPayload(chainId,
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		"GET_QUOTE", // syscontract.PrivateComputeFunction_GET_QUOTE.String(),
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
