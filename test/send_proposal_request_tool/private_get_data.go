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
	flags.StringVarP(&contractName, "contract_name", "c", "", "contract name")
	flags.StringVarP(&key, "key", "k", "", "data key")

	return cmd
}

func getData() error {

	// 构造Payload
	pairs := paramsMap2KVPairs(map[string]string{
		"contract_name": contractName,
		"key":           key,
	})

	payloadBytes, err := constructQueryPayload(
		common.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String(),
		common.PrivateComputeContractFunction_GET_DATA.String(),
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

	return nil
}
