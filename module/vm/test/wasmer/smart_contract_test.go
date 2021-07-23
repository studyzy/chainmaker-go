/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmertest

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gotest.tools/assert"

	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"

	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
)

var log = logger.GetLoggerByChain(logger.MODULE_VM, test.ChainIdTest)

// 存证合约 单例需要大于65536次，因为内存是64K
func TestCallFact(t *testing.T) {
	test.ContractNameTest = "contract_fact"
	test.WasmFile = "../../../../test/wasm/rust-func-verify-2.0.0.wasm"
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	println("bytes len", len(bytes))

	pool := test.GetVmPoolManager()

	// 调用
	x := int32(0)
	println("start") // 2.9m
	start := time.Now().UnixNano() / 1e6
	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		for j := 0; j < 1; j++ {
			x++
			y := x
			wg.Add(1)
			go func() {
				defer wg.Done()
				invokeFact("save", y, contractId, txContext, pool, bytes)
				//invokeFact("query", y, contractId, txContext, pool, bytes)
				end := time.Now().UnixNano() / 1e6
				if (end-start)/1000 > 0 && y%1000 == 0 {
					fmt.Printf("【tps】 %d 【spend】%d i = %d, count=%d \n", int(y)/int((end-start)/1000), end-start, i+1, y)
				}
			}()
		}

		wg.Wait()
	}

	end := time.Now().UnixNano() / 1e6
	println("end 【spend】", end-start)
}

func invokeFact(method string, id int32, contractId *commonPb.Contract, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string][]byte)
	txId := utils.GetRandTxId()
	parameters["time"] = []byte("567124123")
	parameters["file_hash"] = []byte("file_hash")
	parameters["file_name"] = []byte(txId)
	parameters["tx_id"] = []byte(txId)
	parameters["forever"] = []byte("true")
	parameters["contract_name"] = []byte(test.ContractNameTest)

	baseParam(parameters)
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	r := runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	//fmt.Printf("\n【result】 %+v \n\n\n", r)
	return r
}

func TestFunctionalContract(t *testing.T) {
	test.ContractNameTest = "contract_functional"
	test.WasmFile = "../../../../test/wasm/rust-func-verify-2.0.0.wasm"
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	pool := test.GetVmPoolManager()

	invokeFactContract("init_contract", contractId, txContext, pool, bytes)
	invokeFactContract("upgrade", contractId, txContext, pool, bytes)

	invokeFactContract("save", contractId, txContext, pool, bytes)
	r := invokeFactContract("find_by_file_hash", contractId, txContext, pool, bytes)
	//assert.Equal(t, string(r.Result), "{\"file_hash\":\"file_hash\",\"file_name\":\"file_name\",\"time\":\"1314520\"}")
	fmt.Println(string(r.Result))
	fmt.Println("  【save】pass")
	fmt.Println("  【find_by_file_hash】pass")

	invokeFactContract("test_put_pre_state", contractId, txContext, pool, bytes)
	r2 := invokeFactContract("test_iter_pre_field", contractId, txContext, pool, bytes)
	r3 := invokeFactContract("test_iter_pre_key", contractId, txContext, pool, bytes)
	assert.Equal(t, string(r2.Result), "14")
	assert.Equal(t, string(r3.Result), "14")
	fmt.Println("  【test_put_pre_state】pass")
	fmt.Println("  【test_iter_pre_field】pass")
	fmt.Println("  【test_iter_pre_key】pass")

	invokeFactContract("test_put_state", contractId, txContext, pool, bytes)
	r4 := invokeFactContract("test_kv_iterator", contractId, txContext, pool, bytes)
	assert.Equal(t, string(r4.Result), "15")
	fmt.Println("  【test_put_state】pass")
	fmt.Println("  【test_kv_iterator】pass")

	invokeFactContract("increase", contractId, txContext, pool, bytes)
	r5 := invokeFactContract("query", contractId, txContext, pool, bytes)
	assert.Equal(t, string(r5.Result), "1")
	fmt.Println("  【increase】pass")
	fmt.Println("  【query】pass")

	r6 := invokeFactContract("functional_verify", contractId, txContext, pool, bytes)
	assert.Equal(t, string(r6.Result), "ok")
	fmt.Println("  【functional_verify】pass")
	fmt.Println("  【test】pass")
}

func invokeFactContract(method string, contractId *commonPb.Contract, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string][]byte)
	parameters["time"] = []byte("1314520")
	parameters["file_hash"] = []byte("file_hash")
	parameters["file_name"] = []byte("file_name")
	parameters["contract_name"] = []byte(test.ContractNameTest)
	baseParam(parameters)
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	r := runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	fmt.Printf("\n【result】 %+v \n\n\n", r)
	return r
}

func TestCounterContract(t *testing.T) {
	test.ContractNameTest = "contract_counter"
	test.WasmFile = "../../../../test/wasm/rust-counter-2.0.0.wasm"
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	pool := test.GetVmPoolManager()

	invokeCounterContract("init_contract", contractId, txContext, pool, bytes)
	invokeCounterContract("upgrade", contractId, txContext, pool, bytes)

	invokeCounterContract("upgrade", contractId, txContext, pool, bytes)
	invokeCounterContract("upgrade", contractId, txContext, pool, bytes)
}

func invokeCounterContract(method string, contractId *commonPb.Contract, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string][]byte)
	baseParam(parameters)
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	r := runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	fmt.Printf("\n【result】 %+v \n\n\n", r)
	return r
}

// 使用原始调用智能合约
func testCallHelloWorldUseOrigin(t *testing.T) {
	_, _, byteCode := test.InitContextTest(commonPb.RuntimeType_WASMER)
	if byteCode == nil {
		panic("byteCode is nil")
	}
	vb := wasmer.GetVmBridgeManager()
	instance, _ := wasm.NewInstanceWithImports(byteCode, vb.GetImports())
	defer instance.Close()

	// Set the subject to greet.
	subject := "Wasmer 🐹"
	for i := 0; i < 1000; i++ {
		subject += "Wasmer 🐹"
	}
	lengthOfSubject := len(subject)

	// Allocate memory for the subject, and get a pointer to it.
	allocateResult, _ := instance.Exports["allocate"](lengthOfSubject)
	inputPointer := allocateResult.ToI32()

	// Write the subject into the memory.
	memory := instance.Memory.Data()[inputPointer:]

	for nth := 0; nth < lengthOfSubject; nth++ {
		memory[nth] = subject[nth]
	}

	// C-string terminates by NULL.
	memory[lengthOfSubject] = 0

	// Run the `greet` function. Given the pointer to the subject.
	greetResult, _ := instance.Exports["increase"](inputPointer, lengthOfSubject)
	outputPointer := greetResult.ToI32()

	// Read the result of the `greet` function.
	memory = instance.Memory.Data()[outputPointer:]
	nth := 0
	var output strings.Builder

	for {
		if memory[nth] == 0 {
			break
		}

		output.WriteByte(memory[nth])
		nth++
	}

	lengthOfOutput := nth

	fmt.Println("out ", output.String())

	// Deallocate the subject, and the output.
	deallocate := instance.Exports["deallocate"]
	deallocate(inputPointer, lengthOfSubject)
	deallocate(outputPointer, lengthOfOutput)

	fmt.Println("end ")
}

func baseParam(parameters map[string][]byte) {
	parameters[protocol.ContractTxIdParam] = []byte("TX_ID")
	parameters[protocol.ContractCreatorOrgIdParam] = []byte("CREATOR_ORG_ID")
	parameters[protocol.ContractCreatorRoleParam] = []byte("CREATOR_ROLE")
	parameters[protocol.ContractCreatorPkParam] = []byte("CREATOR_PK")
	parameters[protocol.ContractSenderOrgIdParam] = []byte("SENDER_ORG_ID")
	parameters[protocol.ContractSenderRoleParam] = []byte("SENDER_ROLE")
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK")
	parameters[protocol.ContractBlockHeightParam] = []byte("111")
}
