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
	nodeOrgOrgId     string
	nodeOrgAddresses string
)

const (
	nodeOrgId       = "the nodeOrg org id"
	nodeOrgOrgIdStr = "nodeOrg_org_id"
)

func ChainConfigNodeOrgAddCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigNodeOrgAdd",
		Short: "Add nodeOrg",
		Long:  "Add nodeOrg, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,nodeOrg_org_id,nodeOrg_addresses)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodeOrgAdd()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeOrgOrgId, nodeOrgOrgIdStr, "", nodeOrgId)
	flags.StringVar(&nodeOrgAddresses, "nodeOrg_addresses", "", "the nodeOrg addresses")

	return cmd
}

func ChainConfigNodeOrgUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigNodeOrgUpdate",
		Short: "Update nodeOrg",
		Long:  "Update nodeOrg, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,nodeOrg_org_id,nodeOrg_addresses)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodeOrgUpdate()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeOrgOrgId, nodeOrgOrgIdStr, "", nodeOrgId)
	flags.StringVar(&nodeOrgAddresses, "nodeOrg_addresses", "", "the nodeOrg addresses")

	return cmd
}

func ChainConfigNodeOrgDeleteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigNodeOrgDelete",
		Short: "Delete nodeOrg",
		Long:  "Delete nodeOrg, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,nodeOrg_org_id)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodeOrgDelete()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeOrgOrgId, nodeOrgOrgIdStr, "", nodeOrgId)

	return cmd
}

func nodeOrgAdd() error {
	// 构造Payload
	if nodeOrgOrgId == "" || nodeOrgAddresses == "" {
		return errors.New("the nodeOrg orgId or addresses is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeOrgOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte(nodeOrgAddresses),
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), pairs: pairs, oldSeq: seq})
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

func nodeOrgUpdate() error {
	// 构造Payload
	if nodeOrgOrgId == "" || nodeOrgAddresses == "" {
		return errors.New("the nodeOrg orgId or addresses is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeOrgOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte(nodeOrgAddresses),
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), pairs: pairs, oldSeq: seq})
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

func nodeOrgDelete() error {
	// 构造Payload
	if nodeOrgOrgId == "" {
		return errors.New("the nodeOrg orgId is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeOrgOrgId),
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), pairs: pairs, oldSeq: seq})
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
