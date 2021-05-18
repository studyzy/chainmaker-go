/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0

waci: WebAssembly Chainmaker Interface
*/

package waci

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/gasm/gasm-go/wasm"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/wasi"
	"fmt"
	"reflect"
)

const WaciModuleName = "env"

// Wacsi WebAssembly chainmaker system interface
var wacsi = wasi.NewWacsi()

type WaciInstance struct {
	TxSimContext   protocol.TxSimContext
	ContractId     *commonPb.ContractId
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
		s.Log.Debugf("waci log>> [%s] %s", s.TxSimContext.GetTx().Header.TxId, msg)
	})
}

// LogMessage print log to file
func (s *WaciInstance) LogMessage() int32 {
	s.Log.Debugf("waci log>> [%s] %s", s.TxSimContext.GetTx().Header.TxId, string(s.RequestBody))
	return protocol.ContractSdkSignalResultSuccess
}

// SysCall wasmer vm call chain entry
func (s *WaciInstance) SysCall(vm *wasm.VirtualMachine) reflect.Value {
	return reflect.ValueOf(func(requestHeaderPtr int32, requestHeaderLen int32, requestBodyPtr int32, requestBodyLen int32) int32 {
		if requestHeaderLen == 0 {
			s.Log.Errorf("waci log>>[%s] requestHeader is null.", s.TxSimContext.GetTx().Header.TxId)
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
		// kv
		case protocol.ContractMethodGetStateLen:
			return s.GetStateLen()
		case protocol.ContractMethodGetState:
			return s.GetState()
		case protocol.ContractMethodPutState:
			return s.PutState()
		case protocol.ContractMethodDeleteState:
			return s.DeleteState()
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
		case protocol.ContractMethodGetPaillierOperationResultLen:
			return s.GetPaillierOpResultLen()
		case protocol.ContractMethodGetPaillierOperationResult:
			return s.GetPaillierOpResult()
		default:
			s.Log.Errorf("method is %s not match.", method)
		}
		return protocol.ContractSdkSignalResultFail
	})
}

// GetPaillierOpResultLen get result length
func (s *WaciInstance) GetPaillierOpResultLen() int32 {
	data, err := wacsi.PaillierOperation(s.RequestBody, s.Vm.Memory, s.GetStateCache)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// GetPaillierOpResult get result
func (s *WaciInstance) GetPaillierOpResult() int32 {
	return s.GetPaillierOpResultLen()
}

// GetStateLen get state length from chain
func (s *WaciInstance) GetStateLen() int32 {
	data, err := wacsi.GetState(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory, s.GetStateCache)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// GetStateLen get state from chain
func (s *WaciInstance) GetState() int32 {
	return s.GetStateLen()
}

//func (s *WaciInstance) getStateCore(isGetLen bool) int32 {
//	req := serialize.easyUnmarshal(s.RequestBody)
//	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
//	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
//	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
//	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
//		return s.recordMsg(err.Error())
//	}
//
//	valuePtrInt := int(valuePtr.(int32))
//
//	if isGetLen {
//		contractName := s.ContractId.ContractName
//		value, err := s.TxSimContext.Get(contractName, protocol.GetKeyStr(key.(string), field.(string)))
//		if err != nil {
//			msg := fmt.Sprintf("method getStateCore get fail. key=%s, field=%s, error:%s", key.(string), field.(string), err.Error())
//			return s.recordMsg(msg)
//		}
//		copy(s.Vm.Memory[valuePtrInt:valuePtrInt+4], utils.IntToBytes(int32(len(value))))
//		s.GetStateCache = value
//	} else {
//		len := len(s.GetStateCache)
//		if len != 0 {
//			copy(s.Vm.Memory[valuePtrInt:valuePtrInt+len], s.GetStateCache)
//			s.GetStateCache = nil
//		}
//	}
//	return protocol.ContractSdkSignalResultSuccess
//}

// PutState put state to chain
func (s *WaciInstance) PutState() int32 {
	err := wacsi.PutState(s.RequestBody, s.ContractId.ContractName, s.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState delete state from chain
func (s *WaciInstance) DeleteState() int32 {
	err := wacsi.DeleteState(s.RequestBody, s.ContractId.ContractName, s.TxSimContext)
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
	data, err, gas := wacsi.CallContract(s.RequestBody, s.TxSimContext, s.Vm.Memory, s.GetStateCache, s.Vm.Gas)
	s.GetStateCache = data // reset data
	s.Vm.Gas = gas
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

//  CallContractLen get cross contract call result from cache
func (s *WaciInstance) CallContract() int32 {
	return s.CallContractLen()
}

//
//func (s *WaciInstance) callContractCore(isGetLen bool) int32 {
//	ec := serialize.NewEasyCodecWithBytes(s.RequestBody)
//	valuePtr, _ := ec.GetInt32("value_ptr")
//	contractName, _ := ec.GetString("contract_name")
//	method, _ := ec.GetString("method")
//	param, _ := ec.GetBytes("param")
//
//	paramItem := serialize.easyUnmarshal(param)
//	valuePtrInt := int(valuePtr)
//
//	if !isGetLen { // get value from cache
//		result := s.TxSimContext.GetCurrentResult()
//		copy(s.Vm.Memory[valuePtrInt:valuePtrInt+len(result)], result)
//		return protocol.ContractSdkSignalResultSuccess
//	}
//
//	// check param
//	if len(contractName) == 0 {
//		return s.recordMsg("CallContract contractName is null")
//	}
//	if len(method) == 0 {
//		return s.recordMsg("CallContract method is null")
//	}
//	if len(paramItem) > protocol.ParametersKeyMaxCount {
//		return s.recordMsg("expect less than 20 parameters, but get " + strconv.Itoa(len(paramItem)))
//	}
//	for _, item := range paramItem {
//		if len(item.Key) > protocol.DefaultStateLen {
//			msg := fmt.Sprintf("CallContract param expect Key length less than %d, but get %d", protocol.DefaultStateLen, len(item.Key))
//			return s.recordMsg(msg)
//		}
//		match, err := regexp.MatchString(protocol.DefaultStateRegex, item.Key)
//		if err != nil || !match {
//			msg := fmt.Sprintf("CallContract param expect Key no special characters, but get %s. letter, number, dot and underline are allowed", item.Key)
//			return s.recordMsg(msg)
//		}
//		if len(item.Value.(string)) > protocol.ParametersValueMaxLength {
//			msg := fmt.Sprintf("expect Value length less than %d, but get %d", protocol.ParametersValueMaxLength, len(item.Value.(string)))
//			return s.recordMsg(msg)
//		}
//	}
//	if err := protocol.CheckKeyFieldStr(contractName, method); err != nil {
//		return s.recordMsg(err.Error())
//	}
//
//	// call contract
//	s.Vm.Gas = s.Vm.Gas + protocol.CallContractGasOnce
//	paramMap := serialize.easyCodecItemToParamsMap(paramItem)
//	result, code := s.TxSimContext.CallContract(&commonPb.ContractId{ContractName: contractName}, method, nil, paramMap, s.Vm.Gas, commonPb.TxType_INVOKE_USER_CONTRACT)
//	s.Vm.Gas = s.Vm.Gas + uint64(result.GasUsed)
//	if code != commonPb.TxStatusCode_SUCCESS {
//		msg := fmt.Sprintf("CallContract %s, msg:%s", code.String(), result.Message)
//		return s.recordMsg(msg)
//	}
//	// set value length to memory
//	l := utils.IntToBytes(int32(len(result.Result)))
//	copy(s.Vm.Memory[valuePtrInt:valuePtrInt+4], l)
//	return protocol.ContractSdkSignalResultSuccess
//}

func (s *WaciInstance) recordMsg(msg string) int32 {
	s.ContractResult.Message += msg
	s.ContractResult.Code = commonPb.ContractResultCode_FAIL
	s.Log.Errorf("gasm log>> [%s] %s", s.ContractId.ContractName, msg)
	return protocol.ContractSdkSignalResultFail
}
