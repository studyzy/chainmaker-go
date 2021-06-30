/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func ChainConfigGetChainConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "getChainConfig",
		Long: "getChainConfig",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainConfig()
		},
	}

	return cmd
}

func getChainConfig() error {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), commonPb.ConfigFunction_GET_CHAIN_CONFIG.String(), pairs)
	if err != nil {
		return err
	}
	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	chainConfig := &configPb.ChainConfig{}
	err = proto.Unmarshal(resp.ContractResult.Result, chainConfig)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(chainConfig)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
