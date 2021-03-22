package wasmertest

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
)

var log = logger.GetLoggerByChain(logger.MODULE_VM, test.ChainIdTest)

// 存证合约 单例需要大于65536次，因为内存是64K
func TestCallFact(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/rust-functional-verify-1.0.0.wasm"
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
				invokeFact("functional_verify", y, contractId, txContext, pool, bytes)

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
	parameters["time"] = txId
	parameters["file_hash"] = "file_hash"
	parameters["file_name"] = txId
	parameters["tx_id"] = txId
	parameters["forever"] = "true"
	parameters["contract_name"] = test.ContractNameTest

	baseParam(parameters)
	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func TestCallCounter(t *testing.T) {
	//{
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	//bytes, _ = wasm.ReadBytes("../../../../test/wasm/counter.wasm")
	println("bytes len", len(bytes))

	pool := wasmer.NewVmPoolManager("chain001")

	// 调用
	key := "counter001"
	x := 0
	println("start") // 2.9m
	wg := sync.WaitGroup{}
	start := time.Now().UnixNano() / 1e6
	for i := 0; i < 2000; i++ {
		for j := 0; j < 100; j++ {
			x += 1
			y := x
			wg.Add(1)
			go func() {
				defer wg.Done()
				//invokeCounter("increase", key, contractId, txContext, pool, bytes)
				//invokeCounter("test_verify_signature", key, contractId, txContext, pool, bytes)
				//invokeCounter("test_marshal_unmarshal", key, contractId, txContext, pool, bytes)
				invokeCounter("query", key, contractId, txContext, pool, bytes)
				end := time.Now().UnixNano() / 1e6
				if y%1000 == 0 && (end-start)/1000 > 0 {
					fmt.Printf("【tps】 %d 【spend】%d i = %d, j = %d count=%d\n", y/int((end-start)/1000), end-start, i+1, j, y)
				}
			}()
		}
		wg.Wait()
		//time.Sleep(time.Millisecond * 20)
	}

	end := time.Now().UnixNano() / 1e6
	println("end 【spend】", end-start)
	time.Sleep(time.Second * 5) // 73m
	//pool.resetPool()            // 10000*10:73m->63m 1000*10:44->33m 10000*50:281->238m  1000*50:106->75m
	//time.Sleep(time.Second * 3) // 73m
	//CleanMap() // 无用
	//println("gc")
	//runtime.GC()
	//time.Sleep(time.Second * 2)
	//} // 3m
	runtime.GC() // 无用，未回收内存
	println("gc2")
	time.Sleep(time.Second * 20000)
}
func invokeCounter(method string, key string, contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	parameters := make(map[string]string)
	parameters["key"] = key
	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func TestCallTraceability(t *testing.T) {
	contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_WASMER)
	byteCode, _ = wasm.ReadBytes("D:\\develop\\workspace\\chainMaker\\chainmaker-go\\test\\wasm\\traceability.wasm")
	start := time.Now().UnixNano() / 1e6

	pool := wasmer.NewVmPoolManager("chain001")

	var (
		method     string = "init"
		parameters map[string]string
	)
	parameters = make(map[string]string)

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)

	name := "apple_000001"
	category := "fruits"
	// 注册
	log.Infof("注册")
	invokeTraceability("register", category, name, contractId, txContext, pool, byteCode)
	// 重复注册
	log.Infof("重复注册")
	invokeTraceability("register", category, name, contractId, txContext, pool, byteCode)

	// 查询
	log.Infof("查询")
	invokeTraceability("query", category, name, contractId, txContext, pool, byteCode)

	// 进海关
	log.Infof("海关")
	invokeTraceability("customs", category, name, contractId, txContext, pool, byteCode)
	log.Infof("海关")
	invokeTraceability("customs", category, name, contractId, txContext, pool, byteCode)

	// 进北京
	log.Infof("北京")
	invokeTraceability("beijing", category, name, contractId, txContext, pool, byteCode)

	// 进海淀
	log.Infof("海淀")
	invokeTraceability("haidian", category, name, contractId, txContext, pool, byteCode)

	log.Infof("查询")
	invokeTraceability("query", category, name, contractId, txContext, pool, byteCode)

	name = "apple_002"
	// 注册
	log.Infof("注册")
	invokeTraceability("register", category, name, contractId, txContext, pool, byteCode)
	invokeTraceability("register", category, name, contractId, txContext, pool, byteCode)
	// 未入海关直接进海淀
	log.Infof("跳跃")
	//invokeTraceability("beijing", category, name, contractId, txContext, pool, byteCode)

	log.Infof("查询")
	invokeTraceability("query", category, name, contractId, txContext, pool, byteCode)

	x := 0
	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func() {
				invokeTraceability("query", category, name, contractId, txContext, pool, byteCode)
				end := time.Now().UnixNano() / 1e6
				fmt.Printf("【spend】%d i = %d, j = %d count=%d\n", end-start, i, j, x)
				x += 1
				wg.Done()
			}()
		}
		wg.Wait()
		//time.Sleep(time.Millisecond * 20)
	}

	end := time.Now().UnixNano() / 1e6
	println("【spend】", end-start, "【tps】", int64(x)/((end-start)/1000))
	time.Sleep(time.Second * 1000)
}

func invokeTraceability(method string, category string, name string, contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	parameters := make(map[string]string)
	parameters["category"] = category
	parameters["name"] = name
	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

// 使用原始调用智能合约
func TestCallHelloWorldUseOrigin(t *testing.T) {
	_, _, byteCode := test.InitContextTest(commonPb.RuntimeType_WASMER)
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
	greetResult, _ := instance.Exports["invoke"](inputPointer, lengthOfSubject)
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

// 使用pool调用智能合约
func TestCallHelloWorldUsePool(t *testing.T) {
	contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_WASMER)

	start := time.Now().UnixNano() / 1e6
	time.Sleep(time.Second * 1)

	pool := wasmer.NewVmPoolManager("chain001")

	// 创建
	parameters := make(map[string]string)
	runtimeInstance, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtimeInstance.Invoke(contractId, "init", byteCode, parameters, txContext, 0)

	// 调用
	y := 0
	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		for j := 0; j < 100; j++ {
			wg.Add(1)
			x := y
			go func() {
				invokeCounter("increase", "key", contractId, txContext, pool, byteCode)
				end := time.Now().UnixNano() / 1e6
				fmt.Printf("【tps】 %d【spend】%d i = %d, j = %d count=%d\n ", int64(x)/((end-start)/1000), end-start, i, j, x)
				wg.Done()
			}()
			y += 1
		}
		wg.Wait()
		//time.Sleep(time.Millisecond * 20)
	}

	end := time.Now().UnixNano() / 1e6
	println("【spend】", end-start, "【tps】", int64(y)/((end-start)/1000))
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
