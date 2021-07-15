/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/common/crypto"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	logTempMarshalPayLoadFailed     = "marshal payload failed, %s"
	logTempUnmarshalBlockInfoFailed = "blockInfo unmarshal error %s\n"
	logTempSendTx                   = "send tx resp: code:%d, msg:%s, payload:%+v\n"
	logTempSendBlock                = "send tx resp: code:%d, msg:%s, blockInfo:%+v\n"
	fieldWithRWSet                  = "withRWSet"
)

const (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "../../config"
	userKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key"
	userCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	orgId          = "wx-org1.chainmaker.org"
	prePathFmt     = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
)

func CreateContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string, wasmPath string,
	runtimeType commonPb.RuntimeType) string {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ create contract %s [%s] ============\n", contractName, txId)

	//wasmBin, _ := base64.StdEncoding.DecodeString(WasmPath)
	wasmBin, _ := ioutil.ReadFile(wasmPath)
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_NAME.String(),
		Value: []byte(contractName),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_VERSION.String(),
		Value: []byte("1.2.1"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
		Value: []byte(runtimeType.String()),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.InitContract_CONTRACT_BYTECODE.String(),
		Value: wasmBin,
	})
	payload := &commonPb.Payload{
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
		Method:       syscontract.ContractManageFunction_INIT_CONTRACT.String(),
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	if resp.Code != 0 {
		panic(resp.Message)
	}
	return txId
}

func proposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte) *commonPb.TxResponse {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	if txId == "" {
		txId = utils.GetRandTxId()
	}

	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}

	// 构造Sender
	//pubKeyString, _ := sk3.PublicKey().String()
	sender := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		////IsFullCert: true,
		//MemberInfo: []byte(pubKeyString),
	}

	// 构造Header
	header := &commonPb.Payload{
		ChainId: chainId,
		//Sender:         sender,
		TxType:         txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}

	req := &commonPb.TxRequest{
		Payload: header,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed, %s", err.Error())
		os.Exit(0)
	}

	fmt.Errorf("################ %s", string(sender.MemberInfo))

	signer := getSigner(sk3, sender)
	//signBytes, err := signer.Sign("SHA256", rawTxBytes)
	signBytes, err := signer.Sign("SM3", rawTxBytes)
	if err != nil {
		log.Fatalf("sign failed, %s", err.Error())
		os.Exit(0)
	}

	req.Sender.Signature = signBytes

	result, err := (*client).SendRequest(ctx, req)

	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok && statusErr.Code() == codes.DeadlineExceeded {
			fmt.Println("WARN: client.call err: deadline")
			os.Exit(0)
		}
		fmt.Printf("ERROR: client.call err: %v\n", err)
		os.Exit(0)
	}
	return result
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.Member) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}
	//fmt.Printf("skPEM: %s\n", skPEM)

	m, err := accesscontrol.MockAccessControl().NewMemberFromCertPem(sender.OrgId, string(sender.MemberInfo))
	if err != nil {
		panic(err)
	}

	signer, err := accesscontrol.MockAccessControl().NewSigningMember(m, skPEM, "")
	if err != nil {
		panic(err)
	}
	return signer
}
func QueryUserContractInfo(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId, contractName string) *commonPb.TxResponse {

	txId := utils.GetRandTxId()
	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: syscontract.GetContractInfo_CONTRACT_NAME.String(), Value: []byte(contractName)}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := constructPayload(syscontract.SystemContract_CONTRACT_MANAGE.String(), syscontract.ContractQueryFunction_GET_CONTRACT_INFO.String(), pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes)
	return resp

}

func constructPayload(contractName, method string, pairs []*commonPb.KeyValuePair) []byte {
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
		os.Exit(0)
	}

	return payloadBytes
}

func UpgradeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId, contractName, wasmUpgradePath string,
	runtimeType commonPb.RuntimeType) *commonPb.TxResponse {

	txId := utils.GetRandTxId()

	wasmBin, _ := ioutil.ReadFile(wasmUpgradePath)
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_NAME.String(),
		Value: []byte(contractName),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_VERSION.String(),
		Value: []byte("2.0.0"),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_RUNTIME_TYPE.String(),
		Value: []byte(runtimeType.String()),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.UpgradeContract_CONTRACT_BYTECODE.String(),
		Value: wasmBin,
	})
	payload := &commonPb.Payload{
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
		Method:       syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(),
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	return resp
	//	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
}

func FreezeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType) {
	freezeOrUnfreezeOrRevoke(sk3, client, chainId, contractName, runtimeType, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
}
func UnfreezeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType) {
	freezeOrUnfreezeOrRevoke(sk3, client, chainId, contractName, runtimeType, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
}
func RevokeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType) {
	freezeOrUnfreezeOrRevoke(sk3, client, chainId, contractName, runtimeType, syscontract.ContractManageFunction_REVOKE_CONTRACT.String())
}
func freezeOrUnfreezeOrRevoke(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType, method string) {
	txId := utils.GetRandTxId()

	fmt.Printf("\n============ [%s] contract [%s] ============\n", method, txId)
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   syscontract.FreezeContract_CONTRACT_NAME.String(),
		Value: []byte(contractName),
	})
	payload := &commonPb.Payload{
		ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
}
