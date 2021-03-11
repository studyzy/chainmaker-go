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

	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func FreezeContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freezeContract",
		Short: "Freeze Contract",
		Long:  "Freeze Contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return freezeContract()
		},
	}

	return cmd
}

func freezeContract() error {
	txId := utils.GetRandTxId()

	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName: contractName,
		},
		Method:      "",
		Parameters:  nil,
		ByteCode:    nil,
		Endorsement: nil,
	}

	if endorsement, err := acSign(payload); err == nil {
		payload.Endorsement = endorsement
	} else {
		return err
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
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
