/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"strconv"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/spf13/cobra"
)

var (
	txTimestampVerify bool
	txTimeout         int
	blockTxCapacity   int
	blockSize         int
	blockInterval     int
)

func ChainConfigBlockUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigBlockUpdate",
		Short: "Update chainConfig block params",
		Long:  "Update chainConfig block params, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,tx_timestamp_verify,tx_timeout,block_tx_capacity,block_size,block_interval)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return blockUpdate()
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&txTimestampVerify, "tx_timestamp_verify", false, "whether open the switch tx_timestamp_verify")
	flags.IntVar(&txTimeout, "tx_timeout", -100, "txTimeout (second)")
	flags.IntVar(&blockTxCapacity, "block_tx_capacity", -100, "the max block_tx_capacity")
	flags.IntVar(&blockSize, "block_size", -100, "the max block_size")
	flags.IntVar(&blockInterval, "block_interval", -100, "the max block_interval (ms)")

	return cmd
}

func blockUpdate() error {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "tx_timestamp_verify",
		Value: []byte(strconv.FormatBool(txTimestampVerify)),
	})
	if txTimeout > -100 {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "tx_timeout",
			Value: []byte(strconv.Itoa(txTimeout)),
		})
	}
	if blockTxCapacity > -100 {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "block_tx_capacity",
			Value: []byte(strconv.Itoa(blockTxCapacity)),
		})
	}
	if blockSize > -100 {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "block_size",
			Value: []byte(strconv.Itoa(blockSize)),
		})
	}
	if blockInterval > -100 {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "block_interval",
			Value: []byte(strconv.Itoa(blockInterval)),
		})
	}

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), pairs: pairs, oldSeq: seq})

	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	fmt.Println(result.ToJsonString())

	return nil
}
