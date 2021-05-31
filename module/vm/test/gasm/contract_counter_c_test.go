/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package gasmtest

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/gasm"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
)

func TestContract_Counter(t *testing.T) {
	test.WasmFile = "D:/develop/workspace/chainMaker/chainmaker-go/module/vm/sdk/c/counter-c.wasm"
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
				//invokeCallContractCallContract("increase", int32(i), contractId, txContext, byteCode)
				//invokeCallContractCallContract("query", int32(i), contractId, txContext, byteCode)
				invokeCallContractCallContract("dump", int32(i), contractId, txContext, byteCode)
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

func invokeCallContractCallContract(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	parameters["contract_name"] = test.ContractNameTest
	parameters["method"] = "query"
	parameters[protocol.ContractTxIdParam] = parameters["tx_id"]

	runtimeInstance := &gasm.RuntimeInstance{}
	runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
