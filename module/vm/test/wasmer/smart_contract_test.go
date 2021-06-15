/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmertest

import (
	"fmt"
	"gotest.tools/assert"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"

	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
)

var log = logger.GetLoggerByChain(logger.MODULE_VM, test.ChainIdTest)

// 存证合约 单例需要大于65536次，因为内存是64K
func TestCallFact(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/rust-fact-1.2.1.wasm"
	//test.WasmFile = "../../../../test/wasm/rust-func-verify-1.2.1.wasm"
	//test.WasmFile = "D:\\develop\\workspace\\chainMaker\\chainmaker-contract-sdk-rust\\target\\wasm32-unknown-unknown\\release\\chainmaker_contract.wasm"
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
				invokeFact("find_by_file_hash", y, contractId, txContext, pool, bytes)
				//invokeFact("test_put_state", y, contractId, txContext, pool, bytes)
				//invokeFact("test_kv_iterator", y, contractId, txContext, pool, bytes)
				//invokeFact("functional_verify", y, contractId, txContext, pool, bytes)

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
	time.Sleep(time.Second * 2)
	println("reset vm pool")
	pool.ResetAllPool()
	//time.Sleep(time.Second * 500)
	runtime.GC()
}

func invokeFact(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	parameters := make(map[string]string)
	txId := utils.GetRandTxId()
	parameters["time"] = "567124123"
	parameters["file_hash"] = "file_hash"
	parameters["file_name"] = txId
	parameters["tx_id"] = txId
	parameters["forever"] = "true"
	parameters["contract_name"] = test.ContractNameTest

	baseParam(parameters)
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	r := runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	fmt.Println("【result】", r)
}

func TestFactContract(t *testing.T) {
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	test.WasmFile = "../../../../test/wasm/rust-func-verify-1.2.1.wasm"
	pool := wasmer.NewVmPoolManager("chain001")
	invokeFactContract("save", contractId, txContext, pool, bytes)
	r := invokeFactContract("find_by_file_hash", contractId, txContext, pool, bytes)
	assert.Equal(t, string(r.Result), "{\"file_hash\":\"file_hash\",\"file_name\":\"file_name\",\"time\":\"1314520\"}")
}

func invokeFactContract(method string, contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string]string)
	parameters["time"] = "1314520"
	parameters["file_hash"] = "file_hash"
	parameters["file_name"] = "file_name"
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	r := runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	return r
}

func TestCounterContract(t *testing.T) {
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	test.WasmFile = "../../../../test/wasm/rust-func-verify-1.2.1.wasm"
	pool := wasmer.NewVmPoolManager("chain001")
	invokeCounterContract("increase", contractId, txContext, pool, bytes)
	invokeCounterContract("increase", contractId, txContext, pool, bytes)
	r := invokeCounterContract("query", contractId, txContext, pool, bytes)
	assert.Equal(t, string(r.Result), "2")
}

func invokeCounterContract(method string, contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string]string)
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	r := runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	return r
}

// 使用原始调用智能合约
func TestCallHelloWorldUseOrigin(t *testing.T) {
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
	time.Sleep(time.Second * 2)
}

func baseParam(parameters map[string]string) {
	parameters[protocol.ContractTxIdParam] = "TX_ID"
	parameters[protocol.ContractCreatorOrgIdParam] = "CREATOR_ORG_ID"
	parameters[protocol.ContractCreatorRoleParam] = "CREATOR_ROLE"
	parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
	parameters[protocol.ContractSenderOrgIdParam] = "SENDER_ORG_ID"
	parameters[protocol.ContractSenderRoleParam] = "SENDER_ROLE"
	parameters[protocol.ContractSenderPkParam] = "SENDER_PK"
	parameters[protocol.ContractBlockHeightParam] = "111"
}

func TestKVIteratorTest(t *testing.T) {
	//test.WasmFile = "../../../../test/wasm/rust-func-verify-1.2.0.wasm"
	test.WasmFile = "D:\\develop\\workspace\\chainMaker\\chainmaker-contract-sdk-rust\\target\\wasm32-unknown-unknown\\release\\chainmaker_contract.wasm"
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	println("bytes len", len(bytes))

	pool := test.GetVmPoolManager()

	invokeKvIterator("test_put_state", 1, contractId, txContext, pool, bytes)

	// 调用
	x := int32(0)
	println("start") // 2.9m
	start := time.Now().UnixNano() / 1e6
	wg := sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		for j := 0; j < 10; j++ {
			x++
			y := x
			wg.Add(1)
			go func() {
				defer wg.Done()
				invokeKvIterator("test_get_state", y, contractId, txContext, pool, bytes)

				end := time.Now().UnixNano() / 1e6
				if (end-start)/1000 > 0 && y%1000 == 0 {
					fmt.Printf("【tps】 %d 【spend】%d i = %d, count=%d \n", int(y)/int((end-start)/1000), end-start, i+1, y)
				}
			}()
		}
		wg.Wait()
	}
	end := time.Now().UnixNano() / 1e6

	println("end1 ", start, end, end-start, end, (end-start)/500) // 2.9m
	start = time.Now().UnixNano() / 1e6
	wg = sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		for j := 0; j < 10; j++ {
			x++
			y := x
			wg.Add(1)
			go func() {
				defer wg.Done()
				invokeKvIterator("test_kv_iterator", y, contractId, txContext, pool, bytes)

				end := time.Now().UnixNano() / 1e6
				if (end-start)/1000 > 0 && y%1000 == 0 {
					fmt.Printf("【tps】 %d 【spend】%d i = %d, count=%d \n", int(y)/int((end-start)/1000), end-start, i+1, y)
				}
			}()
		}
		wg.Wait()
	}
	end = time.Now().UnixNano() / 1e6
	println("end2 ", start, end, end-start, end, (end-start)/500) // 2.9m

	runtime.GC()
}
func invokeKvIterator(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	parameters := make(map[string]string)
	parameters["key"] = "key"
	parameters["field"] = "field"
	parameters["val"] = "val"
	parameters["start_count"] = "10000"
	parameters["count"] = "20000"

	parameters["start_key"] = "key"
	parameters["start_field"] = "field10000"
	parameters["limit_key"] = "key"
	parameters["limit_field"] = "field20000"

	baseParam(parameters)
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	//fmt.Println("【result】", r)
}
