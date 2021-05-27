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
	"strings"
)

var (
	privateDirString string
	orderId          string
)

func SaveDirCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saveDir",
		Short: "save dir to blockchain",
		Long:  "save dir to blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return saveDir()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&orderId, "order_id", "g", "", "order id")
	flags.StringVarP(&privateDirString, "private_dir", "f", "", "private dir")
	flags.BoolVarP(&withSyncResult, "with_sync_result", "w", false, "with sync result")

	return cmd
}

func saveDir() error {

	privateDir := &common.StrSlice{
		StrArr: strings.Split(privateDirString, ","),
	}
	// 构造Payload
	priDirBytes, err := privateDir.Marshal()
	if err != nil {
		return fmt.Errorf("serielized private dir failed, %s", err.Error())
	}

	pairs := paramsMap2KVPairs(map[string]string{
		"order_id":    orderId,
		"private_dir": string(priDirBytes),
	})

	payloadBytes, err := constructSystemContractPayload(
		chainId,
		common.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		common.PrivateComputeContractFunction_SAVE_DIR.String(),
		pairs,
		defaultSequence,
	)
	if err != nil {
		return fmt.Errorf("construct save dir payload failed, %s", err.Error())
	}

	resp, err = proposalRequest(sk3, client, common.TxType_INVOKE_SYSTEM_CONTRACT, chainId, txId, payloadBytes)
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

	if resp.Code != common.TxStatusCode_SUCCESS || resp.Message != "OK" {
		return fmt.Errorf(errStringFormat, common.TxType_INVOKE_SYSTEM_CONTRACT.String(), err.Error())
	}

	resultStruct := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    string(resp.ContractResult.Result),
	}

	bytes, err := json.Marshal(resultStruct)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
