package wasmertest

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	"fmt"
	"runtime"
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
				invokeFact("save", y, contractId, txContext, pool, bytes)
				invokeFact("find_by_file_hash", y, contractId, txContext, pool, bytes)
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
	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
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
