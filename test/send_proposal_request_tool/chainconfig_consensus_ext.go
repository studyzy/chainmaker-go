/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"errors"
	"fmt"
	"strings"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/spf13/cobra"
)

var (
	consensusExtKeys   string
	consensusExtValues string
)

var (
	errStr = "the consensusExt keys or consensusExt values is empty"

	consensusExtKeysStr   = "consensusExtKeys"
	consensusExtValuesStr = "consensusExtValues"
	consensusextkeys      = "consensus_ext_keys"
	consensusextvalues    = "consensus_ext_values"
)

func ChainConfigConsensusExtAddCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigConsensusExtAdd",
		Short: "Add ConsensusExt",
		Long:  "Add ConsensusExt, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,consensus_ext_keys,consensus_ext_values)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return consensusExtAdd()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&consensusExtKeys, consensusextkeys, "", consensusExtKeysStr)
	flags.StringVar(&consensusExtValues, consensusextvalues, "", consensusExtValuesStr)

	return cmd
}

func ChainConfigConsensusExtUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigConsensusExtUpdate",
		Short: "Update consensusExt",
		Long:  "Update consensusExt, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,consensus_ext_keys,consensus_ext_values)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return consensusExtUpdate()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&consensusExtKeys, consensusextkeys, "", consensusExtKeysStr)
	flags.StringVar(&consensusExtValues, consensusextvalues, "", consensusExtValuesStr)

	return cmd
}

func ChainConfigConsensusExtDeleteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigConsensusExtDelete",
		Short: "Delete consensusExt",
		Long:  "Delete consensusExt, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,consensus_ext_keys)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return consensusExtDelete()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&consensusExtKeys, consensusextkeys, "", consensusExtKeysStr)

	return cmd
}

func consensusExtAdd() error {
	// 构造Payload
	if consensusExtKeys == "" || consensusExtValues == "" {
		return errors.New(errStr)
	}
	consensusExtKeyArray := strings.Split(consensusExtKeys, ",")
	consensusExtValueArray := strings.Split(consensusExtValues, ",")
	if len(consensusExtKeyArray) != len(consensusExtValueArray) {
		return errors.New("the consensusExt keys len is not equal to values len")
	}

	pairs := make([]*commonPb.KeyValuePair, 0)

	for i, key := range consensusExtKeyArray {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   key,
			Value: []byte(consensusExtValueArray[i]),
		})
	}

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), pairs: pairs, oldSeq: seq})
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

func consensusExtUpdate() error {
	// 构造Payload
	if consensusExtKeys == "" || consensusExtValues == "" {
		return errors.New(errStr)
	}
	consensusExtKeyArray := strings.Split(consensusExtKeys, ",")
	consensusExtValueArray := strings.Split(consensusExtValues, ",")
	if len(consensusExtKeyArray) != len(consensusExtValueArray) {
		return errors.New("the consensusExt keys len is not equal to values len")
	}

	pairs := make([]*commonPb.KeyValuePair, 0)

	for i, key := range consensusExtKeyArray {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   key,
			Value: []byte(consensusExtValueArray[i]),
		})
	}

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), pairs: pairs, oldSeq: seq})
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

func consensusExtDelete() error {
	// 构造Payload
	if consensusExtKeys == "" {
		return errors.New(errStr)
	}
	consensusExtKeyArray := strings.Split(consensusExtKeys, ",")
	pairs := make([]*commonPb.KeyValuePair, 0)

	for _, key := range consensusExtKeyArray {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key: key,
		})
	}

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), pairs: pairs, oldSeq: seq})
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
