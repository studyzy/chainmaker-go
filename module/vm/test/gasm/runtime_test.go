/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package gasmtest

import (
	"fmt"
	"gotest.tools/assert"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/gasm"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
)

func TestContract_Fact(t *testing.T) {
	//test.WasmFile = "../../../../test/wasm/go-fact-1.2.0.wasm"
	test.WasmFile = "../../../../test/wasm/go-func-verify-1.2.0.wasm"
	//test.WasmFile = "D:/develop/workspace/chainMaker/chainmaker-go/module/vm/sdk/go/fact-go.wasm"
	contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_GASM)

	if len(byteCode) == 0 {
		panic("error byteCode==0")
	}
	start := time.Now().UnixNano() / 1e6
	x := 0
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			x += 1
			y := int32(x)
			wg.Add(1)
			go func() {
				defer wg.Done()
				invokeCallContractTestSave("increase", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("query", int32(i), contractId, txContext, byteCode)
				end := time.Now().UnixNano() / 1e6
				if (end-start)/1000 > 0 && y%100 == 0 {
					fmt.Printf("【tps】 %d 【spend】%d i = %d, count=%d \n", int(y)/int((end-start)/1000), end-start, i+1, y)
				}
			}()
		}
		wg.Wait()
	}

	end := time.Now().UnixNano() / 1e6
	println("end 【spend】", end-start)
	//time.Sleep(time.Second * 5) // 73m
}

func invokeCallContractTestSave(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	parameters["app_id"] = "app_id"
	parameters["file_hash"] = "staticVal2"
	parameters["file_name"] = "staticVal3"
	parameters["contract_name"] = test.ContractNameTest
	//parameters["method"] = "save"
	parameters["time"] = "12"

	runtimeInstance := &gasm.RuntimeInstance{
		Log: logger.GetLogger(logger.MODULE_VM),
	}
	r := runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	fmt.Printf("\n【result】 %+v \n\n\n", r)
	return r
}

func TestFunctionalContract(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/go-func-verify-1.2.0.wasm"
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_GASM)

	invokeFunctionalContract("init_contract", contractId, txContext, bytes)
	invokeFunctionalContract("upgrade", contractId, txContext, bytes)

	invokeFunctionalContract("save", contractId, txContext, bytes)
	r := invokeFunctionalContract("find_by_file_hash", contractId, txContext, bytes)
	assert.Equal(t, string(r.Result), "{\"file_hash\":\"file_hash\",\"file_name\":\"file_name\",\"time\":\"1314520\"}")
	fmt.Println("  【save】pass")
	fmt.Println("  【find_by_file_hash】pass")

	invokeFunctionalContract("test_put_pre_state", contractId, txContext, bytes)
	r2 := invokeFunctionalContract("test_iter_pre_field", contractId, txContext, bytes)
	r3 := invokeFunctionalContract("test_iter_pre_key", contractId, txContext, bytes)
	assert.Equal(t, string(r2.Result), "14")
	assert.Equal(t, string(r3.Result), "14")
	fmt.Println("  【test_put_pre_state】pass")
	fmt.Println("  【test_iter_pre_field】pass")
	fmt.Println("  【test_iter_pre_key】pass")

	invokeFunctionalContract("test_put_state", contractId, txContext, bytes)
	r4 := invokeFunctionalContract("test_kv_iterator", contractId, txContext, bytes)
	assert.Equal(t, string(r4.Result), "15")
	fmt.Println("  【test_put_state】pass")
	fmt.Println("  【test_kv_iterator】pass")

	invokeFunctionalContract("increase", contractId, txContext, bytes)
	r5 := invokeFunctionalContract("query", contractId, txContext, bytes)
	assert.Equal(t, string(r5.Result), "1")
	fmt.Println("  【increase】pass")
	fmt.Println("  【query】pass")

	r6 := invokeFunctionalContract("functional_verify", contractId, txContext, bytes)
	assert.Equal(t, string(r6.Result), "ok")
	fmt.Println("  【functional_verify】pass")
	fmt.Println("  【test】pass")
}

func invokeFunctionalContract(method string, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) *commonPb.ContractResult {
	parameters := make(map[string]string)
	parameters["time"] = "1314520"
	parameters["file_hash"] = "file_hash"
	parameters["file_name"] = "file_name"
	parameters["contract_name"] = test.ContractNameTest
	test.BaseParam(parameters)

	runtimeInstance := &gasm.RuntimeInstance{
		Log: logger.GetLogger(logger.MODULE_VM),
	}
	r := runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	fmt.Printf("\n【result】 %+v \n\n\n", r)
	return r
}
