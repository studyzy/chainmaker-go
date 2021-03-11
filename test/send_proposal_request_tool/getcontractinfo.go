/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func GetContractInfoCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getContractInfo",
		Short: "Get contract info",
		Long:  "Get contract info",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getContractInfo()
		},
	}

	return cmd
}

func getContractInfo() error {
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}

	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_CONTRACT_INFO", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)
	if err != nil {
		return err
	}

	contractInfo := &commonPb.ContractInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, contractInfo); err != nil {
		return err
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		ContractResultCode:    resp.ContractResult.Code,
		ContractResultMessage: resp.ContractResult.Message,
		ContractInfo:          contractInfo,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
