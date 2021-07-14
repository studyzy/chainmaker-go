/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

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
func (f *Factory) NewVmManager(wxvmCodePathPrefix string, accessControl protocol.AccessControlProvider,
	chainNodesInfoProvider protocol.ChainNodesInfoProvider, chainConf protocol.ChainConf) protocol.VmManager {

	chainId := chainConf.ChainConfig().ChainId
	log := logger.GetLoggerByChain(logger.MODULE_VM, chainId)

	wxvmCodeDir := filepath.Join(wxvmCodePathPrefix, chainId, WxvmCodeFolder)
	log.Infof("init wxvm code dir %s", wxvmCodeDir)
	wasmerVmPoolManager := wasmer.NewVmPoolManager(chainId)
	wxvmCodeManager := xvm.NewCodeManager(chainId, wxvmCodeDir)
	wxvmContextService := xvm.NewContextService(chainId)

	return &VmManagerImpl{
		ChainId:                chainId,
		WasmerVmPoolManager:    wasmerVmPoolManager,
		WxvmCodeManager:        wxvmCodeManager,
		WxvmContextService:     wxvmContextService,
		AccessControl:          accessControl,
		ChainNodesInfoProvider: chainNodesInfoProvider,
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

type VmManagerImpl struct {
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

func (m *VmManagerImpl) GetAccessControl() protocol.AccessControlProvider {
	return m.AccessControl
}

func (m *VmManagerImpl) GetChainNodesInfoProvider() protocol.ChainNodesInfoProvider {
	return m.ChainNodesInfoProvider
}

func (m *VmManagerImpl) RunContract(contract *commonPb.Contract, method string, byteCode []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {

	contractResult := &commonPb.ContractResult{
		Code:    uint32(protocol.ContractResultCode_FAIL),
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

	if contract.Status == commonPb.ContractStatus_FROZEN {
		contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been frozen.", contractName)
		return contractResult, commonPb.TxStatusCode_CONTRACT_FREEZE_FAILED
	}
	if contract.Status == commonPb.ContractStatus_REVOKED {
		contractResult.Message = fmt.Sprintf("failed to run user contract, %s has been revoked.", contractName)
		return contractResult, commonPb.TxStatusCode_CONTRACT_REVOKE_FAILED
	}

	if native.IsNative(contractName, refTxType) {
		if method == "" {
			contractResult.Message = "require param method not found."
			return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_METHOD
		}
		return m.runNativeContract(contract, method, parameters, txContext)
	}
	if !m.isUserContract(refTxType) {
		contractResult.Message = fmt.Sprintf("bad contract call %s, transaction type %s", contractName, refTxType)
		return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_TRANSACTION_TYPE
	}
	// byteCode should have value
	if len(byteCode) == 0 {
		contractResult.Message = fmt.Sprintf("contract %s has no byte code, transaction type %s", contractName, refTxType)
		return contractResult, commonPb.TxStatusCode_CONTRACT_BYTECODE_NOT_EXIST_FAILED
	}

	return m.runUserContract(contract, method, byteCode, parameters, txContext, gasUsed)
}

// runNativeContract invoke native contract
func (m *VmManagerImpl) runNativeContract(contract *commonPb.Contract, method string, parameters map[string][]byte,
	txContext protocol.TxSimContext) (*commonPb.ContractResult, commonPb.TxStatusCode) {

	runtimeInstance := native.GetRuntimeInstance(m.ChainId)
	runtimeContractResult := runtimeInstance.Invoke(contract, method, nil, parameters, txContext)

	if runtimeContractResult.Code == uint32(protocol.ContractResultCode_OK) {
		return runtimeContractResult, commonPb.TxStatusCode_SUCCESS
	}
	return runtimeContractResult, commonPb.TxStatusCode_CONTRACT_FAIL
}

// runUserContract invoke user contract
func (m *VmManagerImpl) runUserContract(contract *commonPb.Contract, method string, byteCode []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult, code commonPb.TxStatusCode) {

	var (
		myContract   = contract
		contractName = contract.Name
		status       = contract.Status
	)
	contractResult = &commonPb.ContractResult{Code: uint32(protocol.ContractResultCode_FAIL)}
	if status == commonPb.ContractStatus_ALL {
		dbContract, err := utils.GetContractByName(txContext.Get, contractName)
		if err != nil {
			return nil, commonPb.TxStatusCode_CONTRACT_FAIL
		}
		myContract = dbContract
	}

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
	//versionKey := []byte(protocol.ContractVersion + contractName)
	var resultVersion string
	msgPre := "verify fail,"

	if v.requireVersion {
		//if versionInContext, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionKey); err != nil {
		//	contractResult.Message = fmt.Sprintf("%s unable to find latest version for contract[%s], system error:%s", msgPre, contractName, err.Error())
		//	return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		//} else if len(versionInContext) == 0 {
		//	contractResult.Message = fmt.Sprintf("%s the contract does not exist. contract[%s], please create a contract ", msgPre, contractName)
		//	return v.errorResult(contractResult, commonPb.TxStatusCode_CONTRACT_VERSION_NOT_EXIST_FAILED, resultVersion)
		//} else {
		//	resultVersion = string(versionInContext)
		//}
		resultVersion = contractId.Version
	}

	if v.requireNullVersion {
		//if versionInContext, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), versionKey); err != nil {
		//	contractResult.Message = fmt.Sprintf("%s unable to find latest version for contract[%s], system error:%s", msgPre, contractName, err.Error())
		//	return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		//} else if versionInContext != nil {
		//	contractResult.Message = fmt.Sprintf("%s the contract already exists. contract[%s], version[%s]", msgPre, contractName, string(versionInContext))
		//	return v.errorResult(contractResult, commonPb.TxStatusCode_CONTRACT_VERSION_EXIST_FAILED, resultVersion)
		//}
		//TODO:不知道检查啥
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
		//versionedByteCodeKey := append([]byte(protocol.ContractByteCode+contractName), []byte(resultVersion)...)
		if byteCodeInContext, err := utils.GetContractBytecode(txContext.Get, contractName); err != nil {
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
		//runtimeTypeKey := []byte(protocol.ContractRuntimeType + contractName)
		//if runtimeTypeBytes, err := txContext.Get(commonPb.SystemContract_CONTRACT_MANAGE.String(), runtimeTypeKey); err != nil {
		//	contractResult.Message = fmt.Sprintf("%s failed to find runtime type %s, system error: %s", msgPre, contractName, err.Error())
		//	return v.errorResult(contractResult, commonPb.TxStatusCode_GET_FROM_TX_CONTEXT_FAILED, resultVersion)
		//} else if runtimeTypeTmp, err := strconv.Atoi(string(runtimeTypeBytes)); err != nil {
		//	contractResult.Message = fmt.Sprintf("%s the contract runtime type not found from db. contract[%s], please create a contract ", msgPre, contractName)
		//	return v.errorResult(contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_RUNTIME_TYPE, resultVersion)
		//} else {
		//	runtimeType = runtimeTypeTmp
		//}
		runtimeType = int(contractId.RuntimeType)
	}

	return nil, commonPb.TxStatusCode_SUCCESS, byteCode, resultVersion, runtimeType
}

func (v *verifyType) errorResult(contractResult *commonPb.ContractResult, code commonPb.TxStatusCode, version string) (*commonPb.ContractResult, commonPb.TxStatusCode, []byte, string, int) {
	return contractResult, code, nil, version, 0
}

func (m *VmManagerImpl) invokeUserContractByRuntime(contract *commonPb.Contract, method string, parameters map[string][]byte,
	txContext protocol.TxSimContext, byteCode []byte, gasUsed uint64) (*commonPb.ContractResult, commonPb.TxStatusCode) {
	contractResult := &commonPb.ContractResult{Code: uint32(protocol.ContractResultCode_FAIL)}
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
			Contract:     contract,
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

func getFullCertMember(sender *acPb.Member, txContext protocol.TxSimContext) (*acPb.Member, commonPb.TxStatusCode) {
	// If the certificate in the transaction is hash, the original certificate is retrieved
	if sender.MemberType == acPb.MemberType_CERT_HASH {
		memberInfoHex := hex.EncodeToString(sender.MemberInfo)
		var fullCertMemberInfo []byte
		var err error
		if fullCertMemberInfo, err = txContext.Get(commonPb.SystemContract_CERT_MANAGE.String(), []byte(memberInfoHex)); err != nil {
			return nil, commonPb.TxStatusCode_GET_SENDER_CERT_FAILED
		}
		sender = &acPb.Member{
			OrgId:      sender.OrgId,
			MemberInfo: fullCertMemberInfo,
			MemberType: acPb.MemberType_CERT,
		}
	}
	return sender, commonPb.TxStatusCode_SUCCESS
}

func (m *VmManagerImpl) isUserContract(refTxType commonPb.TxType) bool {
	switch refTxType {
	case
		commonPb.TxType_INVOKE_CONTRACT,
		commonPb.TxType_QUERY_CONTRACT:
		return true
	default:
		return false
	}
}
