/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/ca"
	"chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/asym"
	"chainmaker.org/chainmaker/common/helper"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
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
	"strings"
	"time"
)

const (
	logTempMarshalPayLoadFailed = "marshal payload failed, %s"
	logTempSendTx               = "send tx resp: code:%d, msg:%s, payload:%+v\n"
)

const (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12351
	certPathPrefix = "../../config"
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
	fmt.Println("\n\n\n\n======wasmer test=====\n\n\n\n")
	initWasmerSqlTest()
	functionalTest(sk3, &client)

	fmt.Println("\n\n\n\n======gasm test=====\n\n\n\n")
	time.Sleep(time.Second * 4)
	initGasmTest()
	functionalTest(sk3, &client)

	//performanceTest(sk3, &client)
	//testWaitTx(sk3, &client, CHAIN1, "20fa21fcff774cef96bcf6294306caa8d30fb9d27dac4484b5ffceaaf018ef79")

}

func functionalTest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient) {
	var (
		txId   string
		id     string
		result string
		rs     = make(map[string]string, 0)
	)

	fmt.Println("//1) 合约创建")
	txId = testCreate(sk3, client, CHAIN1)
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("// 2) 执行合约-sql insert")
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "11")
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "11")

	for i := 0; i < 10; i++ {
		txId = testInvokeSqlInsert(sk3, client, CHAIN1, strconv.Itoa(i))
	}
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)
	id = txId

	fmt.Println("// 3) 查询 age11的 id:" + id)
	_, result = testQuerySqlById(sk3, client, CHAIN1, id)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlUpdate query", rs)
	if rs["id"] != id {
		fmt.Println("result", rs)
		panic("query by id error, id err")
	}

	fmt.Println("// 4) 执行合约-sql update name=长安链chainmaker_update where id=" + id)
	txId = testInvokeSqlUpdate(sk3, client, CHAIN1, id)
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("// 5) 查询 id=" + id + " 看name是不是更新成了长安链chainmaker_update：")
	_, result = testQuerySqlById(sk3, client, CHAIN1, id)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlUpdate query", rs)
	if rs["name"] != "长安链chainmaker_update" {
		fmt.Println("result", rs)
		panic("query update result error")
	} else {
		fmt.Println("↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓")
		fmt.Println("testInvokeSqlUpdate contract create invoke query test 【success】")
	}

	fmt.Println("// 6) 范围查询 rang age 1~10")
	testQuerySqlRangAge(sk3, client, CHAIN1)

	fmt.Println("// 7) 执行合约-sql delete by id age=11")
	txId = testInvokeSqlDelete(sk3, client, CHAIN1, id)
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("// 8) 再次查询 id age=11，应该查不到")
	_, result = testQuerySqlById(sk3, client, CHAIN1, id)
	if result != "{}" {
		fmt.Println("result", result)
		panic("查询结果错误")
	}
	//// 9) 跨合约调用
	testCrossCall(sk3, client, CHAIN1)
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 10) 交易回退
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "2000")
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)
	id = txId
	for i := 0; i < 3; i++ {
		fmt.Println("试图将id=" + id + " 的name改为长安链chainmaker_save_point，但是发生了错误，所以修改不会成功")
		txId = testInvokeSqlUpdateRollbackDbSavePoint(sk3, client, CHAIN1, id)
		time.Sleep(2 * time.Second)
		testWaitTx(sk3, client, CHAIN1, txId)

		fmt.Println("// 11 再次查询age=2000的这条数据，如果name被更新了，那么说明savepoint Rollback失败了")
		_, result = testQuerySqlById(sk3, client, CHAIN1, id)
		rs = make(map[string]string, 0)
		json.Unmarshal([]byte(result), &rs)
		fmt.Println("testInvokeSqlUpdateRollbackDbSavePoint query", rs)
		if rs["name"] == "chainmaker_save_point" {
			panic("testInvokeSqlUpdateRollbackDbSavePoint test 【fail】 query by id error, age err")
		} else if rs["name"] == "长安链chainmaker" {
			fmt.Println("↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓↓")
			fmt.Println("testInvokeSqlUpdateRollbackDbSavePoint test 【success】")
		} else {
			panic("error result")
		}
	}

	// 9) 升级合约
	testUpgrade(sk3, client, CHAIN1)
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 10) 升级合约后执行插入
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "100000")
	time.Sleep(2 * time.Second)
	testWaitTx(sk3, client, CHAIN1, txId)
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

	// 异常功能测试
	if runtimeType == commonPb.RuntimeType_WASMER {
		fmt.Println("\n// 1、建表、索引、视图等DDL语句只能在合约安装init_contract 和合约升级upgrade中使用。")
		_, result = testInvokeSqlCommon(sk3, client, "sql_execute_ddl", CHAIN1, txId)
		panicNotEqual(result, "")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("\n// 2、SQL中，禁止跨数据库操作，无需指定数据库名。比如select * from db.table 是禁止的； use db;是禁止的。")
		_, result = testInvokeSqlCommon(sk3, client, "sql_dbname_table_name", CHAIN1, txId)
		panicNotEqual(result, "")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("\n// 3、SQL中，禁止使用事务相关操作的语句，比如commit 、rollback等，事务由ChainMaker框架自动控制。")
		_, result = testInvokeSqlCommon(sk3, client, "sql_execute_commit", CHAIN1, txId)
		panicNotEqual(result, "")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("\n// 4、SQL中，禁止使用随机数、获得系统时间等不确定性函数，这些函数在不同节点产生的结果可能不一样，导致合约执行结果无法达成共识。")
		_, result = testInvokeSqlCommon(sk3, client, "sql_random_key", CHAIN1, txId)
		panicNotEqual(result, "")
		_, result = testInvokeSqlCommon(sk3, client, "sql_random_str", CHAIN1, txId)
		panicNotEqual(result, "")
		_, result = testInvokeSqlCommon(sk3, client, "sql_random_query_str", CHAIN1, txId)
		panicNotEqual(result, "ok")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("\n// 5、SQL中，禁止多条SQL拼接成一个SQL字符串传入。")
		_, result = testInvokeSqlCommon(sk3, client, "sql_multi_sql", CHAIN1, txId)
		panicNotEqual(result, "")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("\n// 7、禁止建立、修改或删除表名为“state_infos”的表，这是系统自带的提供KV数据存储的表，用于存放PutState函数对应的数据。")
		_, result = testInvokeSqlCommon(sk3, client, "sql_update_state_info", CHAIN1, txId)
		panicNotEqual(result, "")
		_, result = testInvokeSqlCommon(sk3, client, "sql_query_state_info", CHAIN1, txId)
		panicNotEqual(result, "")
	}

	if runtimeType == commonPb.RuntimeType_GASM {
		type FuncWithTxid struct {
			txIdMyself string
			funcName   string
		}
		txIds := []FuncWithTxid{}
		txIds = append(txIds, FuncWithTxid{
			InvokeCreatetable(sk3, client, CHAIN1), //1.创建表
			"InvokeCreatetable",
		})
		txIds = append(txIds, FuncWithTxid{
			InvokeCreatedb(sk3, client, CHAIN1), //2.跨库操作，初始函数
			"InvokeCreatedb",
		})
		txIds = append(txIds, FuncWithTxid{
			InvokeCommit(sk3, client, CHAIN1), //3.commit，rollback
			"InvokeCommit",
		})
		txIds = append(txIds, FuncWithTxid{
			InvokeUnpredictableSql(sk3, client, CHAIN1), //4.随机数
			"InvokeUnpredictableSql",
		})
		txIds = append(txIds, FuncWithTxid{
			InvokeDoubleSql(sk3, client, CHAIN1), //5.多条sql
			"InvokeDoubleSql",
		})
		txIds = append(txIds, FuncWithTxid{
			InvokeAuoIncrement(sk3, client, CHAIN1), //6。自增主键,在初始化函数执行
			"InvokeAuoIncrement",
		})
		txIds = append(txIds, FuncWithTxid{
			InvokeCreateuesr(sk3, client, CHAIN1), //8。禁止DCL语句，GRANT，REVOKE，初始函数
			"InvokeCreateuesr",
		})

		time.Sleep(2 * time.Second)
		for _, testFunc := range txIds {
			txId = testFunc.txIdMyself
			time.Sleep(time.Millisecond * 500)
			testWaitTx(sk3, client, CHAIN1, txId)
			//testCreate(sk3, client, CHAIN1)
			resultInfo := testGetTxByTxId(sk3, client, txId, CHAIN1)
			if resultInfo.Transaction.Result.Code == commonPb.TxStatusCode_SUCCESS {
				fmt.Printf("%s", testFunc.funcName)
				panic("执行%s成功，但该方法是被禁止的，发生错误")
			} else {
				fmt.Printf("%s-校验成功,%s\n", testFunc.funcName, resultInfo.Transaction.Result.ContractResult.Message)
			}
		}
	}
	fmt.Println("\nfinal result: ", txId, result, rs, id)
	fmt.Println("test success!!!")
	fmt.Println("test success!!!")
}
func initWasmerTest() {
	WasmPath = "../wasm/rust-fact-1.0.0.wasm"
	WasmUpgradePath = "../wasm/rust-func-verify-1.0.0.wasm"
	contractName = "contract0001"
	runtimeType = commonPb.RuntimeType_WASMER
}
func initWasmerSqlTest() {
	WasmPath = "../wasm/rust-sql-1.1.1.wasm"
	WasmUpgradePath = "../wasm/rust-sql-1.1.1.wasm"
	contractName = "contract100"
	runtimeType = commonPb.RuntimeType_WASMER
}
func initGasmTest() {
	WasmPath = "../wasm/go-sql-1.1.1.wasm"
	WasmUpgradePath = "../wasm/go-sql-1.1.1.wasm"
	contractName = "contract200"
	runtimeType = commonPb.RuntimeType_GASM
}
func initWxwmTest() {
	WasmPath = "../wasm/cpp-func-verify-1.0.0.wasm"
	WasmUpgradePath = "../wasm/cpp-func-verify-1.0.0.wasm"
	contractName = "contract300"
	runtimeType = commonPb.RuntimeType_WXVM
}
func testCreate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {

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
	return txId
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

func InvokePrintHello(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "printhello",
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

func InvokeDoubleSql(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "doubleSql",
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

func InvokeUnpredictableSql(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "unpredictableSql",
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

func InvokeCreatetable(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "createTable",
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

func InvokeCreatedb(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "createDb",
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

func InvokeCreateuesr(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "createUser",
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

func InvokeAuoIncrement(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[auoIncrement] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "autoIncrementSql",
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

func InvokeCommit(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       "commitSql",
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

func testGetTxByTxId(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txId, chainId string) *commonPb.TransactionInfo {
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Println("========get tx by txId ", txId, "===============")
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Printf("\n============ get tx by txId [%s] ============\n", txId)

	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: "txId", Value: txId}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)

	result := &commonPb.TransactionInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, result)
	if err != nil {
		panic(err)
	}
	return result
}

func testWaitTx(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, txId string) {
	fmt.Printf("\n============ testWaitTx [%s] ============\n", txId)
	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: "txId", Value: txId}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)
	if resp == nil || resp.ContractResult == nil || strings.Contains(resp.Message, "no such transaction") {
		time.Sleep(time.Second * 2)
		testWaitTx(sk3, client, chainId, txId)
	} else if resp != nil && len(resp.Message) != 0 {
		fmt.Println(resp.Message)
	}
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
			Value: "长安链chainmaker_update",
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

func testInvokeSqlCommon(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, method string, chainId string, id string) (string, string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ common contract %s[%s] [%s] ============\n", contractName, method, id)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: id,
		},
		{
			Key:   "name",
			Value: "长安链chainmaker_update",
		},
	}
	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId, string(resp.ContractResult.Result)
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
			Value: "chainmaker_save_point",
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
			Value: "4",
		},
		{
			Key:   "max_age",
			Value: "7",
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

	//fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	//fmt.Println(string(resp.ContractResult.Result))
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
			Value: "4",
		},
		{
			Key:   "min_age",
			Value: "1",
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
func panicNotEqual(a string, b string) {
	if a != b {
		panic(a + " not equal " + b)
	}
}
