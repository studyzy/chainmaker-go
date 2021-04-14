/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/common/ca"
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"chainmaker.org/chainmaker-go/common/helper"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	logTempMarshalPayLoadFailed = "marshal payload failed, %s"
	logTempSendTx               = "send tx resp: code:%d, msg:%s, payload:%+v\n"
)

const (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "../../config-sql"
	userKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key"
	userCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	orgId          = "wx-org1.chainmaker.org"
	prePathFmt     = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
)

var (
	WasmPath        = ""
	WasmUpgradePath = ""
	contractName    = ""
	runtimeType     = commonPb.RuntimeType_WASMER
)

var caPaths = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}

// vm wasmer 整体功能测试，合约创建、升级、执行、查询、冻结、解冻、吊销、交易区块的查询、链配置信息的查询
func main() {
	initWasmerSqlTest()
	//initGasmTest()

	conn, err := initGRPCConnect(true)
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

	// test
	functionalTest(sk3, &client)
	//performanceTest(sk3, &client)
}

func performanceTest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient) {

	// 1) 合约创建
	testCreate(sk3, client, CHAIN1)
	time.Sleep(4 * time.Second)

	start := utils.CurrentTimeMillisSeconds()
	// 2) 执行合约-sql insert
	txId := ""
	count := 100000
	for i := 0; i < count; i++ {
		txId = testInvokeSqlInsert(sk3, client, CHAIN1, strconv.Itoa(i))
	}

	for {
		_, result := testQuerySqlById(sk3, client, CHAIN1, txId)
		if result != "{}" {
			fmt.Println(result)
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	end := utils.CurrentTimeMillisSeconds()
	fmt.Println("time cost", end-start)
	fmt.Println("tps", int64(count)/((end-start)/1000))
}
func functionalTest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient) {
	var (
		txId   string
		result string
		rs     = make(map[string]string, 0)
	)

	//1) 合约创建
	testCreate(sk3, client, CHAIN1)
	time.Sleep(4 * time.Second)

	// 2) 执行合约-sql insert
	for i := 0; i < 10; i++ {
		testInvokeSqlInsert(sk3, client, CHAIN1, strconv.Itoa(i))
	}
	//testInvokeSqlInsert(sk3, client, CHAIN1, "11")
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "11")
	time.Sleep(5 * time.Second)

	// 3) 查询 id
	testQuerySqlById(sk3, client, CHAIN1, txId)

	// 4) 执行合约-sql update name
	testInvokeSqlUpdate(sk3, client, CHAIN1, txId)
	time.Sleep(4 * time.Second)

	// 5) 查询 id
	_, result = testQuerySqlById(sk3, client, CHAIN1, txId)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlUpdate query", rs)
	if rs["id"] != txId {
		panic("query by id error, id err")
	} else {
		fmt.Println("↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓")
		fmt.Println("testInvokeSqlUpdate contract create invoke query test 【success】")
	}

	// 6) 范围查询 rang age
	testQuerySqlRangAge(sk3, client, CHAIN1)

	// 7) 执行合约-sql delete by id
	testInvokeSqlDelete(sk3, client, CHAIN1, txId)
	time.Sleep(4 * time.Second)

	// 8) 查询 id
	_, result = testQuerySqlById(sk3, client, CHAIN1, txId)
	if result != "{}" {
		panic("查询结果错误")
	}
	// 9) 跨合约调用
	testCrossCall(sk3, client, CHAIN1)
	time.Sleep(4 * time.Second)

	// 10) 交易回退
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "200000")
	time.Sleep(4 * time.Second)
	testInvokeSqlUpdateRollbackDbSavePoint(sk3, client, CHAIN1, txId)
	time.Sleep(4 * time.Second)
	_, result = testQuerySqlById(sk3, client, CHAIN1, txId)
	rs = make(map[string]string, 0)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlUpdateRollbackDbSavePoint query", rs)
	if rs["name"] == "长安链chainmaker333333333" {
		panic("testInvokeSqlUpdateRollbackDbSavePoint test 【fail】 query by id error, age err")
	} else if rs["name"] == "长安链chainmaker" {
		fmt.Println("↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓")
		fmt.Println("testInvokeSqlUpdateRollbackDbSavePoint test 【success】")
	} else {
		panic("error result")
	}

	// 9) 升级合约
	testUpgrade(sk3, client, CHAIN1)
	time.Sleep(3 * time.Second)

	// 10) 升级合约后执行插入
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "100000")
	time.Sleep(3 * time.Second)
	_, result = testQuerySqlById(sk3, client, CHAIN1, txId)
	rs = make(map[string]string, 0)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlInsert query", rs)
	if rs["age"] != "100000" {
		panic("query by id error, age err")
	} else {
		fmt.Println("↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓")
		fmt.Println("testInvokeSqlInsert test 【success】")
	}

	fmt.Println(txId, result, rs)
}
func initWasmerTest() {
	WasmPath = "../wasm/rust-fact-1.0.0.wasm"
	WasmUpgradePath = "../wasm/rust-functional-verify-1.0.0.wasm"
	contractName = "contract07"
	runtimeType = commonPb.RuntimeType_WASMER
}
func initWasmerSqlTest() {
	WasmPath = "../wasm/rust-sql-1.1.0.wasm"
	WasmUpgradePath = "../wasm/rust-sql-1.1.0.wasm"
	contractName = "contract112"
	runtimeType = commonPb.RuntimeType_WASMER
}
func initGasmTest() {
	WasmPath = "../wasm/go-sql-1.1.0.wasm"
	WasmUpgradePath = "../wasm/go-sql-1.1.0.wasm"
	contractName = "contract112"
	runtimeType = commonPb.RuntimeType_GASM
}
func initWxwmTest() {
	WasmPath = "../wasm/cpp-func-verify-1.0.0.wasm"
	WasmUpgradePath = "../wasm/cpp-func-verify-1.0.0.wasm"
	contractName = "contract01"
	runtimeType = commonPb.RuntimeType_WXVM
}
func testCreate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ create contract %s [%s] ============\n", contractName, txId)

	//wasmBin, _ := base64.StdEncoding.DecodeString(WasmPath)
	wasmBin, _ := ioutil.ReadFile(WasmPath)
	var pairs []*commonPb.KeyValuePair

	method := commonPb.ManageUserContractFunction_INIT_CONTRACT.String()

	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName:    contractName,
			ContractVersion: "1.0.0",
			//RuntimeType:     commonPb.RuntimeType_GASM,
			RuntimeType: runtimeType,
		},
		Method:     method,
		Parameters: pairs,
		ByteCode:   wasmBin,
	}

	if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("testCreate failed to sign endorsement, %s", err.Error())
		os.Exit(0)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	if resp.Code != 0 {
		panic(resp.Message)
	}
}

func testUpgrade(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========================test upgrade========================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")

	txId := utils.GetRandTxId()

	wasmBin, _ := ioutil.ReadFile(WasmUpgradePath)
	var pairs []*commonPb.KeyValuePair

	method := commonPb.ManageUserContractFunction_UPGRADE_CONTRACT.String()

	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName:    contractName,
			ContractVersion: "2.0.0",
			RuntimeType:     runtimeType,
		},
		Method:     method,
		Parameters: pairs,
		ByteCode:   wasmBin,
	}
	if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("testUpgrade failed to sign endorsement, %s", err.Error())
		os.Exit(0)
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
}

func testInvokeSqlInsert(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, age string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s,%s] ============\n", contractName, txId, age)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: txId,
		},
		{
			Key:   "age",
			Value: age,
		},
		{
			Key:   "name",
			Value: "长安链chainmaker",
		},
		{
			Key:   "id_card_no",
			Value: "510623199202023323",
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_insert",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func testInvokeSqlUpdate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, id string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_update] [%s] ============\n", contractName, id)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: id,
		},
		{
			Key:   "name",
			Value: "长安链chainmaker2222222",
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_update",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}
func testInvokeSqlUpdateRollbackDbSavePoint(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, id string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_update_rollback_save_point] [%s] ============\n", contractName, id)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: id,
		},
		{
			Key:   "name",
			Value: "长安链chainmaker333333333",
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_update_rollback_save_point",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}
func testCrossCall(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_cross_call] ============\n", contractName)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "contract_name",
			Value: contractName,
		},
		{
			Key:   "min_age",
			Value: "1",
		},
		{
			Key:   "max_age",
			Value: "15",
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_cross_call",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func testInvokeSqlDelete(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, id string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[save] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: id,
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_delete",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func testQuerySqlById(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, id string) (string, string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract %s[sql_query_by_id] id=%s ============\n", contractName, id)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: id,
		},
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_query_by_id",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	fmt.Println(string(resp.ContractResult.Result))
	//items := serialize.EasyUnmarshal(resp.ContractResult.Result)
	//for _, item := range items {
	//	fmt.Println(item.Key, item.Value)
	//}
	return txId, string(resp.ContractResult.Result)
}

func testQuerySqlRangAge(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract %s[sql_query_range_of_age] ============\n", contractName)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "max_age",
			Value: "1",
		},
		{
			Key:   "min_age",
			Value: "10",
		},
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "sql_query_range_of_age",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	fmt.Println(string(resp.ContractResult.Result))
	//items := serialize.EasyUnmarshal(resp.ContractResult.Result)
	//for _, item := range items {
	//	fmt.Println(item.Key, item.Value)
	//}
	return txId
}

func proposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte) *commonPb.TxResponse {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
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
	sender := &acPb.SerializedMember{
		OrgId:      orgId,
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

	req.Signature = signBytes

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

func getSigner(sk3 crypto.PrivateKey, sender *acPb.SerializedMember) protocol.SigningMember {
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

func initGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", IP, Port)

	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths,
			CertFile:   userCrtPath,
			KeyFile:    userKeyPath,
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

func constructPayload(contractName, method string, pairs []*commonPb.KeyValuePair) []byte {
	payload := &commonPb.QueryPayload{
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
		//fmt.Println("node", orgId, "crt", string(file2))
		if err != nil {
			panic(err)
		}

		// 获取peerId
		_, err = helper.GetLibp2pPeerIdFromCert(file2)
		//fmt.Println("node", orgId, "peerId", peerId)

		// 构造Sender
		sender1 := &acPb.SerializedMember{
			OrgId:      "wx-org" + numStr + ".chainmaker.org",
			MemberInfo: file2,
			IsFullCert: true,
		}

		signer := getSigner(sk, sender1)
		signers = append(signers, signer)
	}

	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, "SHA256")
}
