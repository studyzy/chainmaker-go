/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"strconv"
	"sync"
	"unsafe"

	"chainmaker.org/chainmaker-go/wasi"

	"chainmaker.org/chainmaker/common/serialize"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
)

// #include <stdlib.h>

// extern int sysCall(void *context, int requestHeaderPtr, int requestHeaderLen, int requestBodyPtr, int requestBodyLen);
// extern void logMessage(void *context, int pointer, int length);
// extern int fdWrite(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern int fdRead(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern int fdClose(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern int fdSeek(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern void procExit(void *contextfd,int exitCode);
import "C"

var log = logger.GetLogger(logger.MODULE_VM)

// Wacsi WebAssembly chainmaker system interface
var wacsi = wasi.NewWacsi()

// WaciInstance record wasmer vm request parameter
type WaciInstance struct {
	Sc          *SimContext
	RequestBody []byte // sdk request param
	Memory      []byte // vm memory
	ChainId     string
}

// LogMessage print log to file
func (s *WaciInstance) LogMessage() int32 {
	s.Sc.Log.Debugf("wasmer log>> [%s] %s", s.Sc.TxSimContext.GetTx().Header.TxId, string(s.RequestBody))
	return protocol.ContractSdkSignalResultSuccess
}

// logMessage print log to file
//export logMessage
func logMessage(context unsafe.Pointer, pointer int32, length int32) {
	var instanceContext = wasm.IntoInstanceContext(context)
	var memory = instanceContext.Memory().Data()

	gotText := string(memory[pointer : pointer+length])
	log.Debugf("wasmer log>> " + gotText)
}

// sysCall wasmer vm call chain entry
//export sysCall
func sysCall(context unsafe.Pointer, requestHeaderPtr int32, requestHeaderLen int32, requestBodyPtr int32, requestBodyLen int32) int32 {
	if requestHeaderLen == 0 {
		log.Error("wasmer log>> requestHeader is null.")
		return protocol.ContractSdkSignalResultFail
	}
	// get param from memory
	instanceContext := wasm.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	requestHeaderByte := make([]byte, requestHeaderLen)
	copy(requestHeaderByte, memory[requestHeaderPtr:requestHeaderPtr+requestHeaderLen])
	requestBody := make([]byte, requestBodyLen)
	copy(requestBody, memory[requestBodyPtr:requestBodyPtr+requestBodyLen])
	ec := serialize.NewEasyCodecWithBytes(requestHeaderByte)
	ctxPtr, err := ec.GetValue("ctx_ptr", serialize.EasyKeyType_SYSTEM)
	if err != nil {
		log.Error("get ctx_ptr failed:%s requestHeader=%s requestBody=%s", "request header have no ctx_ptr", string(requestHeaderByte), string(requestBody), err)
	}
	vbm := GetVmBridgeManager()
	sc := vbm.get(ctxPtr.(int32))

	s := &WaciInstance{
		Sc:          sc,
		RequestBody: requestBody,
		Memory:      memory,
		ChainId:     sc.ChainId,
	}

	method, err := ec.GetValue("method", serialize.EasyKeyType_SYSTEM)
	if err != nil {
		log.Error("get method failed:%s requestHeader=%s requestBody=%s", "request header have no method", string(requestHeaderByte), string(requestBody), err)
	}
	switch method.(string) {
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
	// sql
	case protocol.ContractMethodExecuteUpdate:
		return s.ExecuteUpdate()
	case protocol.ContractMethodExecuteDdl:
		return s.ExecuteDDL()
	case protocol.ContractMethodExecuteQuery:
		return s.ExecuteQuery()
	case protocol.ContractMethodExecuteQueryOne:
		return s.ExecuteQueryOne()
	case protocol.ContractMethodExecuteQueryOneLen:
		return s.ExecuteQueryOneLen()
	case protocol.ContractMethodRSHasNext:
		return s.RSHasNext()
	case protocol.ContractMethodRSNextLen:
		return s.RSNextLen()
	case protocol.ContractMethodRSNext:
		return s.RSNext()
	case protocol.ContractMethodRSClose:
		return s.RSClose()
	default:
		log.Errorf("method[%s] is not match.", method)
	}
	return protocol.ContractSdkSignalResultFail
}

// SuccessResult record the results of contract execution success
func (s *WaciInstance) SuccessResult() int32 {
	return wacsi.SuccessResult(s.Sc.ContractResult, s.RequestBody)
}

// ErrorResult record the results of contract execution error
func (s *WaciInstance) ErrorResult() int32 {
	return wacsi.ErrorResult(s.Sc.ContractResult, s.RequestBody)
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
	data, err, gas := wacsi.CallContract(s.RequestBody, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache, s.Sc.Instance.GetGasUsed(), isLen)
	s.Sc.GetStateCache = data // reset data
	s.Sc.Instance.SetGasUsed(gas)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// EmitEvent emit event to chain
func (s *WaciInstance) EmitEvent() int32 {
	contractEvent, err := wacsi.EmitEvent(s.RequestBody, s.Sc.TxSimContext, s.Sc.ContractId, s.Sc.Log)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	s.Sc.ContractEvent = append(s.Sc.ContractEvent, contractEvent)
	return protocol.ContractSdkSignalResultSuccess
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
	data, err := wacsi.BulletProofsOperation(s.RequestBody, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
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
	data, err := wacsi.PaillierOperation(s.RequestBody, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// wasi
//export fdWrite
func fdWrite(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export fdRead
func fdRead(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export fdClose
func fdClose(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export fdSeek
func fdSeek(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export procExit
func procExit(context unsafe.Pointer, exitCode int32) {
	panic("exit called by contract, code:" + strconv.Itoa(int(exitCode)))
}

func (s *WaciInstance) recordMsg(msg string) int32 {
	if len(s.Sc.ContractResult.Message) > 0 {
		s.Sc.ContractResult.Message += ". error message: " + msg
	} else {
		s.Sc.ContractResult.Message += "error message: " + msg
	}
	s.Sc.ContractResult.Code = commonPb.ContractResultCode_FAIL
	s.Sc.Log.Errorf("wasmer log>> [%s] %s", s.Sc.ContractId.ContractName, msg)
	return protocol.ContractSdkSignalResultFail
}

var (
	vmBridgeManagerMutex = &sync.Mutex{}
	bridgeSingleton      *vmBridgeManager
)

type vmBridgeManager struct {
	//wasmImports *wasm.Imports
	pointerLock     sync.Mutex
	simContextCache map[int32]*SimContext
}

// GetVmBridgeManager get singleton vmBridgeManager struct
func GetVmBridgeManager() *vmBridgeManager {
	if bridgeSingleton == nil {
		vmBridgeManagerMutex.Lock()
		defer vmBridgeManagerMutex.Unlock()
		if bridgeSingleton == nil {
			log.Debugf("init vmBridgeManager")
			bridgeSingleton = &vmBridgeManager{}
			bridgeSingleton.simContextCache = make(map[int32]*SimContext)
			//bridgeSingleton.wasmImports = bridgeSingleton.GetImports()
		}
	}
	return bridgeSingleton
}

// put the context
func (b *vmBridgeManager) put(k int32, v *SimContext) {
	b.pointerLock.Lock()
	defer b.pointerLock.Unlock()
	b.simContextCache[k] = v
}

// get the context
func (b *vmBridgeManager) get(k int32) *SimContext {
	b.pointerLock.Lock()
	defer b.pointerLock.Unlock()
	return b.simContextCache[k]
}

// remove the context
func (b *vmBridgeManager) remove(k int32) {
	b.pointerLock.Lock()
	defer b.pointerLock.Unlock()
	delete(b.simContextCache, k)
}

// NewWasmInstance new wasm instance. Apply for new memory.
func (b *vmBridgeManager) NewWasmInstance(byteCode []byte) (wasm.Instance, error) {
	return wasm.NewInstanceWithImports(byteCode, b.GetImports())
}

// GetImports return export interface to cgo
func (b *vmBridgeManager) GetImports() *wasm.Imports {
	imports := wasm.NewImports().Namespace("env")
	// parameter explain:  1、["log_message"]: rust extern "C" method name 2、[logMessage] go method ptr 3、[C.logMessage] cgo function pointer.
	imports.Append("sys_call", sysCall, C.sysCall)
	imports.Append("log_message", logMessage, C.logMessage)
	// for wacsi empty interface
	imports.Namespace("wasi_unstable")
	imports.Append("fd_write", fdWrite, C.fdWrite)
	imports.Append("fd_read", fdRead, C.fdRead)
	imports.Append("fd_close", fdClose, C.fdClose)
	imports.Append("fd_seek", fdSeek, C.fdSeek)

	imports.Namespace("wasi_snapshot_preview1")
	imports.Append("proc_exit", procExit, C.procExit)
	//imports.Append("fd_write", fdWrite2, C.fdWrite2)
	//imports.Append("environ_sizes_get", fdWrite, C.fdWrite)
	//imports.Append("proc_exit", fdWrite, C.fdWrite)
	//imports.Append("environ_get", fdWrite, C.fdWrite)

	return imports
}
