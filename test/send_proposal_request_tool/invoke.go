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
	"io/ioutil"

	"chainmaker.org/chainmaker-go/utils"
	"github.com/spf13/cobra"
)

func InvokeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Invoke",
		Long:  "Invoke",
		RunE: func(_ *cobra.Command, _ []string) error {
			return invoke()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&pairsString, "pairs", "a", "[{\"key\":\"key\",\"value\":\"counter1\"}]", "specify pairs")
	flags.StringVarP(&pairsFile, "pairs-file", "A", "./pairs.json", "specify pairs file, if used, set --pairs=\"\"")
	flags.StringVarP(&method, "method", "m", "increase", "specify contract method")

	return cmd
}

func invoke() error {
	txId := utils.GetRandTxId()

	// 构造Payload
	if pairsString == "" {
		bytes, err := ioutil.ReadFile(pairsFile)
		if err != nil {
			panic(err)
		}
		pairsString = string(bytes)
	}
	var pairs []*commonPb.KeyValuePair
	err := json.Unmarshal([]byte(pairsString), &pairs)
	if err != nil {
		return err
	}

	payloadBytes, err := constructPayload(contractName, method, pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)
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
