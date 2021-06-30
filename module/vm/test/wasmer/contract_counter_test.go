/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmertest

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
)

// 功能测试，边界测试
//func TestCallCounterFunc(t *testing.T) {
//	test.WasmFile = "../../../../test/wasm/counter-rust-0.7.2.wasm"
//	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
//	//bytes, _ = wasm.ReadBytes("../../../../test/wasm/counter-rust-0.6.2.wasm")
//	println("bytes len", len(bytes))
//
//	pool := test.GetVmPoolManager()
//	println("start")
//	start := time.Now().UnixNano() / 1e6
//
//	//invokeCounterCallContractIncreaseSelf(contractId, txContext, pool, bytes)
//	//invokeCounterCallContractIncreaseForever(contractId, txContext, pool, bytes)
//	invokeCounterCallContractIncrease(contractId, txContext, pool, bytes)
//	invokeCounterCallContractIncrease(contractId, txContext, pool, bytes)
//	invokeCounterCallContractQuery(contractId, txContext, pool, bytes)
//	invokeCounterCallContractIncrease(contractId, txContext, pool, bytes)
//	invokeCounterCallContractIncrease(contractId, txContext, pool, bytes)
//	invokeCounterCallContractQuery(contractId, txContext, pool, bytes)
//
//	invokeCounterQuery(contractId, txContext, pool, bytes)
//	invokeCounterIncrease(contractId, txContext, pool, bytes)
//	invokeCounterQuery(contractId, txContext, pool, bytes)
//	invokeCounterDeleteStore(contractId, txContext, pool, bytes)
//	invokeCounterGetStore(contractId, txContext, pool, bytes)
//	invokeCounterSetStore(contractId, txContext, pool, bytes)
//	invokeCounterGetStore(contractId, txContext, pool, bytes)
//	invokeCounterDeleteStore(contractId, txContext, pool, bytes)
//	invokeCounterGetStore(contractId, txContext, pool, bytes)
//	invokeCounterCallSelf(contractId, txContext, pool, bytes)
//	invokeCounterGetCalc(contractId, txContext, pool, bytes)
//	invokeCounterCalcJson(contractId, txContext, pool, bytes)
//
//	end := time.Now().UnixNano() / 1e6
//	println("end 【spend】", end-start)
//	time.Sleep(time.Second * 5)
//}
//func TestCallCounterPanic(t *testing.T) {
//	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
//	pool := wasmer.NewVmPoolManager("chain001")
//	invokeCounterPanic(contractId, txContext, pool, bytes)
//}

func invokeCounterQuery(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "query"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "key"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeCounterIncrease(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "increase"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "key"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeCounterPanic(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "calc_json"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["func_name"] = "panic"
	parameters["data1"] = "2"
	parameters["data2"] = "3"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeCounterCalcJson(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "calc_json"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["data1"] = "2"
	parameters["data2"] = "3"
	parameters["func_name"] = "add"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	parameters["func_name"] = "sub"
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	parameters["func_name"] = "mul"
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	parameters["func_name"] = "div"
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	parameters["func_name"] = "set_data"
	parameters["data3"] = "data333"
	parameters["data4"] = "3"
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	parameters["func_name"] = "delete"
	parameters["data3"] = "data333"
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

}

func invokeCounterGetCalc(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "set_store"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "zitao"
	parameters["name"] = "func_name"
	parameters["value"] = "111"
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	method = "get_calc"
	parameters = make(map[string]string)
	baseParam(parameters)
	parameters["func_name"] = "func_name"
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeCounterCallSelf(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "set_store"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "zitao"
	parameters["name"] = "callnum"
	parameters["value"] = "3"
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	method = "call_self"
	parameters = make(map[string]string)
	baseParam(parameters)
	runtime, _ = pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeCounterSetStore(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "set_store"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "key"
	parameters["name"] = "name"
	parameters["value"] = "value"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeCounterGetStore(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "get_store"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "key"
	parameters["name"] = "name"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeCounterDeleteStore(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "delete_store"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["key"] = "key"
	parameters["name"] = "name"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeCounterCallContractIncrease(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "call_contract_test"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["contractName"] = test.ContractNameTest
	parameters["method"] = "increase"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeCounterCallContractIncreaseForever(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "call_contract_test"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["contractName"] = test.ContractNameTest
	parameters["method"] = "increase"
	parameters["count"] = "2"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeCounterCallContractIncreaseSelf(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "call_contract_self"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["contractName"] = test.ContractNameTest
	parameters["method"] = "call_contract_self"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeCounterCallContractQuery(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "call_contract_test"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["contractName"] = test.ContractNameTest
	parameters["method"] = "query"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
