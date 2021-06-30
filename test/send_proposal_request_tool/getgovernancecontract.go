/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/consensus"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func ChainConfigGetGovernanceContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getGovernanceContract",
		Short: "getGovernanceContract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getGovernanceContract()
		},
	}

	return cmd
}

func getGovernanceContract() error {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_GOVERNANCE.String(), commonPb.QueryFunction_GET_GOVERNANCE_CONTRACT.String(), pairs)
	if err != nil {
		return err
	}
	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	mbftInfo := &consensusPb.GovernanceContract{}
	err = proto.Unmarshal(resp.ContractResult.Result, mbftInfo)
	if err != nil {
		return err
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		ContractResultCode:    resp.ContractResult.Code,
		ContractResultMessage: resp.ContractResult.Message,
		GovernanceInfo:        mbftInfo,
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
