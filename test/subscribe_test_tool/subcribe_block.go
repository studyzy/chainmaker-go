/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
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
	payload := &commonPb.SubscribeBlockPayload{
		StartBlock: startBlock,
		EndBlock:   endBlock,
		//WithRwSet:  true,
		WithRwSet: withRwSet,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = subscribeRequest(sk3, client, commonPb.TxType_SUBSCRIBE_BLOCK_INFO, chainId, payloadBytes)
	if err != nil {
		return err
	}

	return nil
}
