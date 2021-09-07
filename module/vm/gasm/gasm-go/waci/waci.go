/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0

waci: WebAssembly Chainmaker Interface
*/

package waci

import (
	"fmt"
	"reflect"

	"chainmaker.org/chainmaker-go/gasm/gasm-go/wasm"
	"chainmaker.org/chainmaker-go/wasi"
	"chainmaker.org/chainmaker/common/v2/serialize"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

const WaciModuleName = "env"

// Wacsi WebAssembly chainmaker system interface
var wacsi = wasi.NewWacsi()

type WaciInstance struct {
	TxSimContext   protocol.TxSimContext
	ContractId     *commonPb.Contract
	ContractResult *commonPb.ContractResult
	Log            *logger.CMLogger
	Vm             *wasm.VirtualMachine
	RequestBody    []byte // sdk request param
	GetStateCache  []byte // cache call method GetStateLen value result, one cache per transaction
	ChainId        string
	Method         string
	ContractEvent  []*commonPb.ContractEvent
}

// LogMessage print log to file
func (s *WaciInstance) LogMsg(vm *wasm.VirtualMachine) reflect.Value {
	return reflect.ValueOf(func(msgPtr int32, msgLen int32) {
		msg := vm.Memory[msgPtr : msgPtr+msgLen]
		s.Log.Debugf("gasm log>> [%s] %s", s.TxSimContext.GetTx().GetPayload().TxId, msg)
	})
}

// LogMessage print log to file
func (s *WaciInstance) LogMessage() int32 {
	s.Log.Debugf("gasm log>> [%s] %s", s.TxSimContext.GetTx().GetPayload().TxId, string(s.RequestBody))
	return protocol.ContractSdkSignalResultSuccess
}

// SysCall wasmer vm call chain entry
func (s *WaciInstance) SysCall(vm *wasm.VirtualMachine) reflect.Value {
	return reflect.ValueOf(func(requestHeaderPtr int32, requestHeaderLen int32, requestBodyPtr int32, requestBodyLen int32) int32 {
		if requestHeaderLen == 0 {
			s.Log.Errorf("gasm log>> [%s] requestHeader is null.", s.TxSimContext.GetTx().GetPayload().TxId)
			return protocol.ContractSdkSignalResultFail
		}

		// get param from memory
		requestHeaderByte := make([]byte, requestHeaderLen)
		copy(requestHeaderByte, vm.Memory[requestHeaderPtr:requestHeaderPtr+requestHeaderLen])
		requestBody := make([]byte, requestBodyLen)
		copy(requestBody, vm.Memory[requestBodyPtr:requestBodyPtr+requestBodyLen])

		ec := serialize.NewEasyCodecWithBytes(requestHeaderByte)

		s.Vm = vm
		s.RequestBody = requestBody
		method, err := ec.GetValue("method", serialize.EasyKeyType_SYSTEM)
		if err != nil {
			msg := fmt.Sprintf("get method failed:%s requestHeader=%s requestBody=%s", "request header have no method", string(requestHeaderByte), string(requestBody))
			return s.recordMsg(msg)
		}
		switch method {
		// common
		case protocol.ContractMethodLogMessage:
			return s.LogMessage()
		case protocol.ContractMethodSuccessResult:
			return s.SuccessResult()
		case protocol.ContractMethodErrorResult:
			return s.ErrorResult()
		case protocol.ContractMethodCallContract:
			return s.CallContract()
		case protocol.ContractMethodCallContractLen:
			return s.CallContractLen()
		case protocol.ContractMethodEmitEvent:
			return s.EmitEvent()
		// paillier
		case protocol.ContractMethodGetPaillierOperationResultLen:
			return s.GetPaillierResultLen()
		case protocol.ContractMethodGetPaillierOperationResult:
			return s.GetPaillierResult()
		// bulletproofs
		case protocol.ContractMethodGetBulletproofsResultLen:
			return s.GetBulletProofsResultLen()
		case protocol.ContractMethodGetBulletproofsResult:
			return s.GetBulletProofsResult()
		// kv
		case protocol.ContractMethodGetStateLen:
			return s.GetStateLen()
		case protocol.ContractMethodGetState:
			return s.GetState()
		case protocol.ContractMethodPutState:
			return s.PutState()
		case protocol.ContractMethodDeleteState:
			return s.DeleteState()
		//kv author:whang1234
		case protocol.ContractMethodKvIterator:
			return s.KvIterator()
		case protocol.ContractMethodKvPreIterator:
			return s.KvPreIterator()
		case protocol.ContractMethodKvIteratorHasNext:
			return s.KvIteratorHasNext()
		case protocol.ContractMethodKvIteratorNextLen:
			return s.KvIteratorNextLen()
		case protocol.ContractMethodKvIteratorNext:
			return s.KvIteratorNext()
		case protocol.ContractMethodKvIteratorClose:
			return s.KvIteratorClose()
		//sql
		case protocol.ContractMethodExecuteUpdate:
			return s.ExecuteUpdate()
		case protocol.ContractMethodExecuteDdl:
			return s.ExecuteDDL()
		case protocol.ContractMethodExecuteQueryOneLen:
			return s.ExecuteQueryOneLen()
		case protocol.ContractMethodExecuteQueryOne:
			return s.ExecuteQueryOne()
		case protocol.ContractMethodExecuteQuery:
			return s.ExecuteQuery()
		case protocol.ContractMethodRSHasNext:
			return s.RSHasNext()
		case protocol.ContractMethodRSNextLen:
			return s.RSNextLen()
		case protocol.ContractMethodRSNext:
			return s.RSNext()
		case protocol.ContractMethodRSClose:
			return s.RSClose()
		default:
			s.Log.Errorf("method is %s not match.", method)
		}
		return protocol.ContractSdkSignalResultFail
	})
}

// GetBulletProofsResultLen get bulletproofs operation result length from chain
func (s *WaciInstance) GetBulletProofsResultLen() int32 {
	return s.getBulletProofsResultCore(true)
}

// GetBulletProofsResult get bulletproofs operation result from chain
func (s *WaciInstance) GetBulletProofsResult() int32 {
	return s.getBulletProofsResultCore(false)
}

func (s *WaciInstance) getBulletProofsResultCore(isLen bool) int32 {
	data, err := wacsi.BulletProofsOperation(s.RequestBody, s.Vm.Memory, s.GetStateCache, isLen)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// EmitEvent emit event to chain
func (s *WaciInstance) EmitEvent() int32 {
	contractEvent, err := wacsi.EmitEvent(s.RequestBody, s.TxSimContext, s.ContractId, s.Log)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	s.ContractEvent = append(s.ContractEvent, contractEvent)
	return protocol.ContractSdkSignalResultSuccess
}

// GetPaillierResultLen get paillier operation result length from chain
func (s *WaciInstance) GetPaillierResultLen() int32 {
	return s.getPaillierResultCore(true)
}

// GetPaillierResult get paillier operation result from chain
func (s *WaciInstance) GetPaillierResult() int32 {
	return s.getPaillierResultCore(false)
}

func (s *WaciInstance) getPaillierResultCore(isLen bool) int32 {
	data, err := wacsi.PaillierOperation(s.RequestBody, s.Vm.Memory, s.GetStateCache, isLen)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess

}

// SuccessResult record the results of contract execution success
func (s *WaciInstance) SuccessResult() int32 {
	return wacsi.SuccessResult(s.ContractResult, s.RequestBody)
}

// ErrorResult record the results of contract execution error
func (s *WaciInstance) ErrorResult() int32 {
	return wacsi.ErrorResult(s.ContractResult, s.RequestBody)
}

//  CallContractLen invoke cross contract calls, save result to cache and putout result length
func (s *WaciInstance) CallContractLen() int32 {
	return s.callContractCore(true)
}

//  CallContractLen get cross contract call result from cache
func (s *WaciInstance) CallContract() int32 {
	return s.callContractCore(false)
}

func (s *WaciInstance) callContractCore(isLen bool) int32 {
	data, gas, err := wacsi.CallContract(s.RequestBody, s.TxSimContext, s.Vm.Memory, s.GetStateCache, s.Vm.Gas, isLen)
	s.GetStateCache = data // reset data
	s.Vm.Gas = gas
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

func (s *WaciInstance) recordMsg(msg string) int32 {
	if len(s.ContractResult.Message) > 0 {
		s.ContractResult.Message += ". error message: " + msg
	} else {
		s.ContractResult.Message += "error message: " + msg
	}
	s.ContractResult.Code = 1
	s.Log.Errorf("gasm log>> [%s] %s", s.TxSimContext.GetTx().GetPayload().TxId, s.ContractId.Name, msg)
	return protocol.ContractSdkSignalResultFail
}
