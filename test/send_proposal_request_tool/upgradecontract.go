/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
)

func UpgradeContractCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgradeContract",
		Short: "Upgrade Contract",
		Long:  "Upgrade Contract",
		RunE: func(_ *cobra.Command, _ []string) error {
			return upgradeContract()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&wasmPath, "wasm-path", "w", "../wasm/counter-go.wasm", "specify wasm path")
	flags.Int32VarP(&runTime, "run-time", "r", int32(commonPb.RuntimeType_GASM), "specify run time")
	flags.StringVarP(&version, "version", "v", "2.0.0", "specify contract version")
	flags.StringVarP(&abiPath, "abi-path", "", "", "specify wasm path")
	flags.StringVarP(&pairsString, "pairs", "", "", "specify pairs")
	flags.StringVarP(&pairsFile, "pairs-file", "", "", "specify pairs file, if used, set --pairs=\"\"")

	return cmd
}

func upgradeContract() error {
	txId := utils.GetRandTxId()

	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return err
	}

	// 构造Payload
	if pairsString == "" {
		bytes, err := ioutil.ReadFile(pairsFile)
		if err != nil {
			panic(err)
		}
		pairsString = string(bytes)
	}
	var pairs []*commonPb.KeyValuePair
	err = json.Unmarshal([]byte(pairsString), &pairs)
	if err != nil {
		return err
	}
	var abiData *[]byte
	method, pairs, err = makePairs("", abiPath, pairs, commonPb.RuntimeType(runTime), abiData)
	if err != nil {
		return fmt.Errorf("make pairs filure!")
	}
	if commonPb.RuntimeType(runTime) == commonPb.RuntimeType_EVM {
		wasmBin, err = hex.DecodeString(string(wasmBin))
	}

	//
	//if commonPb.RuntimeType(runTime) == commonPb.RuntimeType_EVM {
	//
	//	data := ""
	//	//对于参数的处理
	//	if initParams != "" {
	//		abiJsonData, err := ioutil.ReadFile(abiPath)
	//		if err != nil {
	//			return err
	//		}
	//		myAbi, _ := abi.JSON(strings.NewReader(string(abiJsonData)))
	//		//addr := evm.BigToAddress(evm.FromDecimalString(initParams))
	//		dataByte, err := myAbi.Pack("", big.NewInt(3), big.NewInt(2))
	//		if err != nil {
	//			return err
	//		}
	//		data = hex.EncodeToString(dataByte)
	//	}
	//	pairs = []*commonPb.KeyValuePair{
	//		{
	//			Key:   "data",
	//			Value: []byte(data),
	//		},
	//	}
	//	wasmBin, err = hex.DecodeString(string(wasmBin))
	//
	//}
	payload, _ := GenerateUpgradeContractPayload(contractName, version, commonPb.RuntimeType(runTime), wasmBin, pairs)
	//method := syscontract.ContractManageFunction_UPGRADE_CONTRACT.String()
	//
	//payload := &commonPb.Payload{
	//	ChainId: chainId,
	//	Contract: &commonPb.Contract{
	//		ContractName:    contractName,
	//		ContractVersion: version,
	//		RuntimeType:     commonPb.RuntimeType(runTime),
	//	},
	//	Method:      method,
	//	Parameters:  pairs,
	//	ByteCode:    wasmBin,
	//	Endorsement: nil,
	//}

	//if endorsement, err := acSign(payload); err == nil {
	//	payload.Endorsement = endorsement
	//} else {
	//	return err
	//}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err = proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	if err != nil {
		return err
	}

	result := &Result{
		Code:    resp.Code,
		Message: resp.Message,
		TxId:    txId,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
