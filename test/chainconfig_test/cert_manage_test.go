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
	"testing"

	"github.com/stretchr/testify/require"

	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 证书添加，个人添加自己的证书
func TestCertAdd(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT,
		ChainId: CHAIN1, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERT_ADD.String()})
	processResults(resp, err)
}

// 证书的删除（管理员操作）
func TestCertDelete(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Printf("\n============ send Tx [%s] ============\n", txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "cert_hashes_1",
			Value: "de536ef8c323ae708586f9486bc3fe8bbba6452ff940a6e69de00db5159e5b1e",
		},
	}
	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_DELETE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书查询
func TestCertQuery(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "cert_hashes",
			Value: "b297d4e74ba0a88f9d154d63e53f2bf57116e09b6b8d2a718c24e59175f74cbe",
		},
	}
	sk, member := native.GetUserSK(1)
	resp, err := native.QueryRequest(sk, member, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书查询
func TestCertQueryWithCertId(t *testing.T) {
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)

	fmt.Println("============ get chain config by blockHeight in TestCertQueryWithCertId============")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "cert_hashes_2",
			Value: "b297d4e74ba0a88f9d154d63e53f2bf57116e09b6b8d2a718c24e59175f74cbe",
		},
	}

	sk, _ := native.GetUserSK(1)
	resp, err := native.QueryRequestWithCertID(sk, &client, &native.InvokeContractMsg{TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_QUERY.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书冻结
func TestCertFrozen(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "certs",
			Value: "-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDDFKZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTExNjA2NDYwNFoXDTI1\nMTExNTA2NDYwNFowgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKICGcmLV1GP\nkTOQQvTQtxFazqlL0lgue2g+2mFuvtysc8sBnzknLzfsNEsLiqzffTqa4yyRKtmE\nU8vos7QgSDSjezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCA1Bb6jyFaHmiWv3oSvZZ76Bmwuvg9fMDtiBzbTQNVDiTArBgNVHSME\nJDAigCAL80Y+n2PdoKCv3oohE7edPMsiNU/NDdkf/GTiENpiuTAKBggqhkjOPQQD\nAgNIADBFAiEA3MWFF2gSG40CZpD3Vukr7eZZjgzqlBXlPIApnpe15IkCIEKKBDBx\ndWY/PCzlgd8S+0PGBXMN1xwTUea3HA+4WQS3\n-----END CERTIFICATE-----\n,-----BEGIN CERTIFICATE-----\nMIIChzCCAiygAwIBAgIDAzprMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTExNjA2NDYwNFoXDTI1\nMTExNTA2NDYwNFowgY4xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjEqMCgGA1UEAxMhYWRtaW4xLnRscy53eC1vcmcxLmNo\nYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEOkgd8FMVWqSK\nH0ZCkGzC+Lmjk4Aagk5XfilUvom1vW5dfMOsmI8swUz+GkZ7eowTic/VYTIfCHQO\nD/0UFczvGaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAwKQYD\nVR0OBCIEIES8seDGskK6/9o9YwWG+lpywEZGzEgMp6F0q/hZ3t/fMCsGA1UdIwQk\nMCKAIAvzRj6fY92goK/eiiETt508yyI1T80N2R/8ZOIQ2mK5MAoGCCqGSM49BAMC\nA0kAMEYCIQDTjM9pE5giYNS2yA2HdnDa9a53sxzneRIZKGQk99412AIhAPErzu0h\n0O8jkwQR0Nwq0YAze0baxJfeArNLeKYXuRgd\n-----END CERTIFICATE-----\n",
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_FREEZE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书解冻
func TestCertUnfrozen(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "certs",
			Value: "-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDDFKZMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTExNjA2NDYwNFoXDTI1\nMTExNTA2NDYwNFowgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKICGcmLV1GP\nkTOQQvTQtxFazqlL0lgue2g+2mFuvtysc8sBnzknLzfsNEsLiqzffTqa4yyRKtmE\nU8vos7QgSDSjezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCA1Bb6jyFaHmiWv3oSvZZ76Bmwuvg9fMDtiBzbTQNVDiTArBgNVHSME\nJDAigCAL80Y+n2PdoKCv3oohE7edPMsiNU/NDdkf/GTiENpiuTAKBggqhkjOPQQD\nAgNIADBFAiEA3MWFF2gSG40CZpD3Vukr7eZZjgzqlBXlPIApnpe15IkCIEKKBDBx\ndWY/PCzlgd8S+0PGBXMN1xwTUea3HA+4WQS3\n-----END CERTIFICATE-----\n,-----BEGIN CERTIFICATE-----\nMIIChzCCAiygAwIBAgIDAzprMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTExNjA2NDYwNFoXDTI1\nMTExNTA2NDYwNFowgY4xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjEqMCgGA1UEAxMhYWRtaW4xLnRscy53eC1vcmcxLmNo\nYWlubWFrZXIub3JnMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEOkgd8FMVWqSK\nH0ZCkGzC+Lmjk4Aagk5XfilUvom1vW5dfMOsmI8swUz+GkZ7eowTic/VYTIfCHQO\nD/0UFczvGaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAwKQYD\nVR0OBCIEIES8seDGskK6/9o9YwWG+lpywEZGzEgMp6F0q/hZ3t/fMCsGA1UdIwQk\nMCKAIAvzRj6fY92goK/eiiETt508yyI1T80N2R/8ZOIQ2mK5MAoGCCqGSM49BAMC\nA0kAMEYCIQDTjM9pE5giYNS2yA2HdnDa9a53sxzneRIZKGQk99412AIhAPErzu0h\n0O8jkwQR0Nwq0YAze0baxJfeArNLeKYXuRgd\n-----END CERTIFICATE-----\n",
		},
	}

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_UNFREEZE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书解冻
func TestCertUnfrozenWithCertHash(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes_3",
		Value: "ae052a0deeffe50ba2b447ed43b77c505dd0cc8c8dc918ce3dbb51073d874729,09ff34fafd2b97c8e9c7e05704b075d90cb7fee93cd2e4234e71cee6df0a88e6",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_UNFREEZE.String(), Pairs: pairs})
	processResults(resp, err)
}

// 证书吊销
func TestCertRevocation(t *testing.T) {
	txId := utils.GetRandTxId()
	require.True(t, len(txId) > 0)
	fmt.Println("============ get chain config by blockHeight in TestCertRevocation============")
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key: "cert_crl",
		// 多个就换行就行
		Value: "-----BEGIN CRL-----\nMIIBVjCB/AIBATAKBggqgRzPVQGDdTCBgzELMAkGA1UEBhMCQ04xEDAOBgNVBAgT\nB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxHzAdBgNVBAoTFnd4LW9yZzEuY2hh\naW5tYWtlci5vcmcxCzAJBgNVBAsTAmNhMSIwIAYDVQQDExljYS53eC1vcmcxLmNo\nYWlubWFrZXIub3JnFw0yMTAxMTMwNjQ4MzhaFw0yMTAxMTMxMDQ4MzhaMBYwFAID\nDn50Fw0yMjAxMTIwMzM4MjJaoC8wLTArBgNVHSMEJDAigCAsQ4wyJIOuunNAHBqt\nESXwwBsY1fTkz7+vyHiD211y2zAKBggqgRzPVQGDdQNJADBGAiEA/ksRnjkjxpia\nfnOSCk557rPYWBFBxyYoyAbb22L39zwCIQCJsIiMNThs8VJN2MKaEeOSSSD1Z/0i\nrjsVWvt1I3nDpQ==\n-----END CRL-----\n",
	})

	sk, member := native.GetUserSK(1)
	resp, err := native.UpdateSysRequest(sk, member, &native.InvokeContractMsg{TxId: txId, TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ChainId: CHAIN1,
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERTS_REVOKE.String(), Pairs: pairs})
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
