/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gogo/protobuf/proto"
)

func SubscribeTxCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribeTx",
		Short: "Subscribe Tx",
		Long:  "Subscribe Tx",
		RunE: func(_ *cobra.Command, _ []string) error {
			return subscribeTx()
		},
	}
	return cmd
}

func subscribeTx() error {
	var ids []string
	if len(txIds) > 0 {
		ids = strings.Split(txIds, ",")
	}
	payload := &commonPb.SubscribeTxPayload{
		StartBlock: startBlock,
		EndBlock:   endBlock,
		TxType:     commonPb.TxType(txType),
		TxIds:      ids,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	_, err = subscribeRequest(sk3, client, commonPb.TxType_SUBSCRIBE_TX_INFO, chainId, payloadBytes)
	if err != nil {
		return err
	}

	return nil
}
