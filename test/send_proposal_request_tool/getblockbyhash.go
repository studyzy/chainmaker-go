/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"encoding/json"
	"fmt"


	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func GetBlockByHashCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getBlockByHash",
		Short: "Get block by hash",
		Long:  "Get block by hash",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getBlockByHash()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&hash, "hash", "s", "", "specify block hash")
	flags.BoolVarP(&withRWSets, "with-rw-sets", "w", false, "specify whether return rw sets")

	return cmd
}

func getBlockByHash() error {
	// 构造Payload
	w := "false"
	if withRWSets {
		w = "true"
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: []byte(hash),
		},
		{
			Key:   "withRWSet",
			Value: []byte(w),
		},
	}

	payloadBytes, err := constructPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_BY_HASH", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes)
	if err != nil {
		return err
	}

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		return err
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		ContractResultCode:    resp.ContractResult.Code,
		ContractResultMessage: resp.ContractResult.Message,
		BlockInfo:             blockInfo,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
