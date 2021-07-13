/*
 Copyright (C) BABEC. All rights reserved.
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/protocol"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"sync"
)

var (
	nativeLock     = &sync.Mutex{}
	nativeInstance = make(map[string]*RuntimeInstance, 0) // singleton map[chainId]instance
)

type RuntimeInstance struct {
	// contracts map[contractName]Contract
	contracts map[string]Contract
	log       *logger.CMLogger
}

// GetRuntimeInstance get singleton RuntimeInstance
func GetRuntimeInstance(chainId string) *RuntimeInstance {
	instance, ok := nativeInstance[chainId]
	if !ok {
		nativeLock.Lock()
		defer nativeLock.Unlock()
		instance, ok = nativeInstance[chainId]
		if !ok {
			log := logger.GetLoggerByChain(logger.MODULE_VM, chainId)
			instance = &RuntimeInstance{
				log:       log,
				contracts: initContract(log),
			}
			nativeInstance[chainId] = instance
		}
	}
	return instance
}

func initContract(log *logger.CMLogger) map[string]Contract {
	contracts := make(map[string]Contract, 64)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()] = newChainConfigContract(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String()] = newBlockContact(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String()] = newCertManageContract(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_GOVERNANCE.String()] = newGovernmentContract(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_MULTI_SIGN.String()] = newMultiSignContract(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_PRIVATE_COMPUTE.String()] = newPrivateComputeContact(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_DPOS_ERC20.String()] = newDPoSERC20Contract(log)
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_DPOS_STAKE.String()] = newDPoSStakeContract(log)
	return contracts
}

// Invoke verify and run Contract method
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, methodName string, _ []byte, parameters map[string]string,
	txContext protocol.TxSimContext) *commonPb.ContractResult {

	result := &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_FAIL,
		Message: "",
		Result:  nil,
	}

	txType := txContext.GetTx().Header.TxType
	if txType == commonPb.TxType_UPDATE_CHAIN_CONFIG {
		if err := r.verifySequence(txContext); err != nil {
			result.Message = fmt.Sprintf(err.Error()+",txType: %s", txType)
			return result
		}
	}

	f, err := r.getContractFunc(contractId, methodName)
	if err != nil {
		r.log.Error(err)
		result.Message = err.Error()
		return result
	}

	// exec
	bytes, err := f(txContext, parameters)
	if err != nil {
		r.log.Error(err)
		result.Message = err.Error()
		return result
	}
	result.Code = commonPb.ContractResultCode_OK
	result.Message = commonPb.ContractResultCode_OK.String()
	result.Result = bytes
	return result
}

func (r *RuntimeInstance) verifySequence(txContext protocol.TxSimContext) error {
	tx := txContext.GetTx()
	payload := tx.RequestPayload
	var config commonPb.SystemContractPayload
	err := proto.Unmarshal(payload, &config)
	if err != nil {
		r.log.Errorw(ErrUnmarshalFailed.Error(), "Position", "SystemContractPayload Unmarshal", "err", err)
		return ErrUnmarshalFailed
	}

	// chainId
	if tx.Header.ChainId != config.ChainId {
		r.log.Errorw("chainId is different", "tx chainId", tx.Header.ChainId, "payload chainId", config.ChainId)
		return errors.New("chainId is different")
	}

	bytes, err := txContext.Get(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), []byte(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()))
	var chainConfig configPb.ChainConfig
	err = proto.Unmarshal(bytes, &chainConfig)
	if err != nil {
		r.log.Errorw(ErrUnmarshalFailed.Error(), "Position", "configPb.ChainConfig Unmarshal", "err", err)
		return ErrUnmarshalFailed
	}

	if config.Sequence != chainConfig.Sequence+1 {
		// the sequence is not incre 1
		r.log.Errorw(ErrSequence.Error(), "chainConfig", chainConfig.Sequence, "sdk chainConfig", config.Sequence)
		return ErrSequence
	}
	return nil
}

func (r *RuntimeInstance) getContractFunc(contractId *commonPb.ContractId, methodName string) (ContractFunc, error) {
	if contractId == nil {
		return nil, ErrContractIdIsNil
	}

	contractName := contractId.ContractName
	contract := r.contracts[contractName]
	if contract == nil {
		return nil, ErrContractNotFound
	}

	f := contract.getMethod(methodName)
	if f == nil {
		return nil, ErrMethodNotFound
	}
	return f, nil
}
