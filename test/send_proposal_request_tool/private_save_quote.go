/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"
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
	flags.StringVarP(&enclaveId, "enclave_id", "eid", "", "enclave id ")
	flags.StringVarP(&quoteId, "quote_id", "qid", "", "quote id")
	flags.StringVarP(&quote, "quote", "q", "", "quote")
	flags.StringVarP(&sign, "sign", "s", "", "sign")

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
		common.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		common.PrivateComputeContractFunction_SAVE_QUOTE.String(),
		pairs,
		defaultSequence,
	)
	if err != nil {
		return fmt.Errorf("construct save quote  payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, common.TxType_INVOKE_SYSTEM_CONTRACT, chainId, "", payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_INVOKE_SYSTEM_CONTRACT.String(), err.Error())
	}

	if resp.Code == common.TxStatusCode_SUCCESS {
		if !withSyncResult {
			resp.ContractResult = &common.ContractResult{
				Code:    common.ContractResultCode_OK,
				Message: common.ContractResultCode_OK.String(),
				Result:  []byte(txId),
			}
		} else {
			contractResult, err := getSyncResult(txId)
			if err != nil {
				return fmt.Errorf("get sync result failed, %s", err.Error())
			}

			if contractResult.Code != common.ContractResultCode_OK {
				resp.Code = common.TxStatusCode_CONTRACT_FAIL
				resp.Message = contractResult.Message
			}

			resp.ContractResult = contractResult
		}
	}

	return nil
}
