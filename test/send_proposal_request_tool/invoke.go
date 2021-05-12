/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
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
	flags.StringVarP(&abiPath, "api-path", "", "", "specify wasm path")

	return cmd
}

func invoke() error {
	txId := utils.GetRandTxId()

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

	testCode := commonPb.TxStatusCode_SUCCESS
	var testMessage string
	method, pairs, err = makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime))
	if err != nil {
		testCode = commonPb.TxStatusCode_CONTRACT_FAIL
		testMessage = "make pairs filure!"
	} else {
		payloadBytes, err := constructPayload(contractName, method, pairs)
		if err != nil {
			return err
		}

		resp, err = proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
			chainId, txId, payloadBytes)
		if err != nil {
			return err
		}
		testCode = resp.Code
		testMessage = resp.Message
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
	//			Value: data,
	//		},
	//	}
	//}



	result := &Result{
		Code:    testCode,
		Message: testMessage,
		TxId:    txId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
