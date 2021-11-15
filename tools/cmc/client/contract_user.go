/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

const CHECK_PROPOSAL_RESPONSE_FAILED_FORMAT = "checkProposalRequestResp failed, %s"
const SEND_CONTRACT_MANAGE_REQUEST_FAILED_FORMAT = "SendContractManageRequest failed, %s"
const ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT = "admin orgId & key & cert list length not equal, " +
	"[keys len: %d]/[certs len:%d]"
const ADMIN_ORGID_KEY_LENGTH_NOT_EQUAL_FORMAT = "admin orgId & key list length not equal, " +
	"[keys len: %d]/[org-ids len:%d]"

var (
	errAdminOrgIdKeyCertIsEmpty = errors.New("admin orgId or key or cert list is empty")
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
		flagAdminKeyFilePaths, flagAdminCrtFilePaths, flagAdminOrgIds,
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
		flagEnableCertHash, flagContractName, flagMethod, flagParams, flagTimeout, flagSyncResult, flagAbiFilePath,
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
		flagSendTimes, flagContractName, flagMethod, flagParams, flagTimeout, flagSyncResult, flagAbiFilePath,
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
		flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
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
		flagSyncResult, flagEnableCertHash, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes, flagTimeout, flagParams,
		flagSyncResult, flagEnableCertHash, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
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
		flagSdkConfPath, flagContractName, flagOrgId, flagChainId, flagSendTimes, flagTimeout, flagParams,
		flagSyncResult, flagEnableCertHash, flagAdminCrtFilePaths, flagAdminKeyFilePaths, flagAdminOrgIds,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagContractName)

	return cmd
}

func createUserContract() error {
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminCrtFilePaths != "" {
			adminCrts = strings.Split(adminCrtFilePaths, ",")
		}
		if len(adminKeys) != len(adminCrts) {
			return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminOrgIds != "" {
			adminOrgs = strings.Split(adminOrgIds, ",")
		}
		if len(adminKeys) != len(adminOrgs) {
			return fmt.Errorf(ADMIN_ORGID_KEY_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminOrgs))
		}
	}

	rt, ok := common.RuntimeType_value[runtimeType]
	if !ok {
		return fmt.Errorf("unknown runtime type [%s]", runtimeType)
	}

	var kvs []*common.KeyValuePair

	if runtimeType != "EVM" {
		if params != "" {
			kvsMap := make(map[string]string)
			err := json.Unmarshal([]byte(params), &kvsMap)
			if err != nil {
				return err
			}
			kvs = util.ConvertParameters(kvsMap)
		}
	} else {
		byteCode, err := ioutil.ReadFile(byteCodePath)
		if err != nil {
			return err
		}
		byteCodePath = string(byteCode)

		if !ethcmn.IsHexAddress(contractName) {
			contractName = util.CalcEvmContractName(contractName)
		}
		fmt.Printf("EVM contract name in hex: %s\n", contractName)
	}

	payload, err := client.CreateContractCreatePayload(
		contractName,
		version,
		byteCodePath,
		common.RuntimeType(rt),
		kvs,
	)
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				crypto.HashAlgoMap[client.GetHashType()],
				adminOrgs[i],
				payload,
			)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		}
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
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	var kvs []*common.KeyValuePair
	var evmMethod *ethabi.Method

	if abiFilePath != "" { // abi file path 非空 意味着调用的是EVM合约
		abiBytes, err := ioutil.ReadFile(abiFilePath)
		if err != nil {
			return err
		}

		contractAbi, err := ethabi.JSON(bytes.NewReader(abiBytes))
		if err != nil {
			return err
		}

		m, exist := contractAbi.Methods[method]
		if !exist {
			return fmt.Errorf("method '%s' not found", method)
		}
		evmMethod = &m

		inputData, err := util.Pack(evmMethod, params)
		if err != nil {
			return err
		}

		inputDataHexStr := hex.EncodeToString(inputData)
		method = inputDataHexStr[0:8]

		kvs = []*common.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(inputDataHexStr),
			},
		}

		if !ethcmn.IsHexAddress(contractName) {
			contractName = util.CalcEvmContractName(contractName)
		}
		fmt.Printf("EVM contract name in hex: %s\n", contractName)
	} else {
		if params != "" {
			kvsMap := make(map[string]string)
			err := json.Unmarshal([]byte(params), &kvsMap)
			if err != nil {
				return err
			}
			kvs = util.ConvertParameters(kvsMap)
		}
	}

	Dispatch(client, contractName, method, kvs, evmMethod)
	return nil
}

func invokeContractTimes() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	var kvs []*common.KeyValuePair
	var evmMethod *ethabi.Method

	if abiFilePath != "" { // abi file path 非空 意味着调用的是EVM合约
		abiBytes, err := ioutil.ReadFile(abiFilePath)
		if err != nil {
			return err
		}

		contractAbi, err := ethabi.JSON(bytes.NewReader(abiBytes))
		if err != nil {
			return err
		}

		m, exist := contractAbi.Methods[method]
		if !exist {
			return fmt.Errorf("method '%s' not found", method)
		}
		evmMethod = &m

		inputData, err := util.Pack(evmMethod, params)
		if err != nil {
			return err
		}

		inputDataHexStr := hex.EncodeToString(inputData)
		method = inputDataHexStr[0:8]

		kvs = []*common.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(inputDataHexStr),
			},
		}

		if !ethcmn.IsHexAddress(contractName) {
			contractName = util.CalcEvmContractName(contractName)
		}
		fmt.Printf("EVM contract name in hex: %s\n", contractName)
	} else {
		if params != "" {
			kvsMap := make(map[string]string)
			err := json.Unmarshal([]byte(params), &kvsMap)
			if err != nil {
				return err
			}
			kvs = util.ConvertParameters(kvsMap)
		}
	}

	DispatchTimes(client, contractName, method, kvs, evmMethod)
	return nil
}

func getUserContract() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
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

	return nil
}

func upgradeUserContract() error {
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminCrtFilePaths != "" {
			adminCrts = strings.Split(adminCrtFilePaths, ",")
		}
		if len(adminKeys) != len(adminCrts) {
			return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminOrgIds != "" {
			adminOrgs = strings.Split(adminOrgIds, ",")
		}
		if len(adminKeys) != len(adminOrgs) {
			return fmt.Errorf(ADMIN_ORGID_KEY_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminOrgs))
		}
	}

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
	payload, err := client.CreateContractUpgradePayload(contractName, version, byteCodePath, common.RuntimeType(rt),
		pairsKv)
	if err != nil {
		return err
	}

	endorsementEntrys := make([]*common.EndorsementEntry, len(adminKeys))
	for i := range adminKeys {
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				crypto.HashAlgoMap[client.GetHashType()],
				adminOrgs[i],
				payload,
			)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		}
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
	var adminKeys []string
	var adminCrts []string
	var adminOrgs []string

	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath,
		userSignCrtFilePath, userSignKeyFilePath)
	if err != nil {
		return err
	}
	defer client.Stop()

	if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminCrtFilePaths != "" {
			adminCrts = strings.Split(adminCrtFilePaths, ",")
		}
		if len(adminKeys) != len(adminCrts) {
			return fmt.Errorf(ADMIN_ORGID_KEY_CERT_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminCrts))
		}
	} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
		if adminKeyFilePaths != "" {
			adminKeys = strings.Split(adminKeyFilePaths, ",")
		}
		if adminOrgIds != "" {
			adminOrgs = strings.Split(adminOrgIds, ",")
		}
		if len(adminKeys) != len(adminOrgs) {
			return fmt.Errorf(ADMIN_ORGID_KEY_LENGTH_NOT_EQUAL_FORMAT, len(adminKeys), len(adminOrgs))
		}
	}

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
		if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithCert {
			e, err := sdkutils.MakeEndorserWithPath(adminKeys[i], adminCrts[i], payload)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		} else if sdk.AuthTypeToStringMap[client.GetAuthType()] == protocol.PermissionedWithKey {
			e, err := sdkutils.MakePkEndorserWithPath(
				adminKeys[i],
				crypto.HashAlgoMap[client.GetHashType()],
				adminOrgs[i],
				payload,
			)
			if err != nil {
				return err
			}

			endorsementEntrys[i] = e
		}
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
