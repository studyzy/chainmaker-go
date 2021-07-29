/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"strconv"

	"chainmaker.org/chainmaker-go/utils"

	"chainmaker.org/chainmaker/pb-go/syscontract"

	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func SubscribeBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribeBlock",
		Short: "Subscribe Block",
		Long:  "Subscribe Block",
		RunE: func(_ *cobra.Command, _ []string) error {
			return subscribeBlock()
		},
	}

	return cmd
}

func subscribeBlock() error {
	start, _ := utils.Int64ToBytes(startBlock)
	end, _ := utils.Int64ToBytes(endBlock)
	payload := &commonPb.Payload{
		Parameters: []*commonPb.KeyValuePair{
			{Key: syscontract.SubscribeBlock_START_BLOCK.String(), Value: start},
			{Key: syscontract.SubscribeBlock_END_BLOCK.String(), Value: end},
			{Key: syscontract.SubscribeBlock_WITH_RWSET.String(), Value: []byte(strconv.FormatBool(withRwSet))},
		},
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = subscribeRequest(sk3, client, syscontract.SubscribeFunction_SUBSCRIBE_BLOCK.String(), chainId, payloadBytes)
	if err != nil {
		return err
	}

	return nil
}
