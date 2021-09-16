/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
)

func genCreateContractTxRequest(orgid string, sk3 crypto.PrivateKey, userCrtPath string,
	chainId string) (*commonPb.TxRequest, error) {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ create contract [%s] ============\n", txId)

	wasmBin, _ := ioutil.ReadFile(wasmPath)
	payload, _ := utils.GenerateInstallContractPayload(contractName, "1.0.0", contractType, wasmBin, nil)

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed in genCreateContractTxRequest, %s", err.Error())
		os.Exit(0)
	}

	return contructTxRequest(orgid, sk3, userCrtPath, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
}

func genInvokeContractTxRequest(orgid string, sk3 crypto.PrivateKey, userCrtPath string,
	chainId string) (*commonPb.TxRequest, error) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "time",
			Value: []byte(fmt.Sprintf("%d", utils.CurrentTimeMillisSeconds())),
		},
		{
			Key:   "id",
			Value: []byte(txId),
		},
		{
			Key:   "hash",
			Value: []byte(txId),
		},
	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "save",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed in genInvokeContractTxRequest, %s", err.Error())
	}

	return contructTxRequest(orgid, sk3, userCrtPath, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
}

func genGetBlockByTxIDTxRequest(orgid string, sk3 crypto.PrivateKey, txid string,
	chainId string) (*commonPb.TxRequest, error) {
	fmt.Printf("\n============ get block by txId [%s] ============\n", txid)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "txId",
			Value: []byte(txid),
		},
		{
			Key:   "withRWSet",
			Value: []byte("false"),
		},
	}

	payload := &commonPb.Payload{
		ContractName: "query_system_contract",
		Method:       "GET_BLOCK_BY_TX_ID",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed in genGetBlockByTxIDTxRequest, %s", err.Error())
	}

	return contructTxRequest(orgid, sk3, userCrtPath, commonPb.TxType_QUERY_CONTRACT,
		chainId, txid, payloadBytes)
}

func contructTxRequest(orgid string, sk3 crypto.PrivateKey, userCrtPath string, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte) (*commonPb.TxRequest, error) {

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
		OrgId:      orgid,
		MemberInfo: file,
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
		return nil, err
	}

	signer := getSigner(sk3, sender)
	//signBytes, err := signer.Sign("SHA256", rawTxBytes)
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	if err != nil {
		log.Fatalf("sign failed, %s", err.Error())
		return nil, err
	}

	req.Sender.Signature = signBytes

	fmt.Printf("gen tx success. id %v", req.Payload.TxId)

	return req, nil
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.Member) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}

	signer, err := accesscontrol.NewCertSigningMember("", sender, skPEM, "")
	if err != nil {
		panic(err)
	}
	return signer
}

//func acSign(msg *commonPb.Payload, orglist []string) ([]*commonPb.EndorsementEntry, error) {
//	msg.Endorsement = nil
//	bytes, _ := proto.Marshal(msg)
//
//	signers := make([]protocol.SigningMember, 0)
//
//	for _, orgid := range orglist {
//
//		path := fmt.Sprintf(prePathFmt, orgid) + "admin1.sign.key"
//		file, err := ioutil.ReadFile(path)
//		if err != nil {
//			panic(err)
//		}
//		sk, err := asym.PrivateKeyFromPEM(file, nil)
//		if err != nil {
//			panic(err)
//		}
//
//		userCrtPath := fmt.Sprintf(prePathFmt, orgid) + "admin1.sign.crt"
//		file2, err := ioutil.ReadFile(userCrtPath)
//		fmt.Println("node", orgid, "crt", string(file2))
//		if err != nil {
//			panic(err)
//		}
//
//		// 获取peerId
//		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
//		fmt.Println("node", orgid, "peerId", peerId)
//
//		// 构造Sender
//		sender1 := &acPb.Member{
//			OrgId:      orgid,
//			MemberInfo: file2,
//		}
//
//		signer := getSigner(sk, sender1)
//		signers = append(signers, signer)
//	}
//	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, "SHA256")
//}
