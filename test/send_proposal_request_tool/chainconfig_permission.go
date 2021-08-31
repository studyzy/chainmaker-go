/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	permissionResourceName  string
	permissionPrincipleJson string
)

const (
	resourceName               = "resource_name"
	principleJson              = "principle_json"
	permissionResourceNameStr  = "permission_resource_name"
	permissionPrincipleJsonStr = "permission_principle_json"
)

func ChainConfigPermissionAddCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigPermissionAdd",
		Short: "Add Permission",
		Long:  "Add Permission, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,permission_resource_name,permission_principle_json)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return permissionAdd()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&permissionResourceName, permissionResourceNameStr, "", resourceName)
	flags.StringVar(&permissionPrincipleJson, permissionPrincipleJsonStr, "", principleJson)

	return cmd
}

func ChainConfigPermissionUpdateCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigPermissionUpdate",
		Short: "Update permission",
		Long:  "Update permission, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,permission_resource_name,permission_principle_json)\"",
		RunE: func(_ *cobra.Command, _ []string) error {
			return permissionUpdate()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&permissionResourceName, permissionResourceNameStr, "", resourceName)
	flags.StringVar(&permissionPrincipleJson, permissionPrincipleJsonStr, "", principleJson)

	return cmd
}

func ChainConfigPermissionDeleteCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainConfigPermissionDelete",
		Short: "Delete permission",
		Long:  "Delete permission, the params(seq,org-ids,admin-sign-keys,admin-sign-crts,permission_resource_name)\"",
		RunE: func(_ *cobra.Command, _ []string) error {
			return permissionDelete()
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&permissionResourceName, permissionResourceNameStr, "", resourceName)

	return cmd
}

func permissionAdd() error {
	// 构造Payload
	if permissionResourceName == "" {
		return errors.New("the permission resource name is empty in permissionAdd")
	}

	principle := &acPb.Policy{}
	err := json.Unmarshal([]byte(permissionPrincipleJson), principle)
	if err != nil {
		return err
	}
	pbStr, err := proto.Marshal(principle)

	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   permissionResourceName,
		Value: pbStr,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_PERMISSION_ADD.String(), pairs: pairs, oldSeq: seq})
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

func permissionUpdate() error {
	// 构造Payload
	if permissionResourceName == "" {
		return errors.New("the permission resource name is empty in permissionUpdate")
	}

	principle := &acPb.Policy{}
	err := json.Unmarshal([]byte(permissionPrincipleJson), principle)
	if err != nil {
		return err
	}
	pbStr, err := proto.Marshal(principle)

	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   permissionResourceName,
		Value: pbStr,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), pairs: pairs, oldSeq: seq})
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

func permissionDelete() error {
	// 构造Payload
	if permissionResourceName == "" {
		return errors.New("the permission resource name is empty in permissionDelete")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: permissionResourceName,
	})

	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), pairs: pairs, oldSeq: seq})
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
