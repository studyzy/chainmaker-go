/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native_test

import (
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"encoding/hex"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

// 查询区块
func TestGetBlockByHeight(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get block by height============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: "0",
		},
		{
			Key:   "withRWSet",
			Value: "false",
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), MethodName: commonPb.QueryFunction_GET_BLOCK_BY_HEIGHT.String(), Pairs: pairs})

	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.DeadlineExceeded {
				fmt.Println("WARN: client.call err: deadline")

			}
		}

		fmt.Printf("ERROR: client.call err: %v\n", err)
		return
	}
	fmt.Printf("response: %v\n", resp)
	//result := &commonPb.CertInfos{}
	//err = proto.Unmarshal(resp.ContractResult.Result, result)
	//fmt.Printf("send tx resp: code:%d, msg:%s, CertInfos:%+v\n", resp.Code, resp.Message, result)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}
	fmt.Println(blockInfo)

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))
}

// 查询区块
func TestGetBlockByHash(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get block by height============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: "54d54331b4a341353c19b82ec7ad4a6f15b78c9cc4ba8caa84759d1805f4ad1f",
		},
		{
			Key:   "withRWSet",
			Value: "false",
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), MethodName: commonPb.QueryFunction_GET_BLOCK_BY_HASH.String(), Pairs: pairs})

	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.DeadlineExceeded {
				fmt.Println("WARN: client.call err: deadline")

			}
		}

		fmt.Printf("ERROR: client.call err: %v\n", err)
		return
	}
	fmt.Printf("response: %v\n", resp)

	blockInfo := &commonPb.BlockInfo{}
	if err = proto.Unmarshal(resp.ContractResult.Result, blockInfo); err != nil {
		panic(err)
	}

	blockHash := blockInfo.Block.Header.BlockHash
	fmt.Println("blockHash", string(blockHash), hex.EncodeToString(blockHash))
}
