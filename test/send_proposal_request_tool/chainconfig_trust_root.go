/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"errors"
	"fmt"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/spf13/cobra"
)

var (
	trustRootOrgId string
	trustRootCrt   string
)

const (
	trustRootOrgIdUsage = "the trustRoot org id"
	trustRootOrgIdStr   = "trust_root_org_id"
)

func ChainConfigTrustRootAddCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trustRootAdd",
		Short: "Add trustRoot",
		Long:  "Add trustRoot",
		RunE: func(_ *cobra.Command, _ []string) error {
			return trustRootAdd()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&trustRootOrgId, trustRootOrgIdStr, "", trustRootOrgIdUsage)
	flags.StringVar(&trustRootCrt, "trust_root_crt", "", "the trustRoot root crt")

	return cmd
}

func ChainConfigTrustRootUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trustRootUpdate",
		Short: "Update trustRoot",
		Long:  "Update trustRoot",
		RunE: func(_ *cobra.Command, _ []string) error {
			return trustRootUpdate()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&trustRootOrgId, trustRootOrgIdStr, "", trustRootOrgIdUsage)
	flags.StringVar(&trustRootCrt, "trust_root_crt", "", "the trustRoot root crt")

	return cmd
}

func ChainConfigTrustRootDeleteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trustRootDelete",
		Short: "Delete trustRoot",
		Long:  "Delete trustRoot",
		RunE: func(_ *cobra.Command, _ []string) error {
			return trustRootDelete()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&trustRootOrgId, trustRootOrgIdStr, "", trustRootOrgIdUsage)

	return cmd
}

func trustRootAdd() error {
	// 构造Payload
	if trustRootOrgId == "" || trustRootCrt == "" {
		return errors.New("the trustRoot orgId or crt is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(trustRootOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "root",
		Value: []byte(trustRootCrt),
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), pairs: pairs, oldSeq: seq})
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

func trustRootUpdate() error {
	// 构造Payload
	if trustRootOrgId == "" || trustRootCrt == "" {
		return errors.New("the trustRoot orgId or crt is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(trustRootOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "root",
		Value: []byte(trustRootCrt),
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), pairs: pairs, oldSeq: seq})
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

func trustRootDelete() error {
	// 构造Payload
	if trustRootOrgId == "" {
		return errors.New("the trustRoot orgId is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(trustRootOrgId),
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), pairs: pairs, oldSeq: seq})
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
