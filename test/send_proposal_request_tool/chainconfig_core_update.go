/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	txSchedulerTimeout         int
	txSchedulerValidateTimeout int
)

func ChainConfigCoreUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigCoreUpdate",
		Short: "Update chainConfig core params",
		Long:  "Update chainConfig core params, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,tx_scheduler_timeout,tx_scheduler_validate_timeout)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return coreUpdate()
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&txSchedulerTimeout, "tx_scheduler_timeout", -100, "tx scheduler validate timeout")
	flags.IntVar(&txSchedulerValidateTimeout, "tx_scheduler_validate_timeout", -100, "tx scheduler validate timeout")

	return cmd
}

func coreUpdate() error {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	if txSchedulerTimeout > -100 {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "tx_scheduler_timeout",
			Value: strconv.Itoa(txSchedulerTimeout),
		})
	}
	if txSchedulerValidateTimeout > -100 {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   "tx_scheduler_validate_timeout",
			Value: strconv.Itoa(txSchedulerValidateTimeout),
		})
	}

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_UPDATE_CHAIN_CONFIG, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), method: commonPb.ConfigFunction_CORE_UPDATE.String(), pairs: pairs, oldSeq: seq})
	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
