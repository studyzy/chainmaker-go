/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/utils/v2"

	"github.com/spf13/cobra"
)

func InvokeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Invoke",
		Long:  "Invoke",
		RunE: func(_ *cobra.Command, _ []string) error {
			return invoke()
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

func invoke() error {
	txId := ""

	// 构造Payload
	if pairsString == "" {
		bytes, err := ioutil.ReadFile(pairsFile)
		if err != nil {
			panic(err)
		}
		pairsString = string(bytes)
	}
	//var pairs []*commonPb.KeyValuePair
	//err := json.Unmarshal([]byte(pairsString), &pairs)
	pairs, err := utils.UnmarshalJsonStrKV2KVPairs(pairsString)
	if err != nil {
		return err
	}

	testCode := commonPb.TxStatusCode_SUCCESS
	var testMessage string
	var abiData *[]byte
	method, pairs, err = makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)
	if err != nil {
		testCode = commonPb.TxStatusCode_CONTRACT_FAIL
		testMessage = "make pairs filure!"
	} else {
		fmt.Println("pairs: ", pairs, ", method: ", method)

		payloadBytes, err := constructInvokePayload(chainId, contractName, method, pairs)
		if err != nil {
			return err
		}

		resp, err = proposalRequest(sk3, client, payloadBytes)
		if err != nil {
			return err
		}
		testCode = resp.Code
		testMessage = resp.Message
		txId = resp.TxId
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

	result := &Result{
		Code:    testCode,
		Message: testMessage,
		TxId:    txId,
	}
	fmt.Println(result.ToJsonString())

	return nil
}
