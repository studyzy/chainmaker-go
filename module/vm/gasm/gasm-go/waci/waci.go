/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package waci

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/gasm/gasm-go/wasm"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
)

const WaciModuleName = "env"

type WaciInstance struct {
	TxSimContext   protocol.TxSimContext
	ContractId     *commonPb.ContractId
	ContractResult *commonPb.ContractResult
	Log            *logger.CMLogger
	Vm             *wasm.VirtualMachine
	RequestHeader  []*serialize.EasyCodecItem
	RequestBody    []byte // sdk request param
	GetStateCache  []byte // cache call method GetStateLen value result
	ContractEvent  []*commonPb.ContractEvent
}

// LogMessage print log to file
func (w *WaciInstance) LogMessage(vm *wasm.VirtualMachine) reflect.Value {
	return reflect.ValueOf(func(msgPtr int32, msgLen int32) {
		msg := vm.Memory[msgPtr : msgPtr+msgLen]
		w.Log.Debugf("waci log>> [%s] %s", w.TxSimContext.GetTx().Header.TxId, msg)
	})
}

// LogMessage print log to file
func (w *WaciInstance) LogMsg() int32 {
	w.Log.Debugf("waci log>> [%s] %s", w.TxSimContext.GetTx().Header.TxId, string(w.RequestBody))
	return protocol.ContractSdkSignalResultSuccess
}

// SysCall wasmer vm call chain entry
func (w *WaciInstance) SysCall(vm *wasm.VirtualMachine) reflect.Value {
	return reflect.ValueOf(func(requestHeaderPtr int32, requestHeaderLen int32, requestBodyPtr int32, requestBodyLen int32) int32 {
		if requestHeaderLen == 0 {
			w.Log.Errorf("waci log>>[%s] requestHeader is null.", w.TxSimContext.GetTx().Header.TxId)
			return protocol.ContractSdkSignalResultFail
		}

		// get param from memory
		requestHeaderByte := make([]byte, requestHeaderLen)
		copy(requestHeaderByte, vm.Memory[requestHeaderPtr:requestHeaderPtr+requestHeaderLen])
		requestBody := make([]byte, requestBodyLen)
		copy(requestBody, vm.Memory[requestBodyPtr:requestBodyPtr+requestBodyLen])
		var requestHeaderItems []*serialize.EasyCodecItem
		requestHeaderItems = serialize.EasyUnmarshal(requestHeaderByte)

		w.Vm = vm
		w.RequestBody = requestBody
		w.RequestHeader = requestHeaderItems
		var method interface{}
		var ok bool
		method, ok = serialize.GetValueFromItems(requestHeaderItems, "method", serialize.EasyKeyType_SYSTEM)
		if !ok {
			msg := fmt.Sprintf("get method failed:%s requestHeader=%s requestBody=%s", "request header have no method", string(requestHeaderByte), string(requestBody))
			return w.recordMsg(w.ContractResult, msg)
		}
		switch method.(string) {
		case protocol.ContractMethodLogMessage:
			return w.LogMsg()
		case protocol.ContractMethodGetStateLen:
			return w.GetStateLen()
		case protocol.ContractMethodGetState:
			return w.GetState()
		case protocol.ContractMethodPutState:
			return w.PutState()
		case protocol.ContractMethodDeleteState:
			return w.DeleteState()
		case protocol.ContractMethodSuccessResult:
			return w.SuccessResult()
		case protocol.ContractMethodErrorResult:
			return w.ErrorResult()
		case protocol.ContractMethodCallContract:
			return w.CallContract()
		case protocol.ContractMethodCallContractLen:
			return w.CallContractLen()
		case protocol.ContractMethodEmitEvent:
			return w.EmitEvent()

		default:
			w.Log.Errorf("method is %s not match.", method)
		}
		return protocol.ContractSdkSignalResultFail
	})
}

// GetStateLen get state length from chain
func (w *WaciInstance) GetStateLen() int32 {
	return w.getStateCore(true)
}

// GetStateLen get state from chain
func (w *WaciInstance) GetState() int32 {
	return w.getStateCore(false)
}

func (w *WaciInstance) getStateCore(isGetLen bool) int32 {
	req := serialize.EasyUnmarshal(w.RequestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return w.recordMsg(w.ContractResult, err.Error())
	}

	valuePtrInt, _ := strconv.Atoi(valuePtr.(string))

	if isGetLen {
		contractName := w.ContractId.ContractName
		value, err := w.TxSimContext.Get(contractName, protocol.GetKeyStr(key.(string), field.(string)))
		if err != nil {
			msg := fmt.Sprintf("method getStateCore get fail. key=%s, field=%s, error:%s", key.(string), field.(string), err.Error())
			return w.recordMsg(w.ContractResult, msg)
		}
		copy(w.Vm.Memory[valuePtrInt:valuePtrInt+4], IntToBytes(int32(len(value))))
		w.GetStateCache = value
	} else {
		len := len(w.GetStateCache)
		if len != 0 {
			copy(w.Vm.Memory[valuePtrInt:valuePtrInt+len], w.GetStateCache)
			w.GetStateCache = nil
		}
	}
	return protocol.ContractSdkSignalResultSuccess
}

// PutState put state to chain
func (w *WaciInstance) PutState() int32 {
	req := serialize.EasyUnmarshal(w.RequestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	value, _ := serialize.GetValueFromItems(req, "value", serialize.EasyKeyType_USER)
	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return w.recordMsg(w.ContractResult, err.Error())
	}

	contractName := w.ContractId.ContractName
	err := w.TxSimContext.Put(contractName, protocol.GetKeyStr(key.(string), field.(string)), []byte(value.(string)))
	if err != nil {
		return w.recordMsg(w.ContractResult, "PutState put fail. "+err.Error())
	}
	return protocol.ContractSdkSignalResultSuccess
}

// EmitEvent emit event to chain
func (w *WaciInstance) EmitEvent() int32 {
	req := serialize.EasyUnmarshal(w.RequestBody)
	topic, _ := serialize.GetValueFromItems(req, "topic", serialize.EasyKeyType_USER)
	if err := protocol.CheckTopicStr(topic.(string)); err != nil {
		return w.recordMsg(w.ContractResult, err.Error())

	}
	var eventData []string
	for i := 1; i < len(req); i++ {
		data := req[i].Value.(string)
		eventData = append(eventData, data)
		w.Log.Debugf("EmitEvent EventData :%v", data)
	}
	if err := protocol.CheckEventData(eventData); err != nil {
		return w.recordMsg(w.ContractResult, err.Error())
	}

	w.ContractEvent = append(w.ContractEvent, &commonPb.ContractEvent{
		ContractName:    w.ContractId.ContractName,
		ContractVersion: w.ContractId.ContractVersion,
		Topic:           topic.(string),
		TxId:            w.TxSimContext.GetTx().Header.TxId,
		EventData:       eventData,
	})

	return protocol.ContractSdkSignalResultSuccess

}

// DeleteState delete state from chain
func (w *WaciInstance) DeleteState() int32 {
	req := serialize.EasyUnmarshal(w.RequestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return w.recordMsg(w.ContractResult, err.Error())
	}

	contractName := w.ContractId.ContractName
	err := w.TxSimContext.Del(contractName, protocol.GetKeyStr(key.(string), field.(string)))
	if err != nil {
		return w.recordMsg(w.ContractResult, err.Error())
	}

	return protocol.ContractSdkSignalResultSuccess
}

// SuccessResult record the results of contract execution success
func (w *WaciInstance) SuccessResult() int32 {
	if w.ContractResult.Code == commonPb.ContractResultCode_FAIL {
		return protocol.ContractSdkSignalResultFail
	}
	w.ContractResult.Code = commonPb.ContractResultCode_OK
	w.ContractResult.Result = w.RequestBody
	return protocol.ContractSdkSignalResultSuccess
}

// ErrorResult record the results of contract execution error
func (w *WaciInstance) ErrorResult() int32 {
	w.ContractResult.Code = commonPb.ContractResultCode_FAIL
	w.ContractResult.Message += string(w.RequestBody)
	return protocol.ContractSdkSignalResultSuccess
}

//  CallContractLen invoke cross contract calls, save result to cache and putout result length
func (w *WaciInstance) CallContractLen() int32 {
	return w.callContractCore(true)
}

//  CallContractLen get cross contract call result from cache
func (w *WaciInstance) CallContract() int32 {
	return w.callContractCore(false)
}

func (w *WaciInstance) callContractCore(isGetLen bool) int32 {
	req := serialize.EasyUnmarshal(w.RequestBody)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	contractName, _ := serialize.GetValueFromItems(req, "contract_name", serialize.EasyKeyType_USER)
	method, _ := serialize.GetValueFromItems(req, "method", serialize.EasyKeyType_USER)
	param, _ := serialize.GetValueFromItems(req, "param", serialize.EasyKeyType_USER)
	paramItem := serialize.EasyUnmarshal(param.([]byte))
	valuePtrInt, _ := strconv.Atoi(valuePtr.(string))

	if !isGetLen { // get value from cache
		result := w.TxSimContext.GetCurrentResult()
		copy(w.Vm.Memory[valuePtrInt:valuePtrInt+len(result)], result)
		return protocol.ContractSdkSignalResultSuccess
	}

	// check param
	if len(contractName.(string)) == 0 {
		return w.recordMsg(w.ContractResult, "CallContract contractName is null")
	}
	if len(method.(string)) == 0 {
		return w.recordMsg(w.ContractResult, "CallContract method is null")
	}
	if len(paramItem) > protocol.ParametersKeyMaxCount {
		return w.recordMsg(w.ContractResult, "expect less than 20 parameters, but get "+strconv.Itoa(len(paramItem)))
	}
	for _, item := range paramItem {
		if len(item.Key) > protocol.DefaultStateLen {
			msg := fmt.Sprintf("CallContract param expect Key length less than %d, but get %d", protocol.DefaultStateLen, len(item.Key))
			return w.recordMsg(w.ContractResult, msg)
		}
		match, err := regexp.MatchString(protocol.DefaultStateRegex, item.Key)
		if err != nil || !match {
			msg := fmt.Sprintf("CallContract param expect Key no special characters, but get %s. letter, number, dot and underline are allowed", item.Key)
			return w.recordMsg(w.ContractResult, msg)
		}
		if len(item.Value.(string)) > protocol.ParametersValueMaxLength {
			msg := fmt.Sprintf("expect Value length less than %d, but get %d", protocol.ParametersValueMaxLength, len(item.Value.(string)))
			return w.recordMsg(w.ContractResult, msg)
		}
	}
	if err := protocol.CheckKeyFieldStr(contractName.(string), method.(string)); err != nil {
		return w.recordMsg(w.ContractResult, err.Error())
	}

	// call contract
	w.Vm.Gas = w.Vm.Gas + protocol.CallContractGasOnce
	paramMap := serialize.EasyCodecItemToParamsMap(paramItem)
	result, code := w.TxSimContext.CallContract(&commonPb.ContractId{ContractName: contractName.(string)}, method.(string), nil, paramMap, w.Vm.Gas, commonPb.TxType_INVOKE_USER_CONTRACT)
	w.Vm.Gas = w.Vm.Gas + uint64(result.GasUsed)
	if code != commonPb.TxStatusCode_SUCCESS {
		msg := fmt.Sprintf("CallContract %s, msg:%s", code.String(), result.Message)
		return w.recordMsg(w.ContractResult, msg)
	}
	// set value length to memory
	l := IntToBytes(int32(len(result.Result)))
	copy(w.Vm.Memory[valuePtrInt:valuePtrInt+4], l)
	return protocol.ContractSdkSignalResultSuccess
}

func IntToBytes(x int32) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.LittleEndian, x)
	return bytesBuffer.Bytes()
}

func (w *WaciInstance) recordMsg(r *commonPb.ContractResult, msg string) int32 {
	r.Message += msg
	r.Code = commonPb.ContractResultCode_FAIL
	w.Log.Errorf("waci log>> [%s] %s:%s", w.TxSimContext.GetTx().GetHeader().GetTxId(), w.ContractId.ContractName, msg)
	return protocol.ContractSdkSignalResultFail
}
