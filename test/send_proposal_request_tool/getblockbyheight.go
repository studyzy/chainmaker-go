/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"math"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

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
	flags.Uint64VarP(&height, "height", "H", math.MaxUint64, "specify block height")
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
			Value: []byte(fmt.Sprintf("%d", height)),
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
	fmt.Println(result.ToJsonString())

	return nil
}
