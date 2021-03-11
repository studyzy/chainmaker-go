/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/common/ca"
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/protocol"
	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CHAIN1         = "chain1"
	certPathPrefix = "../config"
	WasmPath       = "wasm/fact-rust-0.7.2.wasm"
	userKeyPath    = certPathPrefix + "/wx-org1/certs/user/client1/client1.tls.key"
	userCrtPath    = certPathPrefix + "/wx-org1/certs/user/client1/client1.tls.crt"
	orgId          = "wx-org1.chainmaker.org"
	contractName   = "ex_fact"
	prePathFmt     = certPathPrefix + "/wx-org%s/certs/user/admin1/"
	OrgIdFormat    = "wx-org%d.chainmaker.org"
	tps            = 10000 //
)

var (
	caPaths = [][]string{
		{certPathPrefix + "/wx-org1/certs/ca/wx-org1.chainmaker.org/"},
		{certPathPrefix + "/wx-org1/certs/ca/wx-org2.chainmaker.org/"},
		{certPathPrefix + "/wx-org1/certs/ca/wx-org3.chainmaker.org/"},
		{certPathPrefix + "/wx-org1/certs/ca/wx-org4.chainmaker.org/"},
	}
	userKeyPaths = []string{
		certPathPrefix + "/wx-org1/certs/user/client1/client1.tls.key",
		certPathPrefix + "/wx-org2/certs/user/client1/client1.tls.key",
		certPathPrefix + "/wx-org3/certs/user/client1/client1.tls.key",
		certPathPrefix + "/wx-org4/certs/user/client1/client1.tls.key",
	}
	userCrtPaths = []string{
		certPathPrefix + "/wx-org1/certs/user/client1/client1.tls.crt",
		certPathPrefix + "/wx-org2/certs/user/client1/client1.tls.crt",
		certPathPrefix + "/wx-org3/certs/user/client1/client1.tls.crt",
		certPathPrefix + "/wx-org4/certs/user/client1/client1.tls.crt",
	}
	orgIds = []string{
		"wx-org1.chainmaker.org",
		"wx-org2.chainmaker.org",
		"wx-org3.chainmaker.org",
		"wx-org4.chainmaker.org",
	}
	IPs = []string{
		"49.233.1.182",
		"152.136.195.93",
		"152.136.186.128",
		"152.136.14.203",
	}
	Ports = []int{
		12304,
		22302,
		22303,
		22304,
	}
)

func main() {
	var step int
	flag.IntVar(&step, "step", 1, "STEP")
	conn, err := initGRPCConn(false, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	client := apiPb.NewRpcNodeClient(conn)
	file, err := ioutil.ReadFile(userKeyPath)
	if err != nil {
		panic(err)
	}
	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}
	switch step {
	case 0: // 0) 添加个人证书上链
		addCerts(1)
		time.Sleep(1 * time.Second)
		testCertQuery(sk3, client)
		return
	case 1: // 1) 合约创建
		testCreate(sk3, &client, CHAIN1)
		return
	case 2: // 3) 调用合约
		testMultiInvoke()
	default:
		panic("only three flag: upload cert(1), create contract(1), invoke contract(2)")
	}
}

var (
	signFailedStr    = "sign failed, %s"
	marshalFailedStr = "marshal payload failed, %s"
	deadLineErr      = "WARN: client.call err: deadline"
)

func testMultiInvoke() {
	var (
		interval    = 20 * 10000 / tps // 20 ms
		totalAmount int32
	)
	defer func() {
		fmt.Println("\n\n total send tx num : ", totalAmount)
	}()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		index := i % 4
		conn, err := initGRPCConn(false, index)
		if err != nil {
			fmt.Println(err)
			return
		}
		client := apiPb.NewRpcNodeClient(conn)
		keyFile, err := ioutil.ReadFile(userKeyPaths[index])
		if err != nil {
			panic(err)
		}
		sk, err := asym.PrivateKeyFromPEM(keyFile, nil)
		if err != nil {
			panic(err)
		}
		go func(k crypto.PrivateKey, c apiPb.RpcNodeClient, offset int) {
			for j := 0; j < 10000000; j++ {
				txId := testInvoke(k, &c, CHAIN1, offset)
				fmt.Printf("txId: %s\n", txId)
				atomic.AddInt32(&totalAmount, 1)
				time.Sleep(time.Duration(interval) * time.Millisecond) // 休息2 ms
			}
			wg.Done()
		}(sk, client, index)
	}
	wg.Wait()
}
func testCertQuery(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}
	certId, err := utils.GetCertificateIdHex(file, crypto.CRYPTO_ALGO_SHA256)
	if err != nil {
		panic(err)
	}
	fmt.Println(certId)
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes",
		Value: certId,
	})
	resp, err := QueryRequestWithCertID(sk3, &client, "", pairs)
	if err == nil {
		fmt.Printf("response: %v\n", resp)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
	}
	return
}

func QueryRequestWithCertID(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient,
	txId string, pairs []*commonPb.KeyValuePair) (*commonPb.TxResponse, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()
	if txId == "" {
		txId = utils.GetRandTxId()
	}
	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}
	certId, err := utils.GetCertificateId(file, crypto.CRYPTO_ALGO_SHA256)
	if err != nil {
		panic(err)
	}
	// 构造Sender
	sender := &acPb.SerializedMember{
		OrgId:      orgId,
		MemberInfo: certId,
		IsFullCert: false,
	}
	senderFull := &acPb.SerializedMember{
		OrgId:      orgId,
		MemberInfo: file,
		IsFullCert: true,
	}
	// 构造Header
	header := &commonPb.TxHeader{
		ChainId:        CHAIN1,
		Sender:         sender,
		TxType:         commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}
	payload := &commonPb.QueryPayload{
		ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(),
		Method:       commonPb.CertManageFunction_CERTS_QUERY.String(),
		Parameters:   pairs,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalFailedStr, err.Error())
	}
	req := &commonPb.TxRequest{
		Header:    header,
		Payload:   payloadBytes,
		Signature: nil,
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in QueryRequestWithCertID, %s", err.Error())
	}
	signer := getSigner(sk3, senderFull)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	req.Signature = signBytes
	return (*client).SendRequest(ctx, req)
}
func testCreate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ create contract [%s] ============\n", txId)
	wasmBin, _ := ioutil.ReadFile(WasmPath)
	var pairs []*commonPb.KeyValuePair
	method := commonPb.ManageUserContractFunction_INIT_CONTRACT.String()
	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName:    contractName,
			ContractVersion: "1.0.0",
			RuntimeType:     commonPb.RuntimeType_WASMER,
			//RuntimeType: commonPb.RuntimeType_GASM_CPP,
		},
		Method:     method,
		Parameters: pairs,
		ByteCode:   wasmBin,
	}
	if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("failed to sign endorsement, %s", err.Error())
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalFailedStr, err.Error())
	}
	resp := proposalRequestOld(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
		chainId, txId, payloadBytes, 0)
	fmt.Printf("testCreate send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
}
func testInvoke(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, index int) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)
	time := fmt.Sprintf("%d", utils.CurrentTimeMillisSeconds())
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "time",
			Value: time,
		},
		{
			Key:   "file_hash",
			Value: txId[len(txId)/2:],
		},
		{
			Key:   "file_name",
			Value: time,
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "save",
		//Method:     "query",
		Parameters: pairs,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalFailedStr, err.Error())
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes, index)
	fmt.Printf("testInvoke send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	return txId
}
func proposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte, index int) *commonPb.TxResponse {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(60*time.Second)))
	defer cancel()
	if txId == "" {
		txId = utils.GetRandTxId()
	}
	file, err := ioutil.ReadFile(userCrtPaths[index])
	if err != nil {
		panic(err)
	}
	certId, err := utils.GetCertificateId(file, crypto.CRYPTO_ALGO_SHA256)
	if err != nil {
		panic(err)
	}
	// 构造Sender
	//pubKeyString, _ := sk3.PublicKey().String()
	sender := &acPb.SerializedMember{
		OrgId:      orgIds[index],
		MemberInfo: certId,
		IsFullCert: false,
		//MemberInfo: []byte(pubKeyString),
	}
	senderFull := &acPb.SerializedMember{
		OrgId:      orgIds[index],
		MemberInfo: file,
		IsFullCert: true,
	}
	// 构造Header
	header := &commonPb.TxHeader{
		ChainId:        chainId,
		Sender:         sender,
		TxType:         txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}
	req := &commonPb.TxRequest{
		Header:    header,
		Payload:   payloadBytes,
		Signature: nil,
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in proposalRequest, %s", err.Error())
	}
	signer := getSigner(sk3, senderFull)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	//signBytes, err := signer.Sign("SM3", rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	req.Signature = signBytes
	reqBytes, _ := proto.Marshal(req)
	fmt.Println(fmt.Sprintf("req len = %d", len(reqBytes)))
	result, err := (*client).SendRequest(ctx, req)
	if err == nil {
		return result
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
	}
	fmt.Printf("ERROR: client.call err in proposalRequest: %v\n", err)
	os.Exit(0)
	return nil
}
func proposalRequestOld(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte, index int) *commonPb.TxResponse {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(60*time.Second)))
	defer cancel()
	if txId == "" {
		txId = utils.GetRandTxId()
	}
	file, err := ioutil.ReadFile(userCrtPaths[index])
	if err != nil {
		panic(err)
	}
	// 构造Sender
	//pubKeyString, _ := sk3.PublicKey().String()
	sender := &acPb.SerializedMember{
		OrgId:      orgIds[index],
		MemberInfo: file,
		IsFullCert: true,
		//MemberInfo: []byte(pubKeyString),
	}
	// 构造Header
	header := &commonPb.TxHeader{
		ChainId:        chainId,
		Sender:         sender,
		TxType:         txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}
	req := &commonPb.TxRequest{
		Header:    header,
		Payload:   payloadBytes,
		Signature: nil,
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in proposalRequestOld, %s", err.Error())
	}
	signer := getSigner(sk3, sender)
	//signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	req.Signature = signBytes
	result, err := (*client).SendRequest(ctx, req)
	if err == nil {
		return result
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
	}
	fmt.Printf("ERROR: client.call err in proposalRequestOld: %v\n", err)
	os.Exit(0)
	return nil
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.SerializedMember) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}
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
func initGRPCConn(useTLS bool, orgIdIndex int) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", IPs[orgIdIndex], Ports[orgIdIndex])
	fmt.Println(url)
	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths[orgIdIndex],
			CertFile:   userCrtPaths[orgIdIndex],
			KeyFile:    userKeyPaths[orgIdIndex],
		}
		c, err := tlsClient.GetCredentialsByCA()
		if err != nil {
			log.Fatalf("GetTLSCredentialsByCA err: %v", err)
			return nil, err
		}
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}
func acSign(msg *commonPb.ContractMgmtPayload, orgIdList []int) ([]*commonPb.EndorsementEntry, error) {
	msg.Endorsement = nil
	bytes, _ := proto.Marshal(msg)
	signers := make([]protocol.SigningMember, 0)
	for _, orgId := range orgIdList {
		numStr := strconv.Itoa(orgId)
		path := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.key"
		file, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		sk, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			panic(err)
		}
		userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
		file2, err := ioutil.ReadFile(userCrtPath)
		fmt.Println("node", orgId, "crt", string(file2))
		if err != nil {
			panic(err)
		}
		// 获取peerId
		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
		fmt.Println("node", orgId, "peerId", peerId)
		// 构造Sender
		sender1 := &acPb.SerializedMember{
			OrgId:      "wx-org" + numStr + ".chainmaker.org",
			MemberInfo: file2,
			IsFullCert: true,
		}
		signer := getSigner(sk, sender1)
		signers = append(signers, signer)
	}
	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, crypto.CRYPTO_ALGO_SHA256)
}
func addCerts(count int) {
	for i := 0; i < count; i++ {
		txId := utils.GetRandTxId()
		sk, member := getUserSK(i+1, userKeyPaths[i], userCrtPaths[i])
		resp, err := updateSysRequest(sk, member, false, &native.InvokeContractMsg{TxId: txId, ChainId: CHAIN1,
			TxType: commonPb.TxType_INVOKE_SYSTEM_CONTRACT, ContractName: commonPb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), MethodName: commonPb.CertManageFunction_CERT_ADD.String()})
		if err == nil {
			fmt.Printf("addCerts send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
			continue
		}
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			fmt.Println(deadLineErr)
			return
		}
		fmt.Printf("ERROR: client.call err in addCerts: %v\n", err)
		return
	}
}

// 获取用户私钥
func getUserSK(orgIDNum int, keyPath, certPath string) (crypto.PrivateKey, *acPb.SerializedMember) {
	file, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic(err)
	}
	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}
	file2, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(err)
	}
	sender := &acPb.SerializedMember{
		OrgId:      fmt.Sprintf(OrgIdFormat, orgIDNum),
		MemberInfo: file2,
		IsFullCert: true,
	}
	return sk3, sender
}
func updateSysRequest(sk3 crypto.PrivateKey, sender *acPb.SerializedMember, isTls bool, msg *native.InvokeContractMsg) (*commonPb.TxResponse, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalln(err)
		}
	}()

	conn, err := initGRPCConn(isTls, 0)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()
	if msg.TxId == "" {
		msg.TxId = utils.GetRandTxId()
	}
	// 构造Header
	header := &commonPb.TxHeader{
		ChainId:        msg.ChainId,
		Sender:         sender,
		TxType:         msg.TxType,
		TxId:           msg.TxId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}
	payload := &commonPb.SystemContractPayload{
		ChainId:      msg.ChainId,
		ContractName: msg.ContractName,
		Method:       msg.MethodName,
		Parameters:   msg.Pairs,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalFailedStr, err.Error())
	}
	req := &commonPb.TxRequest{
		Header:    header,
		Payload:   payloadBytes,
		Signature: nil,
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in updateSysRequest, %s", err.Error())
	}
	signer := getSigner(sk3, sender)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	fmt.Println(crypto.CRYPTO_ALGO_SHA256, "signBytes"+hex.EncodeToString(signBytes), "rawTxBytes="+hex.EncodeToString(rawTxBytes))
	err = signer.Verify(crypto.CRYPTO_ALGO_SHA256, rawTxBytes, signBytes)
	if err != nil {
		panic(err)
	}
	req.Signature = signBytes
	fmt.Println(req)
	return client.SendRequest(ctx, req)
}
