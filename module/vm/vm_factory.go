/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// verify and run contract
package vm

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	"chainmaker.org/chainmaker-go/utils"

	"chainmaker.org/chainmaker-go/evm"
	"chainmaker.org/chainmaker-go/gasm"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/vm/native"
	"chainmaker.org/chainmaker-go/wasmer"
	"chainmaker.org/chainmaker-go/wxvm"
	"chainmaker.org/chainmaker-go/wxvm/xvm"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

const WxvmCodeFolder = "wxvm"

type Factory struct {
}

// NewVmManager get vm runtime manager
func (f *Factory) NewVmManager(wxvmCodePathPrefix string, AccessControl protocol.AccessControlProvider,
	provider protocol.ChainNodesInfoProvider, chainConf protocol.ChainConf) protocol.VmManager {

	chainId := chainConf.ChainConfig().ChainId
	log := logger.GetLoggerByChain(logger.MODULE_VM, chainId)

	wxvmCodeDir := filepath.Join(wxvmCodePathPrefix, chainId, WxvmCodeFolder)
	log.Infof("init wxvm code dir %s", wxvmCodeDir)
	wasmerVmPoolManager := wasmer.NewVmPoolManager(chainId)
	wxvmCodeManager := xvm.NewCodeManager(chainId, wxvmCodeDir)
	wxvmContextService := xvm.NewContextService(chainId)

	return &ManagerImpl{
		ChainId:                chainId,
		WasmerVmPoolManager:    wasmerVmPoolManager,
		WxvmCodeManager:        wxvmCodeManager,
		WxvmContextService:     wxvmContextService,
		AccessControl:          AccessControl,
		ChainNodesInfoProvider: provider,
		Log:                    log,
		ChainConf:              chainConf,
	}
}

// Interface of smart contract engine runtime
type RuntimeInstance interface {
	// start vm runtime with invoke, call “method”
	Invoke(contractId *commonPb.Contract, method string, byteCode []byte, parameters map[string][]byte,
		txContext protocol.TxSimContext, gasUsed uint64) *commonPb.ContractResult
}

type ManagerImpl struct {
	WasmerVmPoolManager    *wasmer.VmPoolManager
	WxvmCodeManager        *xvm.CodeManager
	WxvmContextService     *xvm.ContextService
	SnapshotManager        protocol.SnapshotManager
	AccessControl          protocol.AccessControlProvider
	ChainNodesInfoProvider protocol.ChainNodesInfoProvider
	ChainId                string
	Log                    *logger.CMLogger
	ChainConf              protocol.ChainConf // chain config
}

func (m *ManagerImpl) GetAccessControl() protocol.AccessControlProvider {
	return m.AccessControl
}

func (m *ManagerImpl) GetChainNodesInfoProvider() protocol.ChainNodesInfoProvider {
	return m.ChainNodesInfoProvider
}

func (m *ManagerImpl) RunContract(contract *commonPb.Contract, method string, byteCode []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {

	contractResult := &commonPb.ContractResult{
		Code:    1,
		Result:  nil,
		Message: "",
	}

	contractName := contract.Name
	if contractName == "" {
		contractResult.Message = "contractName not found"
		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_CONTRACT_NAME
	}

	if parameters == nil {
		parameters = make(map[string][]byte)
	}

	// return error if contract has been revoked
	revokeKey := []byte(protocol.ContractRevoke + contractName)
	if revokeInfo, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), revokeKey); err != nil {
		contractResult.Message = fmt.Sprintf("unable to find revoke info for contract:%s,  error:%s", contractName, err.Error())
		return contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED
	} else if len(revokeInfo) != 0 {
		contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been revoked.", contractName)
		return contractResult, commonPb.TxStatusCode_CONTRACT_REVOKE_FAILED
	}

	if native.IsNative(contractName, refTxType) {
		if method == "" {
			contractResult.Message = "require param method not found."
			return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_METHOD
		}
		return m.runNativeContract(contract, method, parameters, txContext)
	} else if m.isUserContract(refTxType) {
		return m.runUserContract(contract, method, byteCode, parameters, txContext, gasUsed, refTxType)
	} else {
		contractResult.Message = fmt.Sprintf("bad contract call %s, transaction type %s", contractName, refTxType)
		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_TRANSACTION_TYPE
	}
}

// runNativeContract invoke native contract
func (m *ManagerImpl) runNativeContract(contract *commonPb.Contract, method string, parameters map[string][]byte,
	txContext protocol.TxSimContext) (*commonPb.ContractResult, commonPb.TxStatusCode) {

	runtimeInstance := native.GetRuntimeInstance(m.ChainId)
	runtimeContractResult := runtimeInstance.Invoke(contract, method, nil, parameters, txContext)

	if runtimeContractResult.Code == 0 {
		return runtimeContractResult, commonPb.TxStatusCode_SUCCESS
	} else {
		return runtimeContractResult, commonPb.TxStatusCode_CONTRACT_FAIL
	}
}

// runUserContract invoke user contract
func (m *ManagerImpl) runUserContract(contract *commonPb.Contract, method string, byteCode []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64, refTxType commonPb.TxType) (contractResult *commonPb.ContractResult, code commonPb.TxStatusCode) {

	var (
		myContract   = contract
		contractName = contract.Name
		status       = contract.Status
		//versionKey     = []byte(protocol.ContractVersion + contractName)
		//creatorKey     = []byte(protocol.ContractCreator + contractName)
		//freezeKey      = []byte(protocol.ContractFreeze + contractName)
		//revokeKey      = []byte(protocol.ContractRevoke + contractName)
		//runtimeTypeKey = []byte(protocol.ContractRuntimeType + contractName)
	)
	contractResult = &commonPb.ContractResult{Code: 1}
	if status == commonPb.ContractStatus_ALL { // 只传入的ContractName，其他属性需要从DB获取
		dbContract, err := utils.GetContractByName(txContext.Get, contractName)
		if err != nil {
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		}
		myContract = dbContract
	}
	if len(byteCode) == 0 {
		dbByteCode, err := utils.GetContractBytecode(txContext.Get, contractName)
		if err != nil {
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		}
		byteCode = dbByteCode
	}
	// return msg if contract has been frozen
	if myContract.Status != commonPb.ContractStatus_NORMAL {
		contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been frozen.", contractName)
		return contractResult, commonPb.TxStatusCode_CONTRACT_FREEZE_FAILED
	}

	//if refTxType == commonPb.TxType_MANAGE_USER_CONTRACT &&
	//	(method == commonPb.ManageUserContractFunction_UNFREEZE_CONTRACT.String() ||
	//		method == commonPb.ManageUserContractFunction_REVOKE_CONTRACT.String()) {
	//	// nothing
	//} else if freezeInfo, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), freezeKey); err != nil {
	//	contractResult.Message = fmt.Sprintf("unable to find freeze info for contract:%s,  error:%s", contractName, err.Error())
	//	return contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED
	//} else if len(freezeInfo) != 0 {
	//	contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been frozen.", contractName)
	//	return contractResult, commonPb.TxStatusCode_CONTRACT_FREEZE_FAILED
	//}

	// init call user contract data
	//if refTxType == commonPb.TxType_INVOKE_CONTRACT || refTxType == commonPb.TxType_QUERY_CONTRACT {
	//excludeMethodList := make([]string, 0)
	//excludeMethodList = append(excludeMethodList, protocol.ContractInitMethod)
	//excludeMethodList = append(excludeMethodList, protocol.ContractUpgradeMethod)
	//excludeMethodList = append(excludeMethodList, "")
	//
	//vt := &verifyType{
	//	requireVersion:       true,
	//	requireExcludeMethod: true,
	//	requireByteCode:      true,
	//	requireRuntimeType:   true,
	//	excludeMethodList:    excludeMethodList,
	//	currentMethod:        method,
	//}
	//
	//result, code, byteCodeTmp, versionTmp, runtimeTypeTmp := vt.commonVerify(txContext, contractId, contractResult)
	//if code != commonPb.TxStatusCode_SUCCESS {
	//	return result, code
	//}

	//byteCode, _ = txContext.GetBlockchainStore().GetContractBytecode(contractName)
	//contract, _ := txContext.GetBlockchainStore().GetContractByName(contractName)
	//contractId.Version = contract.Version
	//contractId.RuntimeType = contract.RuntimeType
	//version = contract.Version
	//runtimeType = int(contract.RuntimeType)
	//}

	// manager contract logic
	//switch method {
	//case consts.ContractManager_INIT_CONTRACT.String():
	//	method = protocol.ContractInitMethod
	//	vt := &verifyType{
	//		requireVersion:       false,
	//		requireNullVersion:   true,
	//		requireExcludeMethod: false,
	//		currentMethod:        method,
	//		requireFormatVersion: true,
	//	}
	//	result, code, _, _, _ := vt.commonVerify(txContext, contract, contractResult)
	//	if code != commonPb.TxStatusCode_SUCCESS {
	//		return result, code
	//	}
	//
	//	// If you call the constructor, you need to take byteCode in the parameter
	//	if byteCode == nil {
	//		contractResult.Message = fmt.Sprintf("please provide the bytecode of the contract:%+v while creating contract", contract)
	//		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_BYTE_CODE
	//	}
	//
	//	if contractId.RuntimeType != commonPb.RuntimeType_INVALID {
	//		runtimeType = int(contractId.RuntimeType)
	//	} else {
	//		contractResult.Message = fmt.Sprintf("please provide the runtime type of the contract:%+v while creating contract", contractId)
	//		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_RUNTIME_TYPE
	//	}
	//
	//	versionedByteCodeKey := append([]byte(protocol.ContractByteCode+contractName), []byte(version)...) // <contract name>:B:<contract version>
	//	// save versioned byteCode
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionKey, []byte(version)); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store byte code for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//
	//	// save versioned byteCode
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionedByteCodeKey, byteCode); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store byte code for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//
	//	// save sender
	//	if senderByte, err := proto.Marshal(txContext.GetTx().Header.Sender); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store creator for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	} else {
	//		if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), creatorKey, senderByte); err != nil {
	//			contractResult.Message = fmt.Sprintf("failed to store creator for contract:%s, error:%s", contractName, err.Error())
	//			return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//		}
	//	}
	//
	//	// save runtime type
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), runtimeTypeKey, []byte(strconv.Itoa(runtimeType))); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store runtime contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//	m.Log.Infof("create contract begin %+v, %+v, byte code len %d", contractId, parameters, len(byteCode))
	//case consts.ContractManager_UPGRADE_CONTRACT.String():
	//	method = protocol.ContractUpgradeMethod
	//	vt := &verifyType{
	//		requireVersion:       true,
	//		requireNullVersion:   false,
	//		requireExcludeMethod: false,
	//		requireRuntimeType:   true,
	//		currentMethod:        method,
	//		requireFormatVersion: true,
	//	}
	//	result, code, _, _, runtimeTypeTmp := vt.commonVerify(txContext, contractId, contractResult)
	//	if code != commonPb.TxStatusCode_SUCCESS {
	//		return result, code
	//	}
	//	runtimeType = runtimeTypeTmp
	//
	//	// If you call the constructor, you need to take byteCode in the parameter
	//	if byteCode == nil {
	//		contractResult.Message = fmt.Sprintf("please provide the bytecode of the contract:%+v while upgrading", contractId)
	//		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_BYTE_CODE
	//	}
	//
	//	versionedByteCodeKey := append([]byte(protocol.ContractByteCode+contractName), []byte(version)...) // <contract name>:B:<contract version>
	//	// check version exists
	//	if byteCodeInContext, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionedByteCodeKey); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to get byte code in tx context for contract %s, %s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_INTERNAL_ERROR
	//	} else if len(byteCodeInContext) > 0 {
	//		contractResult.Message = fmt.Sprintf("the contract version [%s][%s] already exists.", contractName, version)
	//		return contractResult, commonPb.TxStatusCode_CONTRACT_VERSION_EXIST_FAILED
	//	}
	//
	//	// save versioned byteCode
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionKey, []byte(version)); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store byte code for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//
	//	// save versioned byteCode
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionedByteCodeKey, byteCode); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store byte code for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//
	//	// save runtime type
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), runtimeTypeKey, []byte(strconv.Itoa(runtimeType))); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store runtime contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//	m.Log.Infof("upgrade contract begin %+v, %+v, byte code len %d", contractId, parameters, len(byteCode))
	//case commonPb.ManageUserContractFunction_FREEZE_CONTRACT.String():
	//	vt := &verifyType{requireVersion: true}
	//	result, code, _, _, _ := vt.commonVerify(txContext, contractId, contractResult)
	//	if code != commonPb.TxStatusCode_SUCCESS {
	//		return result, code
	//	}
	//
	//	// add freeze target
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), freezeKey, []byte(contractName)); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store freeze target for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//	m.Log.Infof("contract[%s] freeze finish.", contractName)
	//	contractResult.Code = 0
	//	return contractResult, commonPb.TxStatusCode_SUCCESS
	//case commonPb.ManageUserContractFunction_UNFREEZE_CONTRACT.String():
	//	vt := &verifyType{requireVersion: true}
	//	result, code, _, _, _ := vt.commonVerify(txContext, contractId, contractResult)
	//	if code != commonPb.TxStatusCode_SUCCESS {
	//		return result, code
	//	}
	//
	//	// del freeze target
	//	if err := txContext.Del(commonPb.SystemContract_CONTRACT_MANAGE.String(), freezeKey); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store unfreeze target for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//	m.Log.Infof("contract[%s] unfreeze finish.", contractName)
	//	contractResult.Code = 0
	//	return contractResult, commonPb.TxStatusCode_SUCCESS
	//case commonPb.ManageUserContractFunction_REVOKE_CONTRACT.String():
	//	vt := &verifyType{requireVersion: true}
	//	result, code, _, _, _ := vt.commonVerify(txContext, contractId, contractResult)
	//	if code != commonPb.TxStatusCode_SUCCESS {
	//		return result, code
	//	}
	//
	//	// add revoke target
	//	if err := txContext.Put(commonPb.SystemContract_CONTRACT_MANAGE.String(), revokeKey, []byte(contractName)); err != nil {
	//		contractResult.Message = fmt.Sprintf("failed to store revoke target for contract:%s, error:%s", contractName, err.Error())
	//		return contractResult, commonPb.TxStatusCode_PUT_INTO_TX_CONTEXT_FAILED
	//	}
	//	m.Log.Infof("contract[%s] revoke finish.", contractName)
	//	contractResult.Code = 0
	//	return contractResult, commonPb.TxStatusCode_SUCCESS
	//}
	//contractId.Version = version

	return m.invokeUserContractByRuntime(myContract, method, parameters, txContext, byteCode, gasUsed)
}

type verifyType struct {
	requireVersion       bool     // get contract version from TxSimContext, if not exist then return error message
	requireNullVersion   bool     // get contract version from TxSimContext, if exist then return error message
	requireFormatVersion bool     // get contract version from parameter, if format error  then return error message
	requireExcludeMethod bool     // get contract method from parameter, if `currentMethod`in excludeMethodList then return error message
	requireByteCode      bool     // get contract byteCode from parameter, if not exist then return error message
	requireRuntimeType   bool     // get contract runtimeType from TxSimContext, if not exist then return error message
	currentMethod        string   // for requireExcludeMethod
	excludeMethodList    []string // for requireExcludeMethod
}

// commonVerify verify version、method、byteCode、runtimeType, return (result, code, byteCode, version, runtimeType)
func (v *verifyType) commonVerify(txContext protocol.TxSimContext, contractId *commonPb.Contract, contractResult *commonPb.ContractResult) (*commonPb.ContractResult, commonPb.TxStatusCode, []byte, string, int) {
	contractName := contractId.Name
	versionKey := []byte(protocol.ContractVersion + contractName)
	var resultVersion string
	msgPre := "verify fail,"

	if v.requireVersion {
		if versionInContext, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionKey); err != nil {
			contractResult.Message = fmt.Sprintf("%s unable to find latest version for contract[%s], system error:%s", msgPre, contractName, err.Error())
			return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		} else if len(versionInContext) == 0 {
			contractResult.Message = fmt.Sprintf("%s the contract does not exist. contract[%s], please create a contract ", msgPre, contractName)
			return v.errorResult(contractResult, commonPb.TxStatusCode_CONTRACT_VERSION_NOT_EXIST_FAILED, resultVersion)
		} else {
			resultVersion = string(versionInContext)
		}
	}

	if v.requireNullVersion {
		if versionInContext, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionKey); err != nil {
			contractResult.Message = fmt.Sprintf("%s unable to find latest version for contract[%s], system error:%s", msgPre, contractName, err.Error())
			return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		} else if versionInContext != nil {
			contractResult.Message = fmt.Sprintf("%s the contract already exists. contract[%s], version[%s]", msgPre, contractName, string(versionInContext))
			return v.errorResult(contractResult, commonPb.TxStatusCode_CONTRACT_VERSION_EXIST_FAILED, resultVersion)
		}
	}

	if v.requireExcludeMethod {
		for i := range v.excludeMethodList {
			if v.currentMethod == v.excludeMethodList[i] {
				contractResult.Message = fmt.Sprintf("%s contract[%s], method[%s] is not allowed to be called, it's the retention method", msgPre, contractName, v.excludeMethodList[i])
				return v.errorResult(contractResult, commonPb.TxStatusCode_CONTRACT_INVOKE_METHOD_FAILED, resultVersion)
			}
		}
	}

	if v.requireFormatVersion {
		if contractId.Version == "" {
			contractResult.Message = fmt.Sprintf("%s please provide the param[version] of the contract[%s]", msgPre, contractId.Name)
			return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		} else {
			if len(contractId.Version) > protocol.DefaultVersionLen {
				contractResult.Message = fmt.Sprintf("%s param[version] string of the contract[%+v] too long, should be less than %d", msgPre, contractId, protocol.DefaultVersionLen)
				return v.errorResult(contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_VERSION, resultVersion)
			}

			match, err := regexp.MatchString(protocol.DefaultVersionRegex, contractId.Version)
			if err != nil || !match {
				contractResult.Message = fmt.Sprintf("%s param[version] string of the contract[%+v] invalid while invoke user contract, should match [%s]", msgPre, contractId, protocol.DefaultVersionRegex)
				return v.errorResult(contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_VERSION, resultVersion)
			}
		}
	}

	var byteCode []byte
	if v.requireByteCode {
		versionedByteCodeKey := append([]byte(protocol.ContractByteCode+contractName), []byte(resultVersion)...)
		if byteCodeInContext, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionedByteCodeKey); err != nil {
			contractResult.Message = fmt.Sprintf("%s failed to check byte code in tx context for contract[%s], %s", msgPre, contractName, err.Error())
			return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		} else if len(byteCodeInContext) == 0 {
			contractResult.Message = fmt.Sprintf("%s the contract byte code not found from db. contract[%s], please create a contract ", msgPre, contractName)
			return v.errorResult(contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_BYTE_CODE, resultVersion)
		} else {
			byteCode = byteCodeInContext
		}
	}

	runtimeType := 0
	if v.requireRuntimeType {
		runtimeTypeKey := []byte(protocol.ContractRuntimeType + contractName)
		if runtimeTypeBytes, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), runtimeTypeKey); err != nil {
			contractResult.Message = fmt.Sprintf("%s failed to find runtime type %s, system error: %s", msgPre, contractName, err.Error())
			return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		} else if runtimeTypeTmp, err := strconv.Atoi(string(runtimeTypeBytes)); err != nil {
			contractResult.Message = fmt.Sprintf("%s the contract runtime type not found from db. contract[%s], please create a contract ", msgPre, contractName)
			return v.errorResult(contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_RUNTIME_TYPE, resultVersion)
		} else {
			runtimeType = runtimeTypeTmp
		}
	}

	return nil, commonPb.TxStatusCode_SUCCESS, byteCode, resultVersion, runtimeType
}

func (v *verifyType) errorResult(contractResult *commonPb.ContractResult, code commonPb.TxStatusCode, version string) (*commonPb.ContractResult, commonPb.TxStatusCode, []byte, string, int) {
	return contractResult, code, nil, version, 0
}

func (m *ManagerImpl) invokeUserContractByRuntime(contract *commonPb.Contract, method string, parameters map[string][]byte,
	txContext protocol.TxSimContext, byteCode []byte, gasUsed uint64) (*commonPb.ContractResult, commonPb.TxStatusCode) {
	contractResult := &commonPb.ContractResult{Code: 1}
	txId := txContext.GetTx().Payload.TxId
	txType := txContext.GetTx().Payload.TxType
	runtimeType := contract.RuntimeType
	var runtimeInstance RuntimeInstance
	var err error
	switch runtimeType {
	case commonPb.RuntimeType_WASMER:
		runtimeInstance, err = m.WasmerVmPoolManager.NewRuntimeInstance(contract, byteCode)
		if err != nil {
			contractResult.Message = fmt.Sprintf("failed to create vm runtime, contract: %s, %s", contract.Name, err.Error())
			return contractResult, commonPb.TxStatusCode_CREATE_RUNTIME_INSTANCE_FAILED
		}
	case commonPb.RuntimeType_GASM:
		runtimeInstance = &gasm.RuntimeInstance{
			ChainId: m.ChainId,
			Log:     m.Log,
		}
	case commonPb.RuntimeType_WXVM:
		runtimeInstance = &wxvm.RuntimeInstance{
			ChainId:     m.ChainId,
			CodeManager: m.WxvmCodeManager,
			CtxService:  m.WxvmContextService,
			Log:         m.Log,
		}
	case commonPb.RuntimeType_EVM:
		runtimeInstance = &evm.RuntimeInstance{
			Log:          m.Log,
			ChainId:      m.ChainId,
			TxSimContext: txContext,
			Method:       method,
			ContractId:   contract,
		}
	default:
		contractResult.Message = fmt.Sprintf("no such vm runtime %q", runtimeType)
		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_RUNTIME_TYPE
	}

	sender := txContext.GetSender()

	creator := contract.Creator

	if creator == nil {
		contractResult.Message = fmt.Sprintf("creator is empty for contract:%s", contract.Name)
		return contractResult, commonPb.TxStatusCode_GET_CREATOR_FAILED
	}

	sender, code := getFullCertMember(sender, txContext)
	if code != commonPb.TxStatusCode_SUCCESS {
		return contractResult, code
	}

	creator, code = getFullCertMember(creator, txContext)
	if code != commonPb.TxStatusCode_SUCCESS {
		return contractResult, code
	}

	// Get three items in the certificate: orgid PK role
	if senderMember, err := m.AccessControl.NewMemberFromProto(sender); err != nil {
		contractResult.Message = fmt.Sprintf("failed to unmarshal sender %q", runtimeType)
		return contractResult, commonPb.TxStatusCode_UNMARSHAL_SENDER_FAILED
	} else {
		parameters[protocol.ContractSenderOrgIdParam] = []byte(senderMember.GetOrgId())
		parameters[protocol.ContractSenderRoleParam] = []byte(senderMember.GetRole()[0])
		parameters[protocol.ContractSenderPkParam] = senderMember.GetSKI()
	}

	// Get three items in the certificate: orgid PK role
	if creatorMember, err := m.AccessControl.NewMemberFromProto(creator); err != nil {
		contractResult.Message = fmt.Sprintf("failed to unmarshal creator %q", creator)
		return contractResult, commonPb.TxStatusCode_UNMARSHAL_CREATOR_FAILED
	} else {
		parameters[protocol.ContractCreatorOrgIdParam] = []byte( creator.OrgId)
		parameters[protocol.ContractCreatorRoleParam] = []byte(creatorMember.GetRole()[0])
		parameters[protocol.ContractCreatorPkParam] = creatorMember.GetSKI()
	}

	parameters[protocol.ContractTxIdParam] = []byte(txId)
	parameters[protocol.ContractBlockHeightParam] = []byte(strconv.FormatUint(txContext.GetBlockHeight(), 10))

	// calc the gas used by byte code
	// gasUsed := uint64(GasPerByte * len(byteCode))

	m.Log.Debugf("invoke vm, tx id:%s, tx type:%+v, contractId:%+v, method:%+v, runtime type:%+v, byte code len:%+v, params:%+v",
		txId, txType, contract, method, runtimeType, len(byteCode), len(parameters))

	// begin save point for sql
	var dbTransaction protocol.SqlDBTransaction
	if m.ChainConf.ChainConfig().Contract.EnableSqlSupport && txType != commonPb.TxType_QUERY_CONTRACT {
		txKey := commonPb.GetTxKeyWith(txContext.GetBlockProposer().MemberInfo, txContext.GetBlockHeight())
		dbTransaction, err = txContext.GetBlockchainStore().GetDbTransaction(txKey)
		if err != nil {
			contractResult.Message = fmt.Sprintf("get db transaction from [%s] error %+v", txKey, err)
			return contractResult, commonPb.TxStatusCode_INTERNAL_ERROR
		}
		err := dbTransaction.BeginDbSavePoint(txId)
		if err != nil {
			m.Log.Warn("[%s] begin db save point error, %s", txId, err.Error())
		}
		//txContext.Put(contractId.Name, []byte("target"), []byte("mysql")) // for dag
	}

	runtimeContractResult := runtimeInstance.Invoke(contract, method, byteCode, parameters, txContext, gasUsed)
	if runtimeContractResult.Code == 0 {
		return runtimeContractResult, commonPb.TxStatusCode_SUCCESS
	} else {
		if m.ChainConf.ChainConfig().Contract.EnableSqlSupport && txType != commonPb.TxType_QUERY_CONTRACT {
			err := dbTransaction.RollbackDbSavePoint(txId)
			if err != nil {
				m.Log.Warn("[%s] rollback db save point error, %s", txId, err.Error())
			}
		}
		return runtimeContractResult, commonPb.TxStatusCode_CONTRACT_FAIL
	}
}

func getFullCertMember(sender *acPb.SerializedMember, txContext protocol.TxSimContext) (*acPb.SerializedMember, commonPb.TxStatusCode) {
	// If the certificate in the transaction is hash, the original certificate is retrieved
	if sender.MemberType == acPb.MemberType_CERT_HASH {
		memberInfoHex := hex.EncodeToString(sender.MemberInfo)
		var fullCertMemberInfo []byte
		var err error
		if fullCertMemberInfo, err = txContext.Get(commonPb.SystemContract_CERT_MANAGE.String(), []byte(memberInfoHex)); err != nil {
			return nil, commonPb.TxStatusCode_GET_SENDER_CERT_FAILED
		}
		sender = &acPb.SerializedMember{
			OrgId:      sender.OrgId,
			MemberInfo: fullCertMemberInfo,
			MemberType: acPb.MemberType_CERT,
		}
	}
	return sender, commonPb.TxStatusCode_SUCCESS
}

func (m *ManagerImpl) isUserContract(refTxType commonPb.TxType) bool {
	switch refTxType {
	case
		commonPb.TxType_INVOKE_CONTRACT,
		commonPb.TxType_QUERY_CONTRACT:
		return true
	default:
		return false
	}
}
