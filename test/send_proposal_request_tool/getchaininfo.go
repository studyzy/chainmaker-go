/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	discoveryPb "chainmaker.org/chainmaker/pb-go/discovery"
	"encoding/json"
	"fmt"


	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func GetChainInfoCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getChainInfo",
		Short: "Get chain info",
		Long:  "Get chain info",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainInfo()
		},
	}

	return cmd
}

func getChainInfo() error {
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}

	payloadBytes, err := constructPayload(commonPb.SystemContract_CHAIN_QUERY.String(), "GET_CHAIN_INFO", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes)
	if err != nil {
		return err
	}

	chainInfo := &discoveryPb.ChainInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, chainInfo); err != nil {
		return err
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		ContractResultCode:    resp.ContractResult.Code,
		ContractResultMessage: resp.ContractResult.Message,
		ChainInfo:             chainInfo,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
