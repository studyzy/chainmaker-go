/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"strconv"
	"sync"

	"chainmaker.org/chainmaker/common/serialize"
	"chainmaker.org/chainmaker/common/vmcbor"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
)

// SimContext record the contract context
type SimContext struct {
	TxSimContext   protocol.TxSimContext
	ContractId     *commonPb.ContractId
	ContractResult *commonPb.ContractResult
	Log            *logger.CMLogger
	Instance       *wasm.Instance

	method        string
	Ctx           *vmcbor.RuntimeContext
	parameters    map[string]string
	CtxPtr        int32
	GetStateCache []byte // cache call method GetStateLen value result, one cache per transaction
	ChainId       string
	ContractEvent []*commonPb.ContractEvent
}

// NewSimContext for every transaction
func NewSimContext(method string, log *logger.CMLogger, chainId string) *SimContext {
	sc := SimContext{
		method:  method,
		Log:     log,
		ChainId: chainId,
	}

	sc.putCtxPointer()

	return &sc
}

// CallMethod will call contract method
func (sc *SimContext) CallMethod(instance *wasm.Instance) error {
	var bytes []byte

	runtimeFn, ok := instance.Exports[protocol.ContractRuntimeTypeMethod]
	if !ok {
		return fmt.Errorf("method [%s] not export", protocol.ContractRuntimeTypeMethod)
	}
	sdkType, err := runtimeFn()
	if err != nil {
		return err
	}

	runtimeSdkType := sdkType.ToI32()
	if int32(commonPb.RuntimeType_WASMER) == runtimeSdkType {
		sc.parameters[protocol.ContractContextPtrParam] = strconv.Itoa(int(sc.CtxPtr))
		ec := serialize.NewEasyCodecWithMap(sc.parameters)
		bytes = ec.Marshal()
	} else {
		return fmt.Errorf("runtime type error, expect rust:[%d], but got %d", uint64(commonPb.RuntimeType_WASMER), runtimeSdkType)
	}

	return sc.callContract(instance, sc.method, bytes)
}

func (sc *SimContext) callContract(instance *wasm.Instance, methodName string, bytes []byte) error {

	lengthOfSubject := len(bytes)

	exports := instance.Exports[protocol.ContractAllocateMethod]
	// Allocate memory for the subject, and get a pointer to it.
	allocateResult, err := exports(lengthOfSubject)
	if err != nil {
		return err
	}
	dataPtr := allocateResult.ToI32()

	// Write the subject into the memory.
	memory := instance.Memory.Data()[dataPtr:]

	//copy(memory, bytes)
	for nth := 0; nth < lengthOfSubject; nth++ {
		memory[nth] = bytes[nth]
	}

	// Calls the `invoke` exported function. Given the pointer to the subject.
	export, ok := instance.Exports[methodName]
	if !ok {
		return fmt.Errorf("method [%s] not export", methodName)
	}

	_, err = export()
	if err != nil {
		return err
	}

	// release wasm memory
	//_, err = instance.Exports["deallocate"](dataPtr)
	return err
}

// CallDeallocate deallocate vm memory before closing the instance
func CallDeallocate(instance *wasm.Instance) error {
	_, err := instance.Exports[protocol.ContractDeallocateMethod](0)
	return err
}

// putCtxPointer revmoe SimContext from cache
func (sc *SimContext) removeCtxPointer() {
	vbm := GetVmBridgeManager()
	vbm.remove(sc.CtxPtr)
}

var ctxIndex = int32(0)
var lock sync.Mutex

// putCtxPointer save SimContext to cache
func (sc *SimContext) putCtxPointer() {
	lock.Lock()
	ctxIndex++
	if ctxIndex > 1e8 {
		ctxIndex = 0
	}
	sc.CtxPtr = ctxIndex
	lock.Unlock()
	vbm := GetVmBridgeManager()
	vbm.put(sc.CtxPtr, sc)
}
