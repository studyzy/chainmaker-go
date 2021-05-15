/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/common/ca"
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"chainmaker.org/chainmaker-go/common/helper"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CHAIN1           = "chain1"
	certPathPrefix   = "/big_space/chainmaker/chainmaker-go/build/crypto-config"
	certWasmPath     = "/big_space/chainmaker/chainmaker-go/test/wasm/rust-fact-1.0.0.wasm"
	addWasmPath      = "/big_space/chainmaker/chainmaker-go/test/wasm/rust-counter-1.0.0.wasm"
	userKeyPath      = certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.tls.key"
	userCrtPath      = certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	orgId            = "wx-org1.chainmaker.org"
	certContractName = "ex_fact"
	addContractName  = "add"

	prePathFmt  = certPathPrefix + "/wx-org%s.chainmaker.org/user/admin1/"
	OrgIdFormat = "wx-org%d.chainmaker.org"
	tps         = 10000 //
)

var (
	caPaths = [][]string{
		{certPathPrefix + "/wx-org1.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org2.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org3.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org4.chainmaker.org/ca"},
	}
	userKeyPaths = []string{
		certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.tls.key",
		certPathPrefix + "/wx-org2.chainmaker.org/user/client1/client1.tls.key",
		certPathPrefix + "/wx-org3.chainmaker.org/user/client1/client1.tls.key",
		certPathPrefix + "/wx-org4.chainmaker.org/user/client1/client1.tls.key",
	}
	userCrtPaths = []string{
		certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.tls.crt",
		certPathPrefix + "/wx-org2.chainmaker.org/user/client1/client1.tls.crt",
		certPathPrefix + "/wx-org3.chainmaker.org/user/client1/client1.tls.crt",
		certPathPrefix + "/wx-org4.chainmaker.org/user/client1/client1.tls.crt",
	}
	orgIds = []string{
		"wx-org1.chainmaker.org",
		"wx-org2.chainmaker.org",
		"wx-org3.chainmaker.org",
		"wx-org4.chainmaker.org",
	}
	IPs = []string{
		"127.0.0.1",
		"127.0.0.1",
		"127.0.0.1",
		"127.0.0.1",
	}
	Ports = []int{
		12301,
		12302,
		12303,
		12304,
	}
)

var (
	trustRootOrgId   = ""
	trustRootCrtPath = ""
	nodeOrgOrgId     = ""
	nodeOrgAddresses = ""
)

func main() {
	var (
		step     int
		wasmType int
	)
	flag.IntVar(&step, "step", 1, "0: add certs, 1: creat contract, 2: add trustRoot, 3: add validator, 4: get chainConfig")
	flag.IntVar(&wasmType, "wasm", 0, "0: cert, 1: counter")
	flag.StringVar(&trustRootCrtPath, "trust_root_crt", "", "node crt that will be added to the trust root")
	flag.StringVar(&trustRootOrgId, "trust_root_org_id", "", "node orgID that will be added to the trust root")
	flag.StringVar(&nodeOrgOrgId, "nodeOrg_org_id", "", "node orgID that will be added")
	flag.StringVar(&nodeOrgAddresses, "nodeOrg_addresses", "", "node address that will be added")
	flag.Parse()

	conn, err := initGRPCConn(true, 0)
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
		addCerts(4)
		time.Sleep(1 * time.Second)
		testCertQuery(sk3, client)
		return
	case 1: // 1) 合约创建
		testCreate(sk3, client, CHAIN1, wasmType)
		return
	case 2: // 2) 添加trustRoot
		trustRootAdd(sk3, client, CHAIN1)
	case 3:
		nodeOrgAdd(sk3, client, CHAIN1)
	case 4:
		config := getChainConfig(sk3, client, CHAIN1)
		fmt.Println(config)
	default:
		panic("only three flag: upload cert(1), create contract(1), invoke contract(2)")
	}
}

var (
	signFailedStr    = "sign failed, %s"
	marshalFailedStr = "marshal payload failed, %s"
	deadLineErr      = "WARN: client.call err: deadline"
)

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
func testCreate(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string, wasmType int) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ create contract [%s] ============\n", txId)
	wasmPath := certWasmPath
	wasmName := certContractName
	if wasmType == 1 {
		wasmPath = addWasmPath
		wasmName = addContractName
	}
	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var (
		method = commonPb.ManageUserContractFunction_INIT_CONTRACT.String()
		pairs  []*commonPb.KeyValuePair
	)

	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName:    wasmName,
			ContractVersion: "1.0.0",
			RuntimeType:     commonPb.RuntimeType_WASMER,
		},
		Method:      method,
		Parameters:  pairs,
		ByteCode:    wasmBin,
		Endorsement: nil,
	}
	if endorsement, err := acSignWithManager(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("failed to sign endorsement, %s", err.Error())
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalFailedStr, err.Error())
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
		chainId, txId, payloadBytes, 0)
	fmt.Printf("testCreate send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
}
func proposalRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, txType commonPb.TxType,
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
	result, err := client.SendRequest(ctx, req)
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
func acSignWithManager(msg *commonPb.ContractMgmtPayload, orgIdList []int) ([]*commonPb.EndorsementEntry, error) {
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

func getKeysAndCertsPath(orgIdList []int) (keysFile, certsFile []string) {
	keysFile = make([]string, 0, len(orgIdList))
	certsFile = make([]string, 0, len(orgIdList))
	for _, orgId := range orgIdList {
		numStr := strconv.Itoa(orgId)
		keyPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.key"
		userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
		keysFile = append(keysFile, keyPath)
		certsFile = append(certsFile, userCrtPath)
	}
	return keysFile, certsFile
}

func addCerts(count int) {
	for i := 0; i < count; i++ {
		txId := utils.GetRandTxId()
		sk, member := getUserSK(i+1, userKeyPaths[i], userCrtPaths[i])
		resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{TxId: txId, ChainId: CHAIN1,
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

func getChainConfig(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) *configPb.ChainConfig {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), commonPb.ConfigFunction_GET_CHAIN_CONFIG.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes, 0)
	chainConfig := &configPb.ChainConfig{}
	if err = proto.Unmarshal(resp.ContractResult.Result, chainConfig); err != nil {
		log.Fatalf("unmarshal bytes failed, err: %s", err)
	}
	return chainConfig
}

func constructPayload(contractName, method string, pairs []*commonPb.KeyValuePair) ([]byte, error) {
	payload := &commonPb.QueryPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return payloadBytes, nil
}

type InvokerMsg struct {
	txType       commonPb.TxType
	chainId      string
	txId         string
	method       string
	contractName string
	oldSeq       uint64
	pairs        []*commonPb.KeyValuePair
}

func trustRootAdd(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) error {
	// 构造Payload
	if trustRootOrgId == "" || trustRootCrtPath == "" {
		log.Fatalf("the trustRoot orgId or crt is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: trustRootOrgId,
	})
	file, err := os.Open(trustRootCrtPath)
	if err != nil {
		log.Fatalf("open file failed: %s, reason: %s", trustRootCrtPath, err)
	}
	defer file.Close()
	bz, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("read content from file failed: %s", err)
	}
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "root",
		Value: string(bz),
	})

	config := getChainConfig(sk3, client, chainId)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_UPDATE_CHAIN_CONFIG, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), method: commonPb.ConfigFunction_TRUST_ROOT_ADD.String(), pairs: pairs, oldSeq: config.Sequence})
	if err != nil {
		log.Fatalf("create update request failed, err: %s", err)
	}
	fmt.Println("txId: ", txId, "; result: ", resp)
	return nil
}

func configUpdateRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, msg *InvokerMsg) (*commonPb.TxResponse, string, error) {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	txId := utils.GetRandTxId()
	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		return nil, "", err
	}

	// 构造Sender
	senderFull := &acPb.SerializedMember{
		OrgId:      orgId,
		MemberInfo: file,
		IsFullCert: true,
	}

	// 构造Header
	header := &commonPb.TxHeader{
		ChainId:        msg.chainId,
		Sender:         senderFull,
		TxType:         msg.txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}

	payload := &commonPb.SystemContractPayload{
		ChainId:      msg.chainId,
		ContractName: msg.contractName,
		Method:       msg.method,
		Parameters:   msg.pairs,
		Sequence:     msg.oldSeq + 1,
	}
	adminSignKeys, adminSignCrts := getKeysAndCertsPath([]int{1, 2, 3, 4})
	entries, err := aclSignSystemContract(*payload, orgIds, adminSignKeys, adminSignCrts)
	if err != nil {
		panic(err)
	}
	payload.Endorsement = entries

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	req := &commonPb.TxRequest{
		Header:    header,
		Payload:   payloadBytes,
		Signature: nil,
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, "", err
	}
	signer := getSigner(sk3, senderFull)
	signBytes, err := signer.Sign("SM3", rawTxBytes)
	if err != nil {
		log.Fatalf("sign msg failed")
	}
	req.Signature = signBytes

	result, err := client.SendRequest(ctx, req)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return nil, "", fmt.Errorf("client.call err: deadline\n")
		}
		return nil, "", fmt.Errorf("client.call err: %v\n", err)
	}
	return result, txId, nil
}

func aclSignSystemContract(msg commonPb.SystemContractPayload, orgIds, adminSignKeys, adminSignCrts []string) ([]*commonPb.EndorsementEntry, error) {
	msg.Endorsement = nil
	bytes, _ := proto.Marshal(&msg)

	signers := make([]protocol.SigningMember, 0)
	orgIdArray := orgIds
	adminSignKeyArray := adminSignKeys
	adminSignCrtArray := adminSignCrts

	if len(adminSignKeyArray) != len(adminSignCrtArray) {
		return nil, errors.New(fmt.Sprintf("admin key len is not equal to crt len: %d, %d", len(adminSignKeyArray), len(adminSignCrtArray)))
	}
	if len(adminSignKeyArray) != len(orgIdArray) {
		return nil, errors.New("admin key len is not equal to orgId len")
	}

	for i, key := range adminSignKeyArray {
		file, err := ioutil.ReadFile(key)
		if err != nil {
			panic(err)
		}
		sk3, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			panic(err)
		}

		file2, err := ioutil.ReadFile(adminSignCrtArray[i])
		fmt.Println("node", i, "crt", string(file2))
		if err != nil {
			panic(err)
		}

		// 获取peerId
		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
		fmt.Println("node", i, "peerId", peerId)

		// 构造Sender
		sender1 := &acPb.SerializedMember{
			OrgId:      orgIdArray[i],
			MemberInfo: file2,
			IsFullCert: true,
		}
		signer := getSigner(sk3, sender1)
		signers = append(signers, signer)
	}

	endorsements, err := accesscontrol.MockSignWithMultipleNodes(bytes, signers, crypto.CRYPTO_ALGO_SHA256)
	if err != nil {
		return nil, err
	}
	fmt.Printf("endorsements:\n%v\n", endorsements)
	return endorsements, nil
}

func nodeOrgAdd(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) error {
	// 构造Payload
	if nodeOrgOrgId == "" || nodeOrgAddresses == "" {
		return errors.New("the nodeOrg orgId or addresses is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: nodeOrgOrgId,
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: nodeOrgAddresses,
	})

	config := getChainConfig(sk3, client, chainId)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_UPDATE_CHAIN_CONFIG, chainId: chainId,
		contractName: commonPb.ContractName_SYSTEM_CONTRACT_CHAIN_CONFIG.String(), method: commonPb.ConfigFunction_NODE_ORG_ADD.String(), pairs: pairs, oldSeq: config.Sequence})
	if err != nil {
		log.Fatalf("create configUpdateRequest error")
	}
	fmt.Println("txId: ", txId, ", resp: ", resp)
	return nil
}
