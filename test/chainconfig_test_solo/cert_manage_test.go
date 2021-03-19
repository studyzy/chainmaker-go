/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// description: chainmaker-go
//
// @author: xwc1125
// @date: 2020/11/24
package native_test

import (
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"fmt"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"

	//"github.com/gogo/protobuf/proto"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 证书添加，个人添加自己的证书
func TestCertAdd(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERT_ADD.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书的删除（管理员操作）
func TestCertDelete(t *testing.T) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes",
		Value: "03725dc03b236f098153adea0fdf9a09dfe67fc8606a9ee1be7075c22e209a08",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_DELETE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书查询
func TestCertQuery(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight============")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes",
		Value: "03725dc03b236f098153adea0fdf9a09dfe67fc8606a9ee1be7075c22e209a08",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书查询
func TestCertQueryWithCertId(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight============")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes",
		Value: "03725dc03b236f098153adea0fdf9a09dfe67fc8606a9ee1be7075c22e209a08",
	})

	sk, _ := native.GetUserSK(1)
	resp, err := native.QueryRequestWithCertID(sk, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}

func processReq(txId string, txType commonPb.TxType, contractName, funcName string, pairs []*commonPb.KeyValuePair, sequence uint64) {
	sk, member := native.GetUserSK(1)
	resp, err := native.ConfigUpdateRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: txType, ChainId: CHAIN1,
		ContractName: contractName, MethodName: funcName, Pairs: pairs}, sequence)
	processResults(resp, err)
}

func processResults(resp *commonPb.TxResponse, err error) {
	if err == nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println("WARN: client.call err: deadline")
		return
	}
	fmt.Printf("ERROR: client.call err: %v\n", err)
}
