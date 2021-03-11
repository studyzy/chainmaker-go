package wasmertest

import (
	"chainmaker.org/chainmaker-go/wasmer"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
	"testing"
	"time"
	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
)

// Module 序列化后实例wasm
// 经测试证明 序列化反序列化方式Instantiate慢200倍
func TestSerializationModuleSpendTest(t *testing.T) {
	byteCode, _ := wasm.ReadBytes("D:/develop/workspace/chainMaker/chainmaker-contract-sdk-rust/target/wasm32-unknown-unknown/release/chainmaker_contract.wasm")
	module1, _ := wasm.Compile(byteCode)

	serialization, _ := module1.Serialize()
	module1.Close()

	start := time.Now().UnixNano() / 1e6
	for i := 0; i < 10000; i++ {
		module2, _ := wasm.DeserializeModule(serialization)
		vm := wasmer.GetVmBridgeManager()
		module2.InstantiateWithImports(vm.GetImports()) // 44832ms
		module2.Close()
	}

	end := time.Now().UnixNano() / 1e6
	println("【spend】", end-start)
}

// Module 直接实例wasm
func TestModuleSpendTest(t *testing.T) {
	byteCode, _ := wasm.ReadBytes("D:/develop/workspace/chainMaker/chainmaker-contract-sdk-rust/target/wasm32-unknown-unknown/release/chainmaker_contract.wasm")
	module1, _ := wasm.Compile(byteCode)

	start := time.Now().UnixNano() / 1e6
	for i := 0; i < 10000; i++ {
		vm := wasmer.GetVmBridgeManager()
		module1.InstantiateWithImports(vm.GetImports()) // 643ms
	}

	end := time.Now().UnixNano() / 1e6
	println("【spend】", end-start)
}
