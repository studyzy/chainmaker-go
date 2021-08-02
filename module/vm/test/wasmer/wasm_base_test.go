/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmertest

import (
	"chainmaker.org/chainmaker-go/vm/test"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"fmt"
	"net/http"
	"runtime/debug"
	"testing"

	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
	"runtime"
	"sync"
	"time"
)

// Module 序列化后实例wasm
// 经测试证明 序列化反序列化方式Instantiate慢200倍
//func TestSerializationModuleSpendTest(t *testing.T) {
//	byteCode, _ := wasm.ReadBytes("D:/develop/workspace/chainMaker/chainmaker-contract-sdk-rust/target/wasm32-unknown-unknown/release/chainmaker_contract.wasm")
//	module1, _ := wasm.Compile(byteCode)
//
//	serialization, _ := module1.Serialize()
//	module1.Close()
//
//	start := time.Now().UnixNano() / 1e6
//	for i := 0; i < 10000; i++ {
//		module2, _ := wasm.DeserializeModule(serialization)
//		vm := wasmer.GetVmBridgeManager()
//		module2.InstantiateWithImports(vm.GetImports()) // 44832ms
//		module2.Close()
//	}
//
//	end := time.Now().UnixNano() / 1e6
//	println("【spend】", end-start)
//}

// Module 直接实例wasm
//func TestModuleSpendTest(t *testing.T) {
//	byteCode, _ := wasm.ReadBytes("D:/develop/workspace/chainMaker/chainmaker-contract-sdk-rust/target/wasm32-unknown-unknown/release/chainmaker_contract.wasm")
//	module1, _ := wasm.Compile(byteCode)
//
//	start := time.Now().UnixNano() / 1e6
//	for i := 0; i < 10000; i++ {
//		vm := wasmer.GetVmBridgeManager()
//		module1.InstantiateWithImports(vm.GetImports()) // 643ms
//	}
//
//	end := time.Now().UnixNano() / 1e6
//	println("【spend】", end-start)
//}



func TestInstanceNewAndClose(t *testing.T) {
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal(err)
		}
	}()
	var beginMemStat runtime.MemStats
	var endMemStat runtime.MemStats

	parameters := make(map[string][]byte)
	parameters["time"] = []byte("1314520")
	parameters["file_hash"] = []byte("file_hash")
	parameters["file_name"] = []byte("file_name")
	parameters["contract_name"] = []byte(test.ContractNameTest)
	baseParam(parameters)

	test.ContractNameTest = "contract_fact"
	test.WasmFile = "../../../../test/wasm/rust-func-verify-2.0.0.wasm"
	contractId, _, bytesCode := test.InitContextTest(commonPb.RuntimeType_WASMER)

	// 获取全局唯一的 pool manager
	poolManager := test.GetVmPoolManager()
	// 获取智能合约对应的 RuntimeInstance（对象池）
	runtimeInstance, _ := poolManager.NewRuntimeInstance(contractId, bytesCode)
	//runtimeInstance.Pool().NewInstance()

	println("test begin ==> ")
	runtime.ReadMemStats(&beginMemStat)
	time.Sleep(time.Second * 5)

	wg := sync.WaitGroup{}
	for i := 0; i < 100000; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			// 构建智能合约的 wrappedInstance, 不进入对象池
			wrappedInstance, err := runtimeInstance.Pool().NewInstance()
			if err != nil {
				t.Fatalf("newInstance() error: %v", err)
			}
			// 关闭 wrappedInstrance
			defer runtimeInstance.Pool().CloseInstance(wrappedInstance)

			if err != nil {
				panic(err)
			}
			fmt.Println(i)
		}()
		wg.Wait()
	}
	runtime.ReadMemStats(&endMemStat)
	runtime.GC()
	debug.FreeOSMemory()
	println("finshed, waiting exit...")


	time.Sleep(time.Second * 10)
	fmt.Printf("begin stat =>\t Alloc = %v, TotalAlloc = %v, Sys = %v, NumGC = %v\n",
		beginMemStat.Alloc/1024, beginMemStat.TotalAlloc/1024, beginMemStat.Sys/1024, beginMemStat.NumGC)
	fmt.Printf("end stat   =>\t Alloc = %v, TotalAlloc = %v, Sys = %v, NumGC = %v\n",
		endMemStat.Alloc/1024, endMemStat.TotalAlloc/1024, endMemStat.Sys/1024, endMemStat.NumGC)
}