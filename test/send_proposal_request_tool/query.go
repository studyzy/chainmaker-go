/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/spf13/cobra"
)

func QueryCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query",
		Long:  "Query",
		RunE: func(_ *cobra.Command, _ []string) error {
			return query()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&pairsString, "pairs", "a", "[{\"key\":\"key\",\"value\":\"counter1\"}]", "specify pairs")
	flags.StringVarP(&pairsFile, "pairs-file", "A", "./pairs.json", "specify pairs file, if used, set --pairs=\"\"")
	flags.StringVarP(&method, "method", "m", "increase", "specify contract method")
	flags.Int32VarP(&runTime, "run-time", "", int32(commonPb.RuntimeType_GASM), "run-time")
	flags.StringVarP(&abiPath, "abi-path", "", "", "specify wasm path")

	return cmd
}

func returnResult(code commonPb.TxStatusCode, message string, contractCode uint32, contractMessage string, data string) error {
	var result *Result
	result = &Result{
		Code:                  code,
		Message:               message,
		ContractResultCode:    contractCode,
		ContractResultMessage: contractMessage,
		ContractQueryResult:   data,
	}
	fmt.Println(result.ToJsonString())
	return nil
}

func query() error {
	// 构造Payload
	if pairsString == "" {
		bytes, err := ioutil.ReadFile(pairsFile)
		if err != nil {
			panic(err)
		}
		pairsString = string(bytes)
	}
	var pairs []*commonPb.KeyValuePair
	err := json.Unmarshal([]byte(pairsString), &pairs)
	if err != nil {
		return err
	}

	method_bck := method
	var abiData *[]byte
	method, pairs, err = makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)
	if err != nil {
		err = returnResult(1, "make pairs failure!", 0, "error", "")
		return err
	}
	////暂时不支持传参
	//if commonPb.RuntimeType(runTime) == commonPb.RuntimeType_EVM {
	//	abiJsonData, err := ioutil.ReadFile(abiPath)
	//	//fmt.Println("abiPath : ", abiPath, " ---> abiJsonData: ", abiJsonData)
	//	if err != nil {
	//		return err
	//	}
	//	myAbi, _ := abi.JSON(strings.NewReader(string(abiJsonData)))
	//
	//	dataByte, err := myAbi.Pack(method)
	//	data := hex.EncodeToString(dataByte)
	//	method = data[0:8]
	//	pairs = []*commonPb.KeyValuePair{
	//		{
	//			Key:   "data",
	//			Value: []byte(data),
	//		},
	//	}
	//}

	payloadBytes, err := constructQueryPayload(chainId, contractName, method, pairs)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, payloadBytes)
	if err != nil {
		return err
	}
	log.DebugDynamic(func() string {
		respJson, _ := json.Marshal(resp)
		return string(respJson)
	})
	var dataByte []interface{}
	//var result *Result
	//暂时不支持传参
	if commonPb.RuntimeType(runTime) == commonPb.RuntimeType_EVM {
		abiJsonData, err := ioutil.ReadFile(abiPath)
		//fmt.Println("abiPath : ", abiPath, " ---> abiJsonData: ", abiJsonData)
		if err != nil {
			return err
		}
		myAbi, _ := abi.JSON(strings.NewReader(string(abiJsonData)))
		if resp.ContractResult != nil {
			dataByte, _ = myAbi.Unpack(method_bck, resp.ContractResult.Result)
			//fmt.Println("resp.ContractResult.Result: ", resp.ContractResult.Result, "dataByte: ", dataByte, "type(dataByte): ", reflect.TypeOf(dataByte), "dataByte[0]:", dataByte[0])
		}
		var datas []string
		for _, data := range dataByte {
			datas = append(datas, getStrval(data))
		}

		if resp.Code == commonPb.TxStatusCode_SUCCESS {
			if len(datas) == 0 {
				err = returnResult(resp.Code, resp.Message, resp.ContractResult.Code, resp.ContractResult.Message, "")
			} else if len(datas) == 1 {
				err = returnResult(resp.Code, resp.Message, resp.ContractResult.Code, resp.ContractResult.Message, datas[0])
			} else {
				jsonStr, err := json.Marshal(datas)
				if err != nil {
					return err
				}
				err = returnResult(resp.Code, resp.Message, resp.ContractResult.Code, resp.ContractResult.Message, string(jsonStr))
			}
		} else {
			err = returnResult(resp.Code, resp.Message, resp.ContractResult.Code, resp.ContractResult.Message, "")
		}
		return err
	} else {
		err = returnResult(resp.Code, resp.Message, resp.ContractResult.Code, resp.ContractResult.Message, string(resp.ContractResult.Result))
		return err
	}
	return nil
}
