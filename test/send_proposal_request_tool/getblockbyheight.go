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

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func GetBlockByHeightCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getBlockByHeight",
		Short: "Get block by height",
		Long:  "Get block by height",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getBlockByHeight()
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&height, "height", "H", -1, "specify block height")
	flags.BoolVarP(&withRWSets, "with-rw-sets", "w", false, "specify whether return rw sets")

	return cmd
}

func getBlockByHeight() error {
	// 构造Payload
	w := "false"
	if withRWSets {
		w = "true"
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: []byte(strconv.Itoa(height)),
		},
		{
			Key:   "withRWSet",
			Value: []byte(w),
		},
	}

	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_BY_HEIGHT", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
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
