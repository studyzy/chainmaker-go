/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	sdkPbCommon "chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

const CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT = "checkProposalRequestResp failed, %s"
const SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT = "SendContractManageRequest failed, %s"
const MERGE_CONTRACT_MANAGE_SIGNED_PAYLOAD_FAILED_FORMAT = "MergeContractManageSignedPayload failed, %s"
const SIGN_CONTRACT_MANAGE_PAYLOAD_FAILED_FORMAT = "SignContractManagePayload failed, %s"
const CREATE_USER_CLIENT_FAILED_FORMAT = "create user client failed, %s"
const CREATE_ADMIN_CLIENT_FAILED_FORMAT = "create admin client failed, [No.%d], %s"
const ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT = "admin key and cert list length not equal, [keys len: %d]/[certs len:%d]"

type UserContract struct {
	ContractName string
	Method       string
	Params       map[string]string
}

func userContractCMD() *cobra.Command {
	userContractCmd := &cobra.Command{
		Use:   "user",
		Short: "user contract command",
		Long:  "user contract command",
	}

	userContractCmd.AddCommand(createUserContractCMD())
	userContractCmd.AddCommand(invokeContractTimesCMD())
	userContractCmd.AddCommand(invokeUserContractCMD())
	userContractCmd.AddCommand(upgradeUserContractCMD())
	userContractCmd.AddCommand(freezeUserContractCMD())
	userContractCmd.AddCommand(unfreezeUserContractCMD())
	userContractCmd.AddCommand(revokeUserContractCMD())
	userContractCmd.AddCommand(getUserContractCMD())

	return userContractCmd
}

func createUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create user contract command",
		Long:  "create user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return createUserContract()
		},
	}

	attachFlags(cmd, []string{
		flagUserTlsKeyFilePath, flagUserTlsCrtFilePath, flagUserSignKeyFilePath, flagUserSignCrtFilePath,
		flagSdkConfPath, flagContractName, flagVersion, flagByteCodePath, flagOrgId, flagChainId, flagSendTimes,
		flagRuntimeType, flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagVersion)
	cmd.MarkFlagRequired(flagByteCodePath)
	cmd.MarkFlagRequired(flagRuntimeType)

	return cmd
}

func invokeUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "invoke user contract command",
		Long:  "invoke user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return invokeUserContract()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId, flagSendTimes,
		flagEnableCertHash, flagContractName, flagMethod, flagParams, flagTimeout, flagSyncResult,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagMethod)

	return cmd
}
func invokeContractTimesCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke-times",
		Short: "invoke contract times command",
		Long:  "invoke contract times command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return invokeContractTimes()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagEnableCertHash, flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagSendTimes, flagContractName, flagMethod, flagParams, flagTimeout, flagSyncResult,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagMethod)

	return cmd
}

func getUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get user contract command",
		Long:  "get user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getUserContract()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagEnableCertHash, flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId,
		flagSendTimes, flagContractName, flagMethod, flagParams, flagTimeout,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagMethod)

	return cmd
}

func upgradeUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "upgrade user contract command",
		Long:  "upgrade user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return upgradeUserContract()
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagSdkConfPath, flagContractName, flagVersion, flagByteCodePath, flagOrgId, flagChainId, flagSendTimes,
		flagRuntimeType, flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagVersion)
	cmd.MarkFlagRequired(flagByteCodePath)
	cmd.MarkFlagRequired(flagRuntimeType)

	return cmd
}

func freezeUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze",
		Short: "freeze user contract command",
		Long:  "freeze user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return freezeOrUnfreezeOrRevokeUserContract(1)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes, flagTimeout, flagParams,
		flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

func unfreezeUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze",
		Short: "unfreeze user contract command",
		Long:  "unfreeze user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return freezeOrUnfreezeOrRevokeUserContract(2)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

func revokeUserContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "revoke user contract command",
		Long:  "revoke user contract command",
		RunE: func(_ *cobra.Command, _ []string) error {
			return freezeOrUnfreezeOrRevokeUserContract(3)
		},
	}

	attachFlags(cmd, []string{
		flagUserSignKeyFilePath, flagUserSignCrtFilePath, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

func createUserContract() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err.Error())
	}

	rt, ok := sdkPbCommon.RuntimeType_value[runtimeType]
	if !ok {
		return fmt.Errorf("unknown runtime type [%s]", runtimeType)
	}

	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}
	pairsKv := paramsMap2KVPairs(pairs)
	fmt.Printf("create user contract params:%+v\n", pairsKv)
	payloadBytes, err := client.CreateContractCreatePayload(contractName, version, byteCodePath, sdkPbCommon.RuntimeType(rt), pairsKv)

	// 单签模式
	signedPayloadBytes1, err := client.SignContractManagePayload(payloadBytes)
	if err != nil {
		return fmt.Errorf(MERGE_CONTRACT_MANAGE_SIGNED_PAYLOAD_FAILED_FORMAT, err.Error())
	}

	mergeSignedPayloadBytes, err := client.MergeContractManageSignedPayload([][]byte{signedPayloadBytes1})
	if err != nil {
		return fmt.Errorf(SIGN_CONTRACT_MANAGE_PAYLOAD_FAILED_FORMAT, err.Error())
	}

	// 发送创建合约请求
	resp, err := client.SendContractManageRequest(mergeSignedPayloadBytes, int64(timeout), syncResult)
	if err != nil {
		return fmt.Errorf(SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT, err.Error())
	}

	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return fmt.Errorf(CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT, err.Error())
	}

	fmt.Printf("create contract resp: %+v\n", resp)

	client.Stop()

	return nil
}

func invokeUserContract() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err.Error())
	}

	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	Dispatch(client, contractName, method, pairs)

	client.Stop()
	return nil
}

func invokeContractTimes() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err.Error())
	}

	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	DispatchTimes(client, contractName, method, pairs)

	client.Stop()
	return nil
}

func getUserContract() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err.Error())
	}

	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.QueryContract(contractName, method, pairs, -1)
	if err != nil {
		return fmt.Errorf("query contract failed, %s", err.Error())
	}

	fmt.Printf("QUERY contract resp: %+v\n", resp)

	client.Stop()
	return nil
}

func paramsMap2KVPairs(params map[string]string) (kvPairs []*sdkPbCommon.KeyValuePair) {
	for key, val := range params {
		kvPair := &sdkPbCommon.KeyValuePair{
			Key:   key,
			Value: val,
		}

		kvPairs = append(kvPairs, kvPair)
	}

	return
}

func upgradeUserContract() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err.Error())
	}
	defer client.Stop()
	rt, ok := sdkPbCommon.RuntimeType_value[runtimeType]
	if !ok {
		return fmt.Errorf("unknown runtime type [%s]", runtimeType)
	}

	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}
	pairsKv := paramsMap2KVPairs(pairs)
	fmt.Printf("upgrade user contract params:%+v\n", pairsKv)
	payloadBytes, err := client.CreateContractUpgradePayload(contractName, version, byteCodePath, sdkPbCommon.RuntimeType(rt), pairsKv)

	// 单签模式
	signedPayloadBytes1, err := client.SignContractManagePayload(payloadBytes)
	if err != nil {
		return fmt.Errorf(SIGN_CONTRACT_MANAGE_PAYLOAD_FAILED_FORMAT, err.Error())
	}

	mergeSignedPayloadBytes, err := client.MergeContractManageSignedPayload([][]byte{signedPayloadBytes1})
	if err != nil {
		return fmt.Errorf(MERGE_CONTRACT_MANAGE_SIGNED_PAYLOAD_FAILED_FORMAT, err.Error())
	}

	// 发送创建合约请求
	resp, err := client.SendContractManageRequest(mergeSignedPayloadBytes, int64(timeout), syncResult)
	if err != nil {
		return fmt.Errorf(SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT, err.Error())
	}

	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return fmt.Errorf(CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT, err.Error())
	}

	fmt.Printf("upgrade contract resp: %+v\n", resp)

	return nil
}

func freezeOrUnfreezeOrRevokeUserContract(which int) error {
	var (
		payloadBytes   []byte
		whichOperation string
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return fmt.Errorf(CREATE_USER_CLIENT_FAILED_FORMAT, err.Error())
	}
	defer client.Stop()

	switch which {
	case 1:
		payloadBytes, err = client.CreateContractFreezePayload(contractName)
		whichOperation = "freeze"
	case 2:
		payloadBytes, err = client.CreateContractUnfreezePayload(contractName)
		whichOperation = "unfreeze"
	case 3:
		payloadBytes, err = client.CreateContractRevokePayload(contractName)
		whichOperation = "revoke"
	default:
		err = fmt.Errorf("wrong which param")
	}
	if err != nil {
		return fmt.Errorf("create cert manage %s payload failed, %s", whichOperation, err.Error())
	}

	// 单签模式
	signedPayloadBytes1, err := client.SignContractManagePayload(payloadBytes)
	if err != nil {
		return fmt.Errorf(SIGN_CONTRACT_MANAGE_PAYLOAD_FAILED_FORMAT, err.Error())
	}

	mergeSignedPayloadBytes, err := client.MergeContractManageSignedPayload([][]byte{signedPayloadBytes1})
	if err != nil {
		return fmt.Errorf(MERGE_CONTRACT_MANAGE_SIGNED_PAYLOAD_FAILED_FORMAT, err.Error())
	}

	// 发送创建合约请求
	resp, err := client.SendContractManageRequest(mergeSignedPayloadBytes, int64(timeout), syncResult)
	if err != nil {
		return fmt.Errorf(SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT, err.Error())
	}

	err = util.CheckProposalRequestResp(resp, true)
	if err != nil {
		return fmt.Errorf(CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT, err.Error())
	}

	fmt.Printf("%s contract resp: %+v\n", whichOperation, resp)

	return nil
}
