/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/json"
	"errors"
	"fmt"

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
		Value: nodeAddrOrgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "addresses",
		Value: nodeAddresses,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_UPDATE_CHAIN_CONFIG, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), method: commonPb.ConfigFunction_NODE_ID_ADD.String(), pairs: pairs, oldSeq: seq})
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

func nodeAddrUpdate() error {
	// 构造Payload
	if nodeAddrOrgId == "" || nodeOldAddress == "" || nodeNewAddress == "" {
		return errors.New("the nodeAddr orgId or node_old_address is empty or nodeNewAddress is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: nodeAddrOrgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: nodeOldAddress,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "new_address",
		Value: nodeNewAddress,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_UPDATE_CHAIN_CONFIG, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), method: commonPb.ConfigFunction_NODE_ID_UPDATE.String(), pairs: pairs, oldSeq: seq})
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

func nodeAddrDelete() error {
	// 构造Payload
	if nodeAddrOrgId == "" || nodeOldAddress == "" {
		return errors.New("the nodeAddr orgId is empty or node_old_address is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: nodeAddrOrgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: nodeOldAddress,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_UPDATE_CHAIN_CONFIG, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), method: commonPb.ConfigFunction_NODE_ID_DELETE.String(), pairs: pairs, oldSeq: seq})
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
