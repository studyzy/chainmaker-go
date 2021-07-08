/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	sdk "chainmaker.org/chainmaker-sdk-go"
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
		flagSdkConfPath, flagContractName, flagVersion, flagByteCodePath, flagOrgId, flagChainId, flagSendTimes,
		flagRuntimeType, flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagVersion)
	cmd.MarkFlagRequired(flagByteCodePath)
	cmd.MarkFlagRequired(flagRuntimeType)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)

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
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId, flagSendTimes, flagEnableCertHash,
		flagContractName, flagMethod, flagParams, flagTimeout, flagSyncResult, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
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
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId, flagSendTimes,
		flagContractName, flagMethod, flagParams, flagTimeout, flagSyncResult, flagUserTlsKeyFilePath,
		flagUserTlsCrtFilePath, flagEnableCertHash,
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
		flagConcurrency, flagTotalCountPerGoroutine, flagSdkConfPath, flagOrgId, flagChainId, flagSendTimes,
		flagContractName, flagMethod, flagParams, flagTimeout, flagUserTlsCrtFilePath,
		flagUserTlsKeyFilePath, flagEnableCertHash,
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
		flagSdkConfPath, flagContractName, flagVersion, flagByteCodePath, flagOrgId, flagChainId, flagSendTimes,
		flagRuntimeType, flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagVersion)
	cmd.MarkFlagRequired(flagByteCodePath)
	cmd.MarkFlagRequired(flagRuntimeType)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)

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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes,
		flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)

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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes,
		flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)

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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes,
		flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagUserTlsKeyFilePath, flagUserTlsCrtFilePath,
		flagTimeout, flagParams, flagSyncResult, flagEnableCertHash,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)

	return cmd
}

func createUserContract() error {
	var (
		err error
	)

	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT,
			len(adminKeys), len(adminCrts))
	}

	adminClients := make([]*sdk.ChainClient, len(adminKeys))
	for i := 0; i < len(adminClients); i++ {
		if adminClients[i], err = createAdminWithConfig(adminKeys[i], adminCrts[i]); err != nil {
			return fmt.Errorf(CREATE_ADMIN_CLIENT_FAILED_FORMAT, i, err.Error())
		}
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
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

	if len(adminClients) != 1 {
		return fmt.Errorf("unsupport multi sign request, comming soon...")
		//// 在线多签
		//entry, err := adminClients[0].SignMultiSignPayload(payloadBytes)
		//if err != nil {
		//	return fmt.Errorf("admin0 SignMultiSignPayload failed, %s", err.Error())
		//}
		//
		//resp, err := adminClients[0].SendMultiSignReq(sdkPbCommon.TxType_MANAGE_USER_CONTRACT, payloadBytes,
		//	entry, 100000, int64(timeout))
		//if err != nil {
		//	return fmt.Errorf("admin0 SendMultiSignReq failed, %s", err.Error())
		//}
		//
		//err = util.CheckProposalRequestResp(resp, true)
		//if err != nil {
		//	return fmt.Errorf("admin0 checkProposalRequestResp failed, %s", err.Error())
		//}
		//
		//fmt.Printf("admin0 send multi sign req resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		//txId := string(resp.ContractResult.Result)
		//// 休眠，等待多签请求完成
		//time.Sleep(5 * time.Second)
		//fmt.Printf("txId:%s\n", txId)
		//
		//for i:=1; i<len(adminClients); i++ {
		//	entry, err = adminClients[i].SignMultiSignPayload(payloadBytes)
		//	if err != nil {
		//		return fmt.Errorf("admin%d SignMultiSignPayload failed, %s", i, err.Error())
		//	}
		//
		//	resp, err = adminClients[i].SendMultiSignVote(sdkPbCommon.VoteStatus_AGREE, txId, "", entry, -1)
		//	if err != nil {
		//		return fmt.Errorf("admin%d SendMultiSignVote failed, %s", i, err.Error())
		//	}
		//
		//	err = util.CheckProposalRequestResp(resp, true)
		//	if err != nil {
		//		return fmt.Errorf("admin%d checkProposalRequestResp failed, %s", i, err.Error())
		//	}
		//
		//	fmt.Printf("admin%d send multi sign vote resp: code:%d, msg:%s, payload:%+v\n",
		//		i, resp.Code, resp.Message, resp.ContractResult)
		//}

		// 离线多签
		//var signedPayloadBytes [][]byte
		//for i:=0; i<len(adminClients); i++ {
		//	payload, err := adminClients[i].SignContractManagePayload(payloadBytes)
		//	if err != nil {
		//		return fmt.Errorf("SignContractManagePayload failed, %s", err.Error())
		//	}
		//
		//	signedPayloadBytes = append(signedPayloadBytes, payload)
		//}
		//
		//mergeSignedPayloadBytes, err := client.MergeContractManageSignedPayload(signedPayloadBytes)
		//if err != nil {
		//	return fmt.Errorf("MergeContractManageSignedPayload failed, %s", err.Error())
		//}
		//
		//// 发送创建合约请求
		//resp, err := client.SendContractManageRequest(mergeSignedPayloadBytes, int64(timeout), false)
		//if err != nil {
		//	return fmt.Errorf("SendContractManageRequest failed, %s", err.Error())
		//}
		//
		//err = util.CheckProposalRequestResp(resp, true)
		//if err != nil {
		//	return fmt.Errorf("checkProposalRequestResp failed, %s", err.Error())
		//}
		//
		//fmt.Printf("create contract resp: %+v\n", resp)
	}

	// 单签模式
	signedPayloadBytes1, err := adminClients[0].SignContractManagePayload(payloadBytes)
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
	var (
		err error
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
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
	var (
		err error
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
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
	var (
		err error
	)

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
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
	var (
		err error
	)

	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT,
			len(adminKeys), len(adminCrts))
	}

	adminClients := make([]*sdk.ChainClient, len(adminKeys))
	for i := 0; i < len(adminClients); i++ {
		if adminClients[i], err = createAdminWithConfig(adminKeys[i], adminCrts[i]); err != nil {
			return fmt.Errorf(CREATE_ADMIN_CLIENT_FAILED_FORMAT, i, err.Error())
		}
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
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

	if len(adminClients) != 1 {
		return fmt.Errorf("unsupport multi sign request, comming soon... ")
	}

	// 单签模式
	signedPayloadBytes1, err := adminClients[0].SignContractManagePayload(payloadBytes)
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
		err            error
		payloadBytes   []byte
		whichOperation string
	)

	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_KEY_AND_CERT_NOT_ENOUGH_FORMAT,
			len(adminKeys), len(adminCrts))
	}

	adminClients := make([]*sdk.ChainClient, len(adminKeys))
	for i := 0; i < len(adminClients); i++ {
		if adminClients[i], err = createAdminWithConfig(adminKeys[i], adminCrts[i]); err != nil {
			return fmt.Errorf(CREATE_ADMIN_CLIENT_FAILED_FORMAT, i, err.Error())
		}
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
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
	if len(adminClients) != 1 {
		return fmt.Errorf("unsupport multi sign request, comming soon... ")
	}

	// 单签模式
	signedPayloadBytes1, err := adminClients[0].SignContractManagePayload(payloadBytes)
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
