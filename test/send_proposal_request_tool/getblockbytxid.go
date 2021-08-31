/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func GetBlockByTxIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getBlockByTxId",
		Short: "Get block by tx Id",
		Long:  "Get tx by tx Id",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getBlockByTxId()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&txId, "tx-id", "T", "", "specify tx id")
	flags.BoolVarP(&withRWSets, "with-rw-sets", "w", false, "specify whether return rw sets")

	return cmd
}

func getBlockByTxId() error {
	// 构造Payload
	w := "false"
	if withRWSets {
		w = "true"
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "txId",
			Value: []byte(txId),
		},
		{
			Key:   "withRWSet",
			Value: []byte(w),
		},
	}

	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_BY_TX_ID", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	log.DebugDynamic(func() string {
		data, _ := json.Marshal(resp)
		return string(data)
	})
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

	fmt.Println(result.ToJsonString())

	return nil
}
