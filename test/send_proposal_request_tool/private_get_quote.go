/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/json"
	"fmt"
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

	payloadBytes, err := constructQueryPayload(
		common.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		common.PrivateComputeContractFunction_GET_QUOTE.String(),
		pairs,
	)
	if err != nil {
		return fmt.Errorf("marshal get data payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, common.TxType_QUERY_SYSTEM_CONTRACT, chainId, "", payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_SYSTEM_CONTRACT.String(), err.Error())
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_SYSTEM_CONTRACT.String(), err.Error())
	}

	resultStruct := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
		ContractQueryResult: string(resp.ContractResult.Result),
	}

	bytes, err := json.Marshal(resultStruct)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
