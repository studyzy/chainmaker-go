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
	"strconv"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/crypto"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	logTempMarshalPayLoadFailed     = "marshal payload failed, %s"
	logTempUnmarshalBlockInfoFailed = "blockInfo unmarshal error %s\n"
	logTempSendTx                   = "send tx resp: code:%d, msg:%s txid:%s, result:%+v\n"
	logTempSendBlock                = "send tx resp: code:%d, msg:%s, blockInfo:%+v\n"
	fieldWithRWSet                  = "withRWSet"
)

var (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "./config"

	orgId = "wx-org1.chainmaker.org"
)

func SetCertPathPrefix(path string) {
	certPathPrefix = path
}
func prePathFmt() string {
	return certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
}
func userKeyPath() string {
	return certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key"
}
func userCrtPath() string {
	return certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
}

func adminCrtPath() string {
	return certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"
}

func CreateContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string, wasmPath string,
	runtimeType commonPb.RuntimeType) string {

	txId := utils.GetRandTxId()
	fmt.Printf("\n============ create contract %s [%s] ============\n", contractName, txId)

	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		panic(err)
	}
	payload, _ := utils.GenerateInstallContractPayload(contractName, "1.2.1", runtimeType, wasmBin, nil)

	resp := ProposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload, []int{1, 2, 3, 4})
	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	if resp.Code != 0 {
		panic(resp.Message)
	}
	return txId
}

func UpgradeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId, contractName, wasmUpgradePath string,
	runtimeType commonPb.RuntimeType) *commonPb.TxResponse {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ upgrade contract %s [%s] ============\n", contractName, txId)

	wasmBin, _ := ioutil.ReadFile(wasmUpgradePath)
	payload, _ := utils.GenerateInstallContractPayload(contractName, "2.0.1", runtimeType, wasmBin, nil)
	payload.Method = syscontract.ContractManageFunction_UPGRADE_CONTRACT.String()

	resp := ProposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload, []int{1, 2, 3, 4})
	return resp
}

func acSign(msg *commonPb.Payload, orgIdList []int) ([]*commonPb.EndorsementEntry, error) {
	bytes, _ := msg.Marshal()

	signers := make([]protocol.SigningMember, 0)
	for _, orgId := range orgIdList {

		numStr := strconv.Itoa(orgId)
		path := fmt.Sprintf(prePathFmt(), numStr) + "admin1.sign.key"
		file, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		sk, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			panic(err)
		}

		userCrtPath := fmt.Sprintf(prePathFmt(), numStr) + "admin1.sign.crt"
		file2, err := ioutil.ReadFile(userCrtPath)
		//fmt.Println("node", orgId, "crt", string(file2))
		if err != nil {
			panic(err)
		}

		// 获取peerId
		_, err = helper.GetLibp2pPeerIdFromCert(file2)
		//fmt.Println("node", orgId, "peerId", peerId)

		// 构造Sender
		sender1 := &acPb.Member{
			OrgId:      "wx-org" + numStr + ".chainmaker.org",
			MemberInfo: file2,
		}

		signer := GetSigner(sk, sender1)
		signers = append(signers, signer)
	}

	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, "SHA256")
}

func ProposalMultiRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payload *commonPb.Payload, orgIdList []int, timestamp int64) *commonPb.TxResponse {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	if txId == "" {
		txId = utils.GetRandTxId()
	}

	file, err := ioutil.ReadFile(userCrtPath())
	if err != nil {
		panic(err)
	}

	// 构造Sender
	sender := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
	}

	// 构造Header
	payload.ChainId = chainId
	payload.TxType = txType
	payload.TxId = txId
	payload.Timestamp = timestamp
	req := &commonPb.TxRequest{
		Payload: payload,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}
	if len(orgIdList) > 0 {
		if endorsement, err := acSign(payload, orgIdList); err == nil {
			req.Endorsers = endorsement
		} else {
			log.Fatalf("testCreate failed to sign endorsement, %s", err.Error())
			os.Exit(0)
		}
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed, %s", err.Error())
		os.Exit(0)
	}

	fmt.Errorf("################ %s", string(sender.MemberInfo))

	signer := GetSigner(sk3, sender)
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	//signBytes, err := signer.Sign("SM3", rawTxBytes)
	if err != nil {
		log.Fatalf("sign failed, %s", err.Error())
		os.Exit(0)
	}

	req.Sender.Signature = signBytes
	fmt.Printf("client signed tx request sender:%+v,\nendorsers:%+v\n", req.Sender, req.Endorsers)
	result, err := (*client).SendRequest(ctx, req)
	//result, err := client.SendRequest(ctx, req)

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

func ProposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payload *commonPb.Payload, orgIdList []int) *commonPb.TxResponse {
	return ProposalMultiRequest(sk3, client, txType, chainId, txId, payload, orgIdList, time.Now().Unix())
}

func GetSigner(sk3 crypto.PrivateKey, sender *acPb.Member) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}
	//fmt.Printf("skPEM: %s\n", skPEM)

	signer, err := accesscontrol.NewCertSigningMember("", sender, skPEM, "")
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

	payload := ConstructQueryPayload(syscontract.SystemContract_CONTRACT_MANAGE.String(), syscontract.ContractQueryFunction_GET_CONTRACT_INFO.String(), pairs)

	resp := ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload, nil)
	return resp

}

func ConstructQueryPayload(contractName, method string, pairs []*commonPb.KeyValuePair) *commonPb.Payload {
	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_QUERY_CONTRACT,
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	return payload
}

func FreezeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType) string {
	return freezeOrUnfreezeOrRevoke(sk3, client, chainId, contractName, runtimeType, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
}
func UnfreezeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType) string {
	return freezeOrUnfreezeOrRevoke(sk3, client, chainId, contractName, runtimeType, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
}
func RevokeContract(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType) string {
	return freezeOrUnfreezeOrRevoke(sk3, client, chainId, contractName, runtimeType, syscontract.ContractManageFunction_REVOKE_CONTRACT.String())
}
func freezeOrUnfreezeOrRevoke(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, contractName string,
	runtimeType commonPb.RuntimeType, method string) string {
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
	resp := ProposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload, []int{1, 2, 3, 4})
	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return txId
}
