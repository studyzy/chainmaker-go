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
	"strconv"

	"chainmaker.org/chainmaker/pb-go/syscontract"

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

// RuntimeInstance is the interface of smart contract engine runtime
type RuntimeInstance interface {
	// Invoke starts a vm runtime and call the “method”
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

func (m *VmManagerImpl) RunContract(contract *commonPb.Contract, method string, byteCode []byte,
	parameters map[string][]byte, txContext protocol.TxSimContext, gasUsed uint64, refTxType commonPb.TxType) (
	*commonPb.ContractResult, commonPb.TxStatusCode) {

	contractResult := &commonPb.ContractResult{
		Code:    uint32(1),
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

	if len(contract.Version) == 0 {
		var err error
		contract, err = utils.GetContractByName(txContext.Get, contractName)
		if err != nil {
			contractResult.Message = fmt.Sprintf("query contract[%s] error", contractName)
			return contractResult, commonPb.TxStatusCode_INVALID_CONTRACT_PARAMETER_CONTRACT_NAME
		}
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
		m.Log.Error(contractResult.Message)
		return contractResult, commonPb.TxStatusCode_CONTRACT_BYTE_CODE_NOT_EXIST_FAILED
	}

	return m.runUserContract(contract, method, byteCode, parameters, txContext, gasUsed)
}

// runNativeContract invoke native contract
func (m *VmManagerImpl) runNativeContract(contract *commonPb.Contract, method string, parameters map[string][]byte,
	txContext protocol.TxSimContext) (*commonPb.ContractResult, commonPb.TxStatusCode) {

	runtimeInstance := native.GetRuntimeInstance(m.ChainId)
	runtimeContractResult := runtimeInstance.Invoke(contract, method, nil, parameters, txContext)

	if runtimeContractResult.Code == uint32(0) {
		return runtimeContractResult, commonPb.TxStatusCode_SUCCESS
	}
	return runtimeContractResult, commonPb.TxStatusCode_CONTRACT_FAIL
}

// runUserContract invoke user contract
func (m *VmManagerImpl) runUserContract(contract *commonPb.Contract, method string, byteCode []byte,
	parameters map[string][]byte, txContext protocol.TxSimContext, gasUsed uint64) (
	contractResult *commonPb.ContractResult, code commonPb.TxStatusCode) {

	return m.invokeUserContractByRuntime(contract, method, parameters, txContext, byteCode, gasUsed)
}

func (m *VmManagerImpl) invokeUserContractByRuntime(contract *commonPb.Contract, method string,
	parameters map[string][]byte, txContext protocol.TxSimContext, byteCode []byte,
	gasUsed uint64) (*commonPb.ContractResult, commonPb.TxStatusCode) {
	contractResult := &commonPb.ContractResult{Code: uint32(1)}
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
	senderMember, err := m.AccessControl.NewMemberFromProto(sender)
	if err != nil {
		contractResult.Message = fmt.Sprintf("failed to unmarshal sender %q", runtimeType)
		return contractResult, commonPb.TxStatusCode_UNMARSHAL_SENDER_FAILED
	}

	parameters[protocol.ContractSenderOrgIdParam] = []byte(senderMember.GetOrgId())
	parameters[protocol.ContractSenderRoleParam] = []byte(senderMember.GetRole()[0])
	parameters[protocol.ContractSenderPkParam] = []byte(hex.EncodeToString(senderMember.GetSKI()))

	// Get three items in the certificate: orgid PK role
	creatorMember, err := m.AccessControl.NewMemberFromProto(creator)
	if err != nil {
		contractResult.Message = fmt.Sprintf("failed to unmarshal creator %q", creator)
		return contractResult, commonPb.TxStatusCode_UNMARSHAL_CREATOR_FAILED
	}

	parameters[protocol.ContractCreatorOrgIdParam] = []byte(creator.OrgId)
	parameters[protocol.ContractCreatorRoleParam] = []byte(creatorMember.GetRole()[0])
	parameters[protocol.ContractCreatorPkParam] = []byte(hex.EncodeToString(creatorMember.GetSKI()))
	parameters[protocol.ContractTxIdParam] = []byte(txId)
	parameters[protocol.ContractBlockHeightParam] = []byte(strconv.FormatUint(txContext.GetBlockHeight(), 10))

	// calc the gas used by byte code
	// gasUsed := uint64(GasPerByte * len(byteCode))

	m.Log.Debugf("invoke vm, tx id:%s, tx type:%+v, contractId:%+v, method:%+v, runtime type:%+v, "+
		"byte code len:%+v, params:%+v", txId, txType, contract, method, runtimeType, len(byteCode), len(parameters))

	// begin save point for sql
	var dbTransaction protocol.SqlDBTransaction
	if m.ChainConf.ChainConfig().Contract.EnableSqlSupport && txType != commonPb.TxType_QUERY_CONTRACT {
		txKey := commonPb.GetTxKeyWith(txContext.GetBlockProposer().MemberInfo, txContext.GetBlockHeight())
		dbTransaction, err = txContext.GetBlockchainStore().GetDbTransaction(txKey)
		if err != nil {
			contractResult.Message = fmt.Sprintf("get db transaction from [%s] error %+v", txKey, err)
			return contractResult, commonPb.TxStatusCode_INTERNAL_ERROR
		}
		err = dbTransaction.BeginDbSavePoint(txId)
		if err != nil {
			m.Log.Warn("[%s] begin db save point error, %s", txId, err.Error())
		}
		//txContext.Put(contractId.Name, []byte("target"), []byte("mysql")) // for dag
	}

	runtimeContractResult := runtimeInstance.Invoke(contract, method, byteCode, parameters, txContext, gasUsed)
	if runtimeContractResult.Code == 0 {
		return runtimeContractResult, commonPb.TxStatusCode_SUCCESS
	}

	if m.ChainConf.ChainConfig().Contract.EnableSqlSupport && txType != commonPb.TxType_QUERY_CONTRACT {
		err = dbTransaction.RollbackDbSavePoint(txId)
		if err != nil {
			m.Log.Warn("[%s] rollback db save point error, %s", txId, err.Error())
		}
	}
	return runtimeContractResult, commonPb.TxStatusCode_CONTRACT_FAIL
}

func getFullCertMember(sender *acPb.Member, txContext protocol.TxSimContext) (*acPb.Member, commonPb.TxStatusCode) {
	// If the certificate in the transaction is hash, the original certificate is retrieved
	if sender.MemberType == acPb.MemberType_CERT_HASH {
		memberInfoHex := hex.EncodeToString(sender.MemberInfo)
		var fullCertMemberInfo []byte
		var err error
		if fullCertMemberInfo, err = txContext.Get(syscontract.SystemContract_CERT_MANAGE.String(),
			[]byte(memberInfoHex)); err != nil {
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
