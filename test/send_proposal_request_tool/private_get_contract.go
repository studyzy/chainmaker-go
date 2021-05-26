/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	codeHash string
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
	flags.StringVarP(&contractName, "contract_name", "c", "", "contract name")
	flags.StringVarP(&codeHash, "code_hash", "p", "", "code hash")

	return cmd
}

func getContract() error {
	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"contract_name": contractName,
		"code_hash":     codeHash,
	})

	payloadBytes, err := constructQueryPayload(
		common.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		common.PrivateComputeContractFunction_GET_CONTRACT.String(),
		pairs,
	)
	if err != nil {
		return fmt.Errorf("marshal get contract payload failed, %s", err.Error())
	}

	resp, err := proposalRequest(sk3, client, common.TxType_QUERY_SYSTEM_CONTRACT, chainId, "", payloadBytes)
	if err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_SYSTEM_CONTRACT.String(), err.Error())
	}

	if err = checkProposalRequestResp(resp, true); err != nil {
		return fmt.Errorf(errStringFormat, common.TxType_QUERY_SYSTEM_CONTRACT.String(), err.Error())
	}

	contractInfo := &common.PrivateGetContract{}
	if err = proto.Unmarshal(resp.ContractResult.Result, contractInfo); err != nil {
		return fmt.Errorf("GetContract unmarshal contract info payload failed, %s", err.Error())
	}

	return nil
}
