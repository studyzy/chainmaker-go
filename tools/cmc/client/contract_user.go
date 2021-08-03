/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/common"
	sdk "chainmaker.org/chainmaker/sdk-go"
	"github.com/spf13/cobra"
)

const CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT = "checkProposalRequestResp failed, %s"
const SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT = "SendContractManageRequest failed, %s"
const ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT = "admin orgId & key & cert list length not equal, [keys len: %d]/[certs len:%d]"

var (
	ErrAdminOrgIdKeyCertIsEmpty = errors.New("admin orgId or key or cert list is empty")
)

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
		flagAdminKeyFilePaths, flagAdminCrtFilePaths,
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
		flagAdminCrtFilePaths, flagAdminKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagVersion)
	cmd.MarkFlagRequired(flagByteCodePath)
	cmd.MarkFlagRequired(flagRuntimeType)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)

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
		flagSyncResult, flagEnableCertHash, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)

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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes, flagTimeout, flagParams,
		flagSyncResult, flagEnableCertHash, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)

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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes, flagTimeout, flagParams,
		flagSyncResult, flagEnableCertHash, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)

	return cmd
}

func createUserContract() error {
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 {
		return ErrAdminOrgIdKeyCertIsEmpty
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	rt, ok := common.RuntimeType_value[runtimeType]
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
	kvs := util.ConvertParameters(pairs)
	payload, err := client.CreateContractCreatePayload(contractName, version, byteCodePath, common.RuntimeType(rt), kvs)
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdk.SignPayloadWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}

		endorsementEntrys[i] = e
	}

	resp, err := client.SendContractManageRequest(payload, endorsementEntrys, timeout, syncResult)
	if err != nil {
		return err
	}
	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return err
	}
	fmt.Printf("response: %+v\n", resp)
	return nil
}

func invokeUserContract() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

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
		return err
	}
	defer client.Stop()

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
		return err
	}
	defer client.Stop()

	pairs := make(map[string]string)
	if params != "" {
		err := json.Unmarshal([]byte(params), &pairs)
		if err != nil {
			return err
		}
	}

	resp, err := client.QueryContract(contractName, method, util.ConvertParameters(pairs), -1)
	if err != nil {
		return fmt.Errorf("query contract failed, %s", err.Error())
	}

	fmt.Printf("QUERY contract resp: %+v\n", resp)

	client.Stop()
	return nil
}

func upgradeUserContract() error {
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 {
		return ErrAdminOrgIdKeyCertIsEmpty
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	rt, ok := common.RuntimeType_value[runtimeType]
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
	pairsKv := util.ConvertParameters(pairs)
	fmt.Printf("upgrade user contract params:%+v\n", pairsKv)
	payload, err := client.CreateContractUpgradePayload(contractName, version, byteCodePath, common.RuntimeType(rt), pairsKv)
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdk.SignPayloadWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}

		endorsementEntrys[i] = e
	}

	// 发送更新合约请求
	resp, err := client.SendContractManageRequest(payload, endorsementEntrys, timeout, syncResult)
	if err != nil {
		return fmt.Errorf(SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT, err.Error())
	}

	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf(CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT, err.Error())
	}

	fmt.Printf("upgrade contract resp: %+v\n", resp)

	return nil
}

func freezeOrUnfreezeOrRevokeUserContract(which int) error {
	adminKeys := strings.Split(adminKeyFilePaths, ",")
	adminCrts := strings.Split(adminCrtFilePaths, ",")
	if len(adminKeys) == 0 || len(adminCrts) == 0 {
		return ErrAdminOrgIdKeyCertIsEmpty
	}
	if len(adminKeys) != len(adminCrts) {
		return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
	}

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath, userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	var (
		payload        *common.Payload
		whichOperation string
	)

	switch which {
	case 1:
		payload, err = client.CreateContractFreezePayload(contractName)
		whichOperation = "freeze"
	case 2:
		payload, err = client.CreateContractUnfreezePayload(contractName)
		whichOperation = "unfreeze"
	case 3:
		payload, err = client.CreateContractRevokePayload(contractName)
		whichOperation = "revoke"
	default:
		err = fmt.Errorf("wrong which param")
	}
	if err != nil {
		return fmt.Errorf("create cert manage %s payload failed, %s", whichOperation, err.Error())
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		e, err := sdk.SignPayloadWithPath(adminKeys[i], adminCrts[i], payload)
		if err != nil {
			return err
		}

		endorsementEntrys[i] = e
	}

	// 发送创建合约请求
	resp, err := client.SendContractManageRequest(payload, endorsementEntrys, timeout, syncResult)
	if err != nil {
		return fmt.Errorf(SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT, err.Error())
	}

	err = util.CheckProposalRequestResp(resp, false)
	if err != nil {
		return fmt.Errorf(CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT, err.Error())
	}

	fmt.Printf("%s contract resp: %+v\n", whichOperation, resp)

	return nil
}
