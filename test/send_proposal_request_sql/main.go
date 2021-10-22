/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
sql rust test/wasm/rust-sql-2.0.0.wasm 源码所在目录：chainmaker-contract-sdk-rust v2.0.0_dev src/contract_fact_sql.rs
sql tinygo go-test/wasm/sql-2.0.0.wasm 源码所在目录：chainmaker-contract-sdk-tinygo develop demo/main_fact_sql.go

*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"chainmaker.org/chainmaker-go/test/common"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	userKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key"
	userCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
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
	common.SetCertPathPrefix(certPathPrefix)

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
		rs     = make(map[string][]byte, 0)
	)

	fmt.Println("//1) 合约创建")
	txId = testCreate(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("// 2) 执行合约-sql insert")
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "11", true)
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "11", true)

	for i := 0; i < 10; i++ {
		txId = testInvokeSqlInsert(sk3, client, CHAIN1, strconv.Itoa(i), false)
	}
	testWaitTx(sk3, client, CHAIN1, txId)
	id = txId

	fmt.Println("// 3) 查询 age11的 id:" + id)
	_, result = testQuerySqlById(sk3, client, CHAIN1, id)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlUpdate query", rs)
	if string(rs["id"]) != id {
		fmt.Println("result", rs)
		panic("query by id error, id err")
	} else {
		fmt.Println("  【testInvokeSqlInsert】 pass")
		fmt.Println("  【testQuerySqlById】 pass")
	}

	fmt.Println("// 4) 执行合约-sql update name=长安链chainmaker_update where id=" + id)
	txId = testInvokeSqlUpdate(sk3, client, CHAIN1, id)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("// 5) 查询 id=" + id + " 看name是不是更新成了长安链chainmaker_update：")
	_, result = testQuerySqlById(sk3, client, CHAIN1, id)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlUpdate query", rs)
	if string(rs["name"]) != "长安链chainmaker_update" {
		fmt.Println("result", rs)
		panic("query update result error")
	} else {
		fmt.Println("  【testInvokeSqlUpdate】 pass")
	}

	fmt.Println("// 6) 范围查询 rang age 1~10")
	testQuerySqlRangAge(sk3, client, CHAIN1)

	fmt.Println("// 7) 执行合约-sql delete by id age=11")
	txId = testInvokeSqlDelete(sk3, client, CHAIN1, id)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("// 8) 再次查询 id age=11，应该查不到")
	_, result = testQuerySqlById(sk3, client, CHAIN1, id)
	if result != "{}" {
		fmt.Println("result", result)
		panic("查询结果错误")
	} else {
		fmt.Println("  【testInvokeSqlDelete】 pass")
	}
	//// 9) 跨合约调用
	txId = testCrossCall(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 10) 交易回退
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "2000", true)
	testWaitTx(sk3, client, CHAIN1, txId)
	id = txId
	for i := 0; i < 3; i++ {
		fmt.Println("试图将id=" + id + " 的name改为长安链chainmaker_save_point，但是发生了错误，所以修改不会成功")
		txId = testInvokeSqlUpdateRollbackDbSavePoint(sk3, client, CHAIN1, id)
		testWaitTx(sk3, client, CHAIN1, txId)

		fmt.Println("// 11 再次查询age=2000的这条数据，如果name被更新了，那么说明savepoint Rollback失败了")
		_, result = testQuerySqlById(sk3, client, CHAIN1, id)
		rs = make(map[string][]byte, 0)
		json.Unmarshal([]byte(result), &rs)
		fmt.Println("testInvokeSqlUpdateRollbackDbSavePoint query", rs)
		if string(rs["name"]) == "chainmaker_save_point" {
			panic("testInvokeSqlUpdateRollbackDbSavePoint test 【fail】 query by id error, age err")
		} else if string(rs["name"]) == "长安链chainmaker" {
			fmt.Println("  【testInvokeSqlUpdateRollbackDbSavePoint】 pass")
		} else {
			panic("error result")
		}
	}

	// 9) 升级合约
	txId = testUpgrade(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 10) 升级合约后执行插入
	txId = testInvokeSqlInsert(sk3, client, CHAIN1, "100000", true)

	testWaitTx(sk3, client, CHAIN1, txId)
	_, result = testQuerySqlById(sk3, client, CHAIN1, txId)
	rs = make(map[string][]byte, 0)
	json.Unmarshal([]byte(result), &rs)
	fmt.Println("testInvokeSqlInsert query", rs)
	if string(rs["age"]) != "100000" {
		panic("query by id error, age err")
	} else {
		fmt.Println("  【testUpgrade】 pass")
		fmt.Println("  【testInvokeSqlInsert】 pass")
	}

	// 并发测试
	for i := 500; i < 600; i++ {
		txId = testInvokeSqlInsert(sk3, client, CHAIN1, strconv.Itoa(i), false)
	}
	testWaitTx(sk3, client, CHAIN1, txId)

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
func initWasmerSqlTest() {
	WasmPath = "../wasm/rust-sql-2.0.0.wasm"
	WasmUpgradePath = WasmPath
	contractName = "contract110"
	runtimeType = commonPb.RuntimeType_WASMER
}
func initGasmTest() {
	WasmPath = "../wasm/go-sql-2.0.0.wasm"
	WasmUpgradePath = WasmPath
	contractName = "contract210"
	runtimeType = commonPb.RuntimeType_GASM
}
func testCreate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	return common.CreateContract(sk3, client, chainId, contractName, WasmPath, runtimeType)
}

func testUpgrade(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========================test upgrade========================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")

	resp := common.UpgradeContract(sk3, client, chainId, contractName, WasmUpgradePath, runtimeType)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return resp.TxId
}

func testInvokeSqlInsert(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, age string, print bool) string {
	txId := utils.GetRandTxId()
	if print {
		fmt.Printf("\n============ invoke contract %s[sql_insert] [%s,%s] ============\n", contractName, txId, age)
	}
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "id",
			Value: []byte(txId),
		},
		{
			Key:   "age",
			Value: []byte(age),
		},
		{
			Key:   "name",
			Value: []byte("长安链chainmaker"),
		},
		{
			Key:   "id_card_no",
			Value: []byte("510623199202023323"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_insert",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)
	if print {
		fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	}
	return txId
}

func InvokePrintHello(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "printhello",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeDoubleSql(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "doubleSql",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeUnpredictableSql(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "unpredictableSql",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeCreatetable(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "createTable",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeCreatedb(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "createDb",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeCreateuesr(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "createUser",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeAuoIncrement(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[auoIncrement] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "autoIncrementSql",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func InvokeCommit(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sql_insert] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "commitSql",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

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
	pair := &commonPb.KeyValuePair{Key: "txId", Value: []byte(txId)}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := constructPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
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
	pair := &commonPb.KeyValuePair{Key: "txId", Value: []byte(txId)}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := constructPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes)
	if resp == nil || resp.ContractResult == nil || strings.Contains(resp.Message, "no such transaction") {
		time.Sleep(time.Second)
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
			Value: []byte(id),
		},
		{
			Key:   "name",
			Value: []byte("长安链chainmaker_update"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_update",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

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
			Value: []byte(id),
		},
		{
			Key:   "name",
			Value: []byte("长安链chainmaker_update"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload)

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
			Value: []byte(id),
		},
		{
			Key:   "name",
			Value: []byte("chainmaker_save_point"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_update_rollback_save_point",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

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
			Value: []byte(contractName),
		},
		{
			Key:   "min_age",
			Value: []byte("4"),
		},
		{
			Key:   "max_age",
			Value: []byte("7"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_cross_call",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

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
			Value: []byte(id),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_delete",
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)

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
			Value: []byte(id),
		},
	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_query_by_id",
		Parameters:   pairs,
	}

	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	//}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload)

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
			Value: []byte("4"),
		},
		{
			Key:   "min_age",
			Value: []byte("1"),
		},
	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sql_query_range_of_age",
		Parameters:   pairs,
	}

	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	//}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.ContractResult)
	fmt.Println(string(resp.ContractResult.Result))
	//items := serialize.EasyUnmarshal(resp.ContractResult.Result)
	//for _, item := range items {
	//	fmt.Println(item.Key, item.Value)
	//}
	return txId
}

func proposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payload *commonPb.Payload) *commonPb.TxResponse {
	payload.ChainId = chainId
	payload.TxType = txType
	payload.Timestamp = time.Now().Unix()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	if txId == "" {
		txId = utils.GetRandTxId()

	}
	payload.TxId = txId

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
	//header := &commonPb.Payload{
	//	ChainId:        chainId,
	//	TxType:         txType,
	//	TxId:           txId,
	//	Timestamp:      time.Now().Unix(),
	//	ExpirationTime: 0,
	//}

	req := &commonPb.TxRequest{
		Payload: payload,
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
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	//signBytes, err := signer.Sign("SM3", rawTxBytes)
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

	signer, err := accesscontrol.NewCertSigningMember("", sender, skPEM, "")
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

func constructPayload(contractName, method string, pairs []*commonPb.KeyValuePair) *commonPb.Payload {
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	return payload
}

//func acSign(msg *commonPb.Payload, orgIdList []int) ([]*commonPb.EndorsementEntry, error) {
//	msg.Endorsement = nil
//	bytes, _ := proto.Marshal(msg)
//
//	signers := make([]protocol.SigningMember, 0)
//	for _, orgId := range orgIdList {
//
//		numStr := strconv.Itoa(orgId)
//		path := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.key"
//		file, err := ioutil.ReadFile(path)
//		if err != nil {
//			panic(err)
//		}
//		sk, err := asym.PrivateKeyFromPEM(file, nil)
//		if err != nil {
//			panic(err)
//		}
//
//		userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
//		file2, err := ioutil.ReadFile(userCrtPath)
//		//fmt.Println("node", orgId, "crt", string(file2))
//		if err != nil {
//			panic(err)
//		}
//
//		// 获取peerId
//		_, err = helper.GetLibp2pPeerIdFromCert(file2)
//		//fmt.Println("node", orgId, "peerId", peerId)
//
//		// 构造Sender
//		sender1 := &acPb.Member{
//			OrgId:      "wx-org" + numStr + ".chainmaker.org",
//			MemberInfo: file2,
//			//IsFullCert: true,
//		}
//
//		signer := getSigner(sk, sender1)
//		signers = append(signers, signer)
//	}
//
//	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, "SHA256")
//}
func panicNotEqual(a string, b string) {
	if a != b {
		panic(a + " not equal " + b)
	}
}
