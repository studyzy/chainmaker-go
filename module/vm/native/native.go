/*
 Copyright (C) BABEC. All rights reserved.
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"sync"

	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
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
	contracts[commonPb.ContractName_SYSTEM_CONTRACT_USER_CONTRACT_MANAGE.String()] = newContractManager(log)
	return contracts
}

// Invoke verify and run Contract method
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, methodName string, _ []byte, parameters map[string]string,
	txContext protocol.TxSimContext) *commonPb.ContractResult {

	result := &commonPb.ContractResult{
		Code:    1,
		Message: "contract internal error",
		Result:  nil,
	}

	//txType := txContext.GetTx().Header.TxType
	//if txType == commonPb.TxType_INVOKE_CONTRACT {
	//	if err := r.verifySequence(txContext); err != nil {
	//		result.Message = fmt.Sprintf(err.Error()+",txType: %s", txType)
	//		return result
	//	}
	//}

	f, err := r.getContractFunc(contract, methodName)
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

	if len(bytes) == 0 {
		result.Message = "not found"
		return result
	}

	result.Code = 0
	result.Message = "OK"
	result.Result = bytes
	return result
}

func (r *RuntimeInstance) verifySequence(txContext protocol.TxSimContext) error {
	tx := txContext.GetTx()
	payload := tx.Payload
	//var config commonPb.SystemContractPayload
	//err := proto.Unmarshal(payload, &config)
	//if err != nil {
	//	r.log.Errorw(ErrUnmarshalFailed.Error(), "Position", "SystemContractPayload Unmarshal", "err", err)
	//	return ErrUnmarshalFailed
	//}

	// chainId
	//if tx.Payload.ChainId != config.ChainId {
	//	r.log.Errorw("chainId is different", "tx chainId", tx.Header.ChainId, "payload chainId", config.ChainId)
	//	return errors.New("chainId is different")
	//}

	bytes, err := txContext.Get(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), []byte(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String()))
	var chainConfig configPb.ChainConfig
	err = proto.Unmarshal(bytes, &chainConfig)
	if err != nil {
		r.log.Errorw(ErrUnmarshalFailed.Error(), "Position", "configPb.ChainConfig Unmarshal", "err", err)
		return ErrUnmarshalFailed
	}

	if payload.Sequence != chainConfig.Sequence+1 {
		// the sequence is not incre 1
		r.log.Errorw(ErrSequence.Error(), "chainConfig", chainConfig.Sequence, "sdk chainConfig", payload.Sequence)
		return ErrSequence
	}
	return nil
}

func (r *RuntimeInstance) getContractFunc(contract *commonPb.Contract, methodName string) (ContractFunc, error) {
	if contract == nil {
		return nil, ErrContractIdIsNil
	}

	contractName := contract.Name
	contractInst := r.contracts[contractName]
	if contractInst == nil {
		return nil, ErrContractNotFound
	}

	f := contractInst.getMethod(methodName)
	if f == nil {
		return nil, ErrMethodNotFound
	}
	return f, nil
}
