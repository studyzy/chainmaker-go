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
	nodeAddrOrgId  string
	nodeAddresses  string
	nodeOldAddress string
	nodeNewAddress string
)
var (
	nodeOrgIdStr     = "the nodeAddr org id"
	nodeaddrorgidStr = "node_addr_org_id"
)

func ChainConfigNodeAddrAddCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigNodeAddrAdd",
		Short: "Add nodeAddr",
		Long:  "Add nodeAddr, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,node_addr_org_id,node_addresses)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodeAddrAdd()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeAddrOrgId, nodeaddrorgidStr, "", nodeOrgIdStr)
	flags.StringVar(&nodeAddresses, "node_addresses", "", "the node addresses")

	return cmd
}

func ChainConfigNodeAddrUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigNodeAddrUpdate",
		Short: "Update nodeAddr",
		Long:  "Update nodeAddr, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,node_addr_org_id,node_old_address,node_new_address)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodeAddrUpdate()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeAddrOrgId, nodeaddrorgidStr, "", nodeOrgIdStr)
	flags.StringVar(&nodeOldAddress, "node_old_address", "", "the old address")
	flags.StringVar(&nodeNewAddress, "node_new_address", "", "the new address")

	return cmd
}

func ChainConfigNodeAddrDeleteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigNodeAddrDelete",
		Short: "Delete nodeAddr",
		Long:  "Delete nodeAddr, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,node_addr_org_id,node_old_address)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nodeAddrDelete()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&nodeAddrOrgId, nodeaddrorgidStr, "", nodeOrgIdStr)
	flags.StringVar(&nodeOldAddress, "node_old_address", "", "the old address")

	return cmd
}

func nodeAddrAdd() error {
	// 构造Payload
	if nodeAddrOrgId == "" || nodeAddresses == "" {
		return errors.New("the nodeAddr orgId or addresses is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeAddrOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte(nodeAddresses),
	})

	fmt.Println("pairs: ", pairs)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ID_ADD.String(), pairs: pairs, oldSeq: seq})
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

func nodeAddrUpdate() error {
	// 构造Payload
	if nodeAddrOrgId == "" || nodeOldAddress == "" || nodeNewAddress == "" {
		return errors.New("the nodeAddr orgId or node_old_address is empty or nodeNewAddress is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeAddrOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_id",
		Value: []byte(nodeOldAddress),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "new_node_id",
		Value: []byte(nodeNewAddress),
	})

	fmt.Println("pairs: ", pairs)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), pairs: pairs, oldSeq: seq})
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

func nodeAddrDelete() error {
	// 构造Payload
	if nodeAddrOrgId == "" || nodeOldAddress == "" {
		return errors.New("the nodeAddr orgId is empty or node_old_address is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeAddrOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_id",
		Value: []byte(nodeOldAddress),
	})

	fmt.Println("pairs: ", pairs)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), pairs: pairs, oldSeq: seq})
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
