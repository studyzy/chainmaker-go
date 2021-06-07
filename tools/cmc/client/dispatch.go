/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	sdk "chainmaker.org/chainmaker-sdk-go"
	sdkPbCommon "chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

func Dispatch(client *sdk.ChainClient, contractName, method string, params map[string]string) {
	var (
		wgSendReq sync.WaitGroup
	)

	for i := 0; i < concurrency; i++ {
		wgSendReq.Add(1)
		go runInvokeContract(client, contractName, method, params, &wgSendReq)
	}

	wgSendReq.Wait()
}
func DispatchTimes(client *sdk.ChainClient, contractName, method string, params map[string]string) {
	var (
		wgSendReq sync.WaitGroup
	)
	times := maxi(1, sendTimes)
	wgSendReq.Add(times)
	txId := GetRandTxId()
	for i := 0; i < times; i++ {
		go runInvokeContractOnce(client, contractName, method, params, &wgSendReq, txId)
	}
	wgSendReq.Wait()
}

func runInvokeContract(client *sdk.ChainClient, contractName, method string, params map[string]string,
	wg *sync.WaitGroup) {

	defer func() {
		wg.Done()
	}()

	for i := 0; i < totalCntPerGoroutine; i++ {
		resp, err := client.InvokeContract(contractName, method, "", params, int64(timeout), syncResult)
		if err != nil {
			fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
			return
		}

		if resp.Code != sdkPbCommon.TxStatusCode_SUCCESS {
			fmt.Printf("[ERROR] invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
			return
		}

		fmt.Printf("INVOKE contract resp, [code:%d]/[msg:%s]/[contractResult:%+v]\n", resp.Code, resp.Message, resp.ContractResult)
	}
}
func runInvokeContractOnce(client *sdk.ChainClient, contractName, method string, params map[string]string,
	wg *sync.WaitGroup, txId string) {

	defer func() {
		wg.Done()
	}()
	resp, err := client.InvokeContract(contractName, method, txId, params, int64(timeout), syncResult)
	if err != nil {
		fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
		return
	}

	if resp.Code != sdkPbCommon.TxStatusCode_SUCCESS {
		fmt.Printf("[ERROR] invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		return
	}

	fmt.Printf("INVOKE contract resp, [code:%d]/[msg:%s]/[contractResult:%+v]\n", resp.Code, resp.Message, resp.ContractResult)

}
