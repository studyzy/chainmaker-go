///*
//Copyright (C) BABEC. All rights reserved.
//Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
//SPDX-License-Identifier: Apache-2.0
//*/
//
//// description: chainmaker-go
////
//// @author: xwc1125
//// @date: 2020/12/21
package native

//
//import (
//	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
//	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
//	consensusPb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
//	"fmt"
//	"testing"
//
//	"github.com/stretchr/testify/require"
//
//	"github.com/gogo/protobuf/proto"
//	"google.golang.org/grpc/codes"
//	"google.golang.org/grpc/status"
//)
//
////查询治理相关配置
//func TestGetGovernmentContract(t *testing.T) {
//	conn, err := InitGRPCConnect(isTls)
//	require.NoError(t, err)
//	client := apiPb.NewRpcNodeClient(conn)
//
//	fmt.Println("============ get chain config ============")
//	// 构造Payload
//	var pairs []*commonPb.KeyValuePair
//
//	sk, member := GetUserSK(1)
//	resp, err := QueryRequest(sk, member, &client, &InvokeContractMsg{TxType: commonPb.TxType_QUERY_SYSTEM_CONTRACT, ChainId: "chain1",
//		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), MethodName: "GET_GOVERNMENT_CONTRACT", Pairs: pairs})
//	if err == nil {
//		result := &consensusPb.GovernmentContract{}
//		err = proto.Unmarshal(resp.ContractResult.Result, result)
//		fmt.Printf("send tx resp: code:%d, msg:%s, chainConfig:%+v\n", resp.Code, resp.Message, result)
//		return
//	}
//	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
//		fmt.Println("WARN: client.call err: deadline")
//		return
//	}
//	fmt.Printf("ERROR: client.call err: %v\n", err)
//}
