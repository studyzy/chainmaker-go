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
	result string
	rwSet  string
	events string
)

func SaveDataCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saveData",
		Short: "save data to blockchain",
		Long:  "save data to blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return saveData()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&result, "result", "r", "", "result")
	flags.StringVarP(&contractName, "contract_name", "x", "", "contract name")
	flags.StringVarP(&rwSet, "rw_set", "s", "", "read write set")
	flags.StringVarP(&events, "events", "q", "", "events")
	flags.BoolVarP(&withSyncResult, "with_sync_result", "w", false, "with sync result")

	return cmd
}

func saveData() error {
	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"result":        result,
		"contract_name": contractName,
		"rw_set":        rwSet,
		"events":        events,
	})

	payloadBytes, err := constructSystemContractPayload(
		chainId,
		syscontract.SystemContract_PRIVATE_COMPUTE.String(),
		syscontract.PrivateComputeFunction_SAVE_DATA.String(),
		pairs,
		defaultSequence,
	)

	if err != nil {
		return fmt.Errorf("construct save data payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_INVOKE_CONTRACT.String(), err.Error())
	}

	if resp.Code == common.TxStatusCode_SUCCESS {
		if !withSyncResult {
			resp.ContractResult = &common.ContractResult{
				Code:    0,
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

	if resp.Code != common.TxStatusCode_SUCCESS || resp.Message != "OK" {
		return fmt.Errorf(errStringFormat, common.TxType_INVOKE_CONTRACT.String(), err.Error())
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
