/*
 Copyright (C) BABEC. All rights reserved.
 Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
   SPDX-License-Identifier: Apache-2.0
*/

package native

import (
	"sync"

	"chainmaker.org/chainmaker-go/vm/native/privatecompute"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker-go/vm/native/blockcontract"
	"chainmaker.org/chainmaker-go/vm/native/certmgr"
	"chainmaker.org/chainmaker-go/vm/native/chainconfigmgr"
	"chainmaker.org/chainmaker-go/vm/native/common"
	"chainmaker.org/chainmaker-go/vm/native/contractmgr"
	"chainmaker.org/chainmaker-go/vm/native/dposmgr"
	"chainmaker.org/chainmaker-go/vm/native/government"
	"chainmaker.org/chainmaker-go/vm/native/multisign"

	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

var (
	nativeLock     = &sync.Mutex{}
	nativeInstance = make(map[string]*RuntimeInstance, 0) // singleton map[chainId]instance
)

type RuntimeInstance struct {
	// contracts map[contractName]Contract
	contracts map[string]common.Contract
	log       protocol.Logger
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

func initContract(log protocol.Logger) map[string]common.Contract {
	contracts := make(map[string]common.Contract, 64)
	contracts[syscontract.SystemContract_CHAIN_CONFIG.String()] = chainconfigmgr.NewChainConfigContract(log)
	contracts[syscontract.SystemContract_CHAIN_QUERY.String()] = blockcontract.NewBlockContact(log)
	contracts[syscontract.SystemContract_CERT_MANAGE.String()] = certmgr.NewCertManageContract(log)
	contracts[syscontract.SystemContract_GOVERNANCE.String()] = government.NewGovernmentContract(log)
	contracts[syscontract.SystemContract_MULTI_SIGN.String()] = multisign.NewMultiSignContract(log)
	contracts[syscontract.SystemContract_PRIVATE_COMPUTE.String()] = privatecompute.NewPrivateComputeContact(log)
	contracts[syscontract.SystemContract_DPOS_ERC20.String()] = dposmgr.NewDPoSERC20Contract(log)
	contracts[syscontract.SystemContract_DPOS_STAKE.String()] = dposmgr.NewDPoSStakeContract(log)
	contracts[syscontract.SystemContract_CONTRACT_MANAGE.String()] = contractmgr.NewContractManager(log)
	return contracts
}

// Invoke verify and run Contract method
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, methodName string, _ []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext) *commonPb.ContractResult {

	result := &commonPb.ContractResult{
		Code:    uint32(1),
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
		r.log.Warn(err)
		result.Message = err.Error()
		return result
	}

	// exec
	bytes, err := f(txContext, parameters)
	if err != nil {
		r.log.Warn(err)
		result.Message = err.Error()
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
	//var config commonPb.Payload
	//err := proto.Unmarshal(payload, &config)
	//if err != nil {
	//	r.log.Errorw(ErrUnmarshalFailed.Error(), "Position", "SystemContractPayload Unmarshal", "err", err)
	//	return ErrUnmarshalFailed
	//}

	// chainId
	//if tx.Payload.ChainId != config.ChainId {
	//	r.log.Errorw("chainId is different", "tx chainId", tx.Payload.ChainId, "payload chainId", config.ChainId)
	//	return errors.New("chainId is different")
	//}

	bytes, err := txContext.Get(syscontract.SystemContract_CHAIN_CONFIG.String(), []byte(syscontract.SystemContract_CHAIN_CONFIG.String()))
	var chainConfig configPb.ChainConfig
	err = proto.Unmarshal(bytes, &chainConfig)
	if err != nil {
		r.log.Errorw(common.ErrUnmarshalFailed.Error(), "Position", "configPb.ChainConfig Unmarshal", "err", err)
		return common.ErrUnmarshalFailed
	}

	if payload.Sequence != chainConfig.Sequence+1 {
		// the sequence is not incre 1
		r.log.Errorw(common.ErrSequence.Error(), "chainConfig", chainConfig.Sequence, "sdk chainConfig", payload.Sequence)
		return common.ErrSequence
	}
	return nil
}

func (r *RuntimeInstance) getContractFunc(contract *commonPb.Contract, methodName string) (common.ContractFunc, error) {
	if contract == nil {
		return nil, common.ErrContractIdIsNil
	}

	contractName := contract.Name
	contractInst := r.contracts[contractName]
	if contractInst == nil {
		return nil, common.ErrContractNotFound
	}

	f := contractInst.GetMethod(methodName)
	if f == nil {
		return nil, common.ErrMethodNotFound
	}
	return f, nil
}
