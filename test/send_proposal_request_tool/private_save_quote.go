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
	quoteId string
	quote   string
	sign    string
)

func SaveQuoteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saveQuote",
		Short: "save quote to  blockchain",
		Long:  "save quote to  blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return saveQuote()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&enclaveId, "enclave_id", "l", "", "enclave id ")
	flags.StringVarP(&quoteId, "quote_id", "u", "", "quote id")
	flags.StringVarP(&quote, "quote", "a", "", "quote")
	flags.StringVarP(&sign, "sign", "b", "", "sign")
	flags.BoolVarP(&withSyncResult, "with_sync_result", "w", false, "with sync result")

	return cmd
}

func saveQuote() error {
	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"enclave_id": enclaveId,
		"quote_id":   quoteId,
		"quote":      quote,
		"sign":       sign,
	})

	payloadBytes, err := constructSystemContractPayload(
		chainId,
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		"SAVE_QUOTE", // syscontract.PrivateComputeFunction_SAVE_QUOTE.String(),
		pairs,
		defaultSequence,
	)
	if err != nil {
		return fmt.Errorf("construct save quote  payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	if resp.Code == common.TxStatusCode_SUCCESS {
		if !withSyncResult {
			resp.ContractResult = &common.ContractResult{
				Code:    uint32(0),
				Message: "OK",
				Result:  []byte(txId),
			}
		} else {
			contractResult, err := getSyncResult(txId)
			if err != nil {
				return fmt.Errorf("get sync result failed, %s", err.Error())
			}

			if contractResult.Code != 0 {
				resp.Code = common.TxStatusCode_CONTRACT_FAIL
				resp.Message = contractResult.Message
			}

			resp.ContractResult = contractResult
		}
	}

	resultStruct := &Result{
		Code:    resp.Code,
		Message: resp.Message,
	}

	if resp.ContractResult != nil {
		resultStruct.TxId = string(resp.ContractResult.Result)
		resultStruct.ContractResultCode = resp.ContractResult.Code
		resultStruct.ContractResultMessage = resp.ContractResult.Message
		resultStruct.ContractQueryResult = string(resp.ContractResult.Result)
	} else {
		fmt.Println("resp.ContractResult is nil ")
	}
	fmt.Println(resultStruct.ToJsonString())

	return nil
}
