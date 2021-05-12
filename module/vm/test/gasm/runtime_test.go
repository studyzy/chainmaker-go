/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package gasmtest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	test.WasmFile = "../../sdk/go/test_functional-0.7.2-go.wasm"
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
				invokeCallContractTestSave("function_test", int32(i), contractId, txContext, byteCode)
				//invokeCallContractTestSave("save", int32(i), contractId, txContext, byteCode)
				//invokeCallContractTestSave("find_by_file_hash", int32(i), contractId, txContext, byteCode)
				//invokeFact("increase", int32(i), contractId, txContext, byteCode)
				//invokeFact("query", int32(i), contractId, txContext, byteCode)
				//invokeFact("increase_only_key", int32(i), contractId, txContext, byteCode)
				//invokeFact("query_only_key", int32(i), contractId, txContext, byteCode)
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

func TestContractEvidence(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/counter-go-0.7.2.wasm"
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
				invokeCallContractTestSave("increase", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("query", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("increase", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("query", int32(i), contractId, txContext, byteCode)
				//invokeCallContractTestSave("function_test", int32(i), contractId, txContext, byteCode)
				//invokeCallContractTestSave("save", int32(i), contractId, txContext, byteCode)
				//invokeCallContractTestSave("find_by_file_hash", int32(i), contractId, txContext, byteCode)
				//invokeFact("increase", int32(i), contractId, txContext, byteCode)
				//invokeFact("query", int32(i), contractId, txContext, byteCode)
				//invokeFact("increase_only_key", int32(i), contractId, txContext, byteCode)
				//invokeFact("query_only_key", int32(i), contractId, txContext, byteCode)
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
}

func invokeCallContractTestSave(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	parameters["app_id"] = "app_id"
	//parameters["file_hash"] = "staticVal2"
	//parameters["file_name"] = "staticVal3"
	//parameters["contract_name"] = test.ContractNameTest
	//parameters["method"] = "save"
	//parameters["time"] = "12"

	runtimeInstance := &gasm.RuntimeInstance{
		Log: logger.GetLogger(logger.MODULE_VM),
	}
	runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeCallContractTestQuery(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	parameters["file_hash"] = "staticVal2"
	parameters["file_name"] = "staticVal3"
	parameters["contract_name"] = test.ContractNameTest
	parameters["method"] = "query"
	parameters[protocol.ContractTxIdParam] = parameters["tx_id"]

	runtimeInstance := &gasm.RuntimeInstance{}
	runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeFact(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	t := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	//hash := sha256.Sum256([]byte(parameters["time"]))
	hash2 := sha256.Sum256([]byte("hash"))
	parameters["time"] = t
	parameters["file_hash"] = hex.EncodeToString(hash2[:])
	parameters["file_name"] = fmt.Sprintf("%d-%s.pdf", id, t)
	//parameters["tx_id"] = hex.EncodeToString(hash[:])
	parameters["time"] = "staticVal1"
	parameters["file_hash"] = "staticVal2"
	parameters["file_name"] = "staticVal3"
	parameters["tx_id"] = "staticVal4"
	parameters["contract_name"] = test.ContractNameTest
	parameters["method"] = "save"
	parameters[protocol.ContractTxIdParam] = parameters["tx_id"]

	runtimeInstance := &gasm.RuntimeInstance{}
	runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
