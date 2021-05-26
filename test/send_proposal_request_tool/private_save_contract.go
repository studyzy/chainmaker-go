/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"crypto/sha256"
	"fmt"
	"github.com/spf13/cobra"
)

var (
	contractCode string
)

func SaveContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "saveContract",
		Short: "save contract to blockchain",
		Long:  "save contract to blockchain",
		RunE: func(_ *cobra.Command, _ []string) error {
			return saveContract()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&contractName, "contract_name", "x", "", "contract name")
	flags.StringVarP(&contractCode, "contract_code", "r", "", "contract code")
	flags.StringVarP(&version, "version", "v", "", "version")
	flags.BoolVarP(&withSyncResult, "with_sync_result", "w", false, "with sync result")

	return cmd
}

func saveContract() error {

	contractCodeHash := sha256.Sum256([]byte(contractCode))
	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"contract_code": contractCode,
		"code_hash":     string(contractCodeHash[:]),
		"contract_name": contractName,
		"version":       version,
	})

	payloadBytes, err := constructSystemContractPayload(
		chainId,
		common.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		common.PrivateComputeContractFunction_SAVE_CONTRACT.String(),
		pairs,
		defaultSequence,
	)
	if err != nil {
		return fmt.Errorf("construct save contract code payload failed, %s", err.Error())
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
