/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func GetTxByTxIdCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getTxByTxId",
		Short: "Get tx by tx Id",
		Long:  "Get tx by tx Id",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getTxByTxId()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&txId, "tx-id", "T", "", "specify tx id")

	return cmd
}

func getTxByTxId() error {
	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: "txId", Value: []byte(txId)}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes, err := constructQueryPayload(chainId, syscontract.SystemContract_CHAIN_QUERY.String(), "GET_TX_BY_TX_ID", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	log.DebugDynamic(func() string {
		return fmt.Sprintf("send tx resp: code:%d, msg:%s", resp.Code, resp.Message)
	})

	transactionInfo := &commonPb.TransactionInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, transactionInfo); err != nil {
		return err
	}
	result := &Result{
		Code:                  resp.Code,
		Message:               resp.Message,
		ContractResultCode:    resp.ContractResult.Code,
		ContractResultMessage: resp.ContractResult.Message,
		TransactionInfo:       transactionInfo,
	}
	fmt.Println(result.ToJsonString())

	return nil
}
