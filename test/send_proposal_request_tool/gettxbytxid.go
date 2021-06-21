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
	pair := &commonPb.KeyValuePair{Key: "txId", Value: txId}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_TX_BY_TX_ID", pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	fmt.Printf("send tx resp: code:%d, msg:%s\n", resp.Code, resp.Message)

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
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
