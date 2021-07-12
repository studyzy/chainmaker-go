/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package gasmtest

//import (
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"sync"
//	"testing"
//	"time"
//
//	"chainmaker.org/chainmaker-go/gasm"
//	commonPb "chainmaker.org/chainmaker/pb-go/common"
//	"chainmaker.org/chainmaker/protocol"
//	"chainmaker.org/chainmaker-go/vm/test"
//	"github.com/stretchr/testify/require"
//)
//
//type person struct {
//	Name string `json:"name"`
//	Age  int64  `json:"age"`
//}
//
//type something struct {
//	Num1    int64    `json:"num1"`
//	Num2    int64    `json:"num2"`
//	Str1    string   `json:"str1"`
//	Str2    string   `json:"str2"`
//	Persons []person `json:"persons"`
//}
//
//func Test_invoke_c(t *testing.T) {
//	runtimeInstance := &gasm.RuntimeInstance{ChainId: "Chain1"}
//
//	byteCode, err := ioutil.ReadFile("C:\\workspace\\chainmaker-go\\module\\vm\\sdk\\c\\counter-c.wasm")
//	require.NoError(t, err)
//
//	contractId := &commonPb.Contract{
//		ContractName:    "",
//		ContractVersion: "",
//		RuntimeType:     0,
//	}
//	method := "dump"
//	count := 1000
//	start := time.Now()
//	var wg sync.WaitGroup
//	for i := 0; i < count; i++ {
//		wg.Add(1)
//		go func() {
//
//			parameters := make(map[string]string)
//			parameters[protocol.ContractCreatorOrgIdParam] = "CREATOR_ORG_ID"
//			parameters[protocol.ContractCreatorRoleParam] = "CREATOR_ROLE"
//			parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
//			parameters[protocol.ContractSenderOrgIdParam] = "SENDER_ORG_ID"
//			parameters[protocol.ContractSenderRoleParam] = "SENDER_ROLE"
//			parameters[protocol.ContractSenderPkParam] = "SENDER_PK"
//			parameters[protocol.ContractBlockHeightParam] = "1"
//			parameters[protocol.ContractTxIdParam] = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
//			parameters[protocol.ContractContextPtrParam] = "0"
//
//			parameters["name"] = "微芯"
//			parameters["num"] = "100"
//			parameters["num1"] = "165746187348471046811"
//			parameters["num2"] = "165746187348471046812"
//			parameters["str"] = "测试"
//
//			p1 := person{"Alice", 22}
//			p2 := person{"Bob", 33}
//			var sth something
//			//only support 0~(1e9-1) due to the bugs(features?) of strconv package in tinygo
//			sth.Num1 = 999999999
//			sth.Num2 = 0
//			sth.Str1 = "I'm the first string."
//			sth.Str2 = ""
//			sth.Persons = []person{p1, p2}
//			sthString, err := json.Marshal(sth)
//			require.NoError(t, err)
//			parameters["sth"] = string(sthString)
//
//			txSimContext := &test.TxContextMockTest{}
//			runtimeInstance.Invoke(contractId, method, byteCode, parameters, txSimContext, 0)
//			wg.Done()
//		}()
//	}
//	wg.Wait()
//	fmt.Printf("method [%+v], tx count %+v, time used %+v\n", method, count, time.Since(start))
//}
//
//func Test_invoke_go(t *testing.T) {
//
//	runtimeInstance := &gasm.RuntimeInstance{ChainId: "Chain1"}
//
//	byteCode, err := ioutil.ReadFile("C:\\workspace\\chainmaker-go\\module\\vm\\sdk\\go\\counter-go.wasm")
//	require.NoError(t, err)
//	contractId := &commonPb.Contract{
//		ContractName:    "",
//		ContractVersion: "",
//		RuntimeType:     0,
//	}
//	method := "increase"
//	count := 1000
//	start := time.Now()
//	var wg sync.WaitGroup
//	for i := 0; i < count; i++ {
//		wg.Add(1)
//		go func() {
//			txSimContext := &test.TxContextMockTest{}
//
//			parameters := make(map[string]string)
//			parameters[protocol.ContractCreatorOrgIdParam] = "CREATOR_ORG_ID"
//			parameters[protocol.ContractCreatorRoleParam] = "CREATOR_ROLE"
//			parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
//			parameters[protocol.ContractSenderOrgIdParam] = "SENDER_ORG_ID"
//			parameters[protocol.ContractSenderRoleParam] = "SENDER_ROLE"
//			parameters[protocol.ContractSenderPkParam] = "SENDER_PK"
//			parameters[protocol.ContractBlockHeightParam] = "1"
//			parameters[protocol.ContractTxIdParam] = "txid"
//			parameters[protocol.ContractContextPtrParam] = "0"
//
//			parameters["name"] = "微芯"
//			parameters["num"] = "100"
//			parameters["num1"] = "165746187348471046811"
//			parameters["num2"] = "165746187348471046812"
//			parameters["str"] = "测试"
//
//			p1 := person{"Alice", 22}
//			p2 := person{"Bob", 33}
//			var sth something
//			//only support 0~(1e9-1) due to the bugs(features?) of strconv package in tinygo
//			sth.Num1 = 999999999
//			sth.Num2 = 0
//			sth.Str1 = "I'm the first string."
//			sth.Str2 = ""
//			sth.Persons = []person{p1, p2}
//			sthString, err := json.Marshal(sth)
//			require.NoError(t, err)
//			parameters["sth"] = string(sthString)
//
//			runtimeInstance.Invoke(contractId, method, byteCode, parameters, txSimContext, 0)
//			wg.Done()
//		}()
//	}
//	wg.Wait()
//	fmt.Printf("method [%+v], tx count %+v, time used %+v\n", method, count, time.Since(start))
//}
