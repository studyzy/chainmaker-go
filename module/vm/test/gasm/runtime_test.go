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
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
)

func TestContract_Fact(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/go-fact-1.2.1.wasm"
	//test.WasmFile = "../../../../test/wasm/go-func-verify-1.2.1.wasm"
	//test.WasmFile = "D:/develop/workspace/chainMaker/chainmaker-go/module/vm/sdk/go/fact-go.wasm"
	contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_GASM)

	if len(byteCode) == 0 {
		panic("error byteCode==0")
	}
	start := time.Now().UnixNano() / 1e6
	x := 0
	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		for j := 0; j < 1; j++ {
			x += 1
			y := int32(x)
			wg.Add(1)
			go func() {
				defer wg.Done()
				invokeCallContractTestSave("save", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("find_by_file_hash", int32(i), contractId, txContext, byteCode)
				//invokeCallContractTestSave("functional_verify", int32(i), contractId, txContext, byteCode)
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

func TestContract_Kv(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/go-func-verify-1.2.1.wasm"
	//test.WasmFile = "D:/develop/workspace/chainMaker/chainmaker-go/module/vm/sdk/go/fact-go.wasm"
	contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_GASM)

	if len(byteCode) == 0 {
		panic("error byteCode==0")
	}
	invokeCallContractTestSave("test_put_state", 0, contractId, txContext, byteCode)
	r := invokeCallContractTestSave("test_kv_iterator", 0, contractId, txContext, byteCode)
	assert.Equal(t, "15", string(r.Result))
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
	fmt.Println("【result】", r)
	return r
}
