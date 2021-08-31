/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	blockHeight int64
)

func ChainConfigGetChainConfigByBlockHeightCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getChainConfigByBlockHeight",
		Short: "getChainConfigByBlockHeight",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainConfigByBlockHeight()
		},
	}
	flags := cmd.Flags()
	flags.Int64Var(&blockHeight, "block_height", 0, "blockHeight")
	return cmd
}

func getChainConfigByBlockHeight() error {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "block_height",
		Value: []byte(strconv.Itoa(int(blockHeight))),
	})
	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_GET_CHAIN_CONFIG_AT.String(), pairs)
	if err != nil {
		return err
	}
	resp, err = proposalRequest(sk3, client, payloadBytes)
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
