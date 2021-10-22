/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

/*
sql rust test/wasm/rust-func-verify-2.0.0.wasm 源码所在目录：chainmaker-contract-sdk-rust - v2.0.0_dev - src/contract_functional_verify.rs
sql tinygo go-test/wasm/go-func-verify-2.0.0.wasm 源码所在目录：chainmaker-contract-sdk-tinygo - develop - demo/main_functional_verify.go
*/
package main

import (
	"context"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/test/common"
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/evmutils"
	evm "chainmaker.org/chainmaker/common/v2/evmutils"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	discoveryPb "chainmaker.org/chainmaker/pb-go/v2/discovery"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	logTempUnmarshalBlockInfoFailed = "blockInfo unmarshal error %s\n"
	logTempSendTx                   = "send tx resp: code:%d, msg:%s, txid:%s, payload:%+v\n"
	logTempSendBlock                = "send tx resp: code:%d, msg:%s, blockInfo:%+v\n"
	fieldWithRWSet                  = "withRWSet"
)

const (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12301
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
	evmtest()
	initWasmerTest()
	runTest()
}

func runTest() {
	var (
		conn   *grpc.ClientConn
		client *apiPb.RpcNodeClient
		sk3    crypto.PrivateKey
		err    error
		txId   string
	)
	// init
	{
		conn, err = initGRPCConnect(true)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()
		c := apiPb.NewRpcNodeClient(conn)
		client = &c

		file, err := ioutil.ReadFile(userKeyPath)
		if err != nil {
			panic(err)
		}

		sk3, err = asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			panic(err)
		}
	}
	// 1) 合约创建
	txId = testCreate(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 2) 执行合约
	testUpgradeInvokeSum(sk3, client, CHAIN1) // method [sum] not export, 合约升级后则有

	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 3) 合约查询
	_, result := testQueryFindByHash(sk3, client, CHAIN1)
	if string(result) != "{\"file_hash\":\"b4018d181b6f\",\"file_name\":\"长安链chainmaker\",\"time\":\"1615188470000\"}" {
		fmt.Println("query result:", string(result))
		log.Panicf("query error")
	} else {
		fmt.Println("    【testQueryFindByHash】 pass")
	}

	// 4) 根据TxId查交易
	testGetTxByTxId(sk3, client, txId, CHAIN1)

	// 5) 根据区块高度查区块，若height为max，表示查当前区块
	hash := testGetBlockByHeight(sk3, client, CHAIN1, math.MaxUint64)

	// 6) 根据区块高度查区块（包含读写集），若height为-1，表示查当前区块
	testGetBlockWithTxRWSetsByHeight(sk3, client, CHAIN1, math.MaxUint64)

	// 7) 根据区块哈希查区块
	testGetBlockByHash(sk3, client, CHAIN1, hash)

	// 8) 根据区块哈希查区块（包含读写集）
	testGetBlockWithTxRWSetsByHash(sk3, client, CHAIN1, hash)

	// 9) 根据TxId查区块
	testGetBlockByTxId(sk3, client, txId, CHAIN1)

	// 10) 查询最新配置块
	testGetLastConfigBlock(sk3, client, CHAIN1)

	// 11) 查询最新区块
	testGetLastBlock(sk3, client, CHAIN1)

	// 12) 查询链信息
	testGetChainInfo(sk3, client, CHAIN1)

	// 13) 合约升级
	txId = testUpgrade(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)
	// 14) 合约执行
	testUpgradeInvokeSum(sk3, client, CHAIN1)

	// 15) 批量执行
	txId = testInvokeFactSave(sk3, client, CHAIN1)

	testWaitTx(sk3, client, CHAIN1, txId)
	txId = testPerformanceModeTransfer(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 16) 功能测试
	txId = testInvokeFunctionalVerify(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 17) KV迭代器测试
	testKvIterator(sk3, client)

	// 18) 冻结、解冻、吊销用户合约功能测试
	testFreezeOrUnfreezeOrRevokeFlow(sk3, client)

	fmt.Println("    【runTest】 pass", "txId", txId)
}
func initWasmerTest() {
	WasmPath = "../wasm/rust-func-verify-2.0.0.wasm"
	WasmUpgradePath = WasmPath
	contractName = "contract101"
	runtimeType = commonPb.RuntimeType_WASMER
	printConfig("wasmer")
}

func printConfig(wasmType string) {
	fmt.Printf("=========init %s=========\n", wasmType)
	fmt.Println("  wasm path         : ", WasmPath)
	fmt.Println("  wasm upgrade path : ", WasmUpgradePath)
	fmt.Println("  wasm contract name: ", contractName)
	fmt.Println("  wasm runtime type : ", runtimeType)
	fmt.Println()
}

func testKvIterator(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient) {
	txId := testInvokeMethod(sk3, client, "test_put_state")
	testWaitTx(sk3, client, CHAIN1, txId)

	r := testQueryMethod(sk3, client, "test_kv_iterator")

	if "15" != string(r) {
		panic("testKvIterator error count!=15 count=" + string(r))
	} else {
		fmt.Println("    【testKvIterator】 pass")
	}
}
func testPerformanceModeTransfer(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	fmt.Println("==============================================")
	fmt.Println("==============================================")
	fmt.Println("==============start batch invoke==============")
	fmt.Println("==============================================")
	fmt.Println("==============================================")
	start := utils.CurrentTimeMillisSeconds()
	wg := sync.WaitGroup{}
	txId := ""
	for j := 0; j < 10; j++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				txId = testInvokeFactSave(sk3, client, CHAIN1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	end := utils.CurrentTimeMillisSeconds()
	spend := end - start
	fmt.Println("发送100个交易所花时间", spend, "ms")
	return txId
}
func testFreezeOrUnfreezeOrRevokeFlow(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("==============test freeze unfreeze revoke flow==============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	txId := ""
	//执行合约
	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testQueryFindByHash(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 冻结
	txId = common.FreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testQueryFindByHash(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 解冻
	txId = common.UnfreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testQueryFindByHash(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 冻结
	txId = common.FreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testQueryFindByHash(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 解冻
	txId = common.UnfreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testQueryFindByHash(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	// 冻结
	txId = common.FreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	// 吊销
	txId = common.RevokeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = testInvokeFactSave(sk3, client, CHAIN1)
	testQueryFindByHash(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = common.FreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_FREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
	txId = common.UnfreezeContract(sk3, client, CHAIN1, contractName, runtimeType)
	//testFreezeOrUnfreezeOrRevoke(sk3, client, CHAIN1, syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String())
	testWaitTx(sk3, client, CHAIN1, txId)
}

func testGetTxByTxId(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txId, chainId string) []byte {
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Println("========get tx by txId ", txId, "===============")
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")

	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: "txId", Value: []byte(txId)}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes, nil)

	result := &commonPb.TransactionInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, result)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v", result)
	if result.Transaction.Result.Code != 0 {
		panic(result.Transaction.Result.ContractResult.Message)
	}
	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, result.Transaction.Result.ContractResult)
	fmt.Println("GasUsed：", result.Transaction.Result.ContractResult.GasUsed)
	fmt.Println("Message：", result.Transaction.Result.ContractResult.Message)
	fmt.Println("Result：", result.Transaction.Result.ContractResult.Result)
	fmt.Println("Code：", result.Transaction.Result.ContractResult.Code)
	return result.Transaction.Result.ContractResult.Result
}

func testGetBlockByTxId(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txId, chainId string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block by txId ", txId, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "txId",
			Value: []byte(txId),
		},
		{
			Key:   fieldWithRWSet,
			Value: []byte("false"),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_BY_TX_ID", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetBlockByHeight(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, height uint64) string {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block by height ", height, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get block by height [%d] ============\n", height)
	// 构造Payload

	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: []byte(strconv.FormatUint(height, 10)),
		},
		{
			Key:   fieldWithRWSet,
			Value: []byte("false"),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_BY_HEIGHT", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)

	return hex.EncodeToString(blockInfo.Block.Header.BlockHash)
}

func testGetBlockWithTxRWSetsByHeight(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, height uint64) string {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block with txRWsets by height ", height, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get block with txRWsets by height [%d] ============\n", height)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: []byte(strconv.FormatUint(height, 10)),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_WITH_TXRWSETS_BY_HEIGHT", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)

	return hex.EncodeToString(blockInfo.Block.Header.BlockHash)
}

func testGetBlockByHash(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, hash string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block by hash ", hash, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get block by hash [%s] ============\n", hash)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: []byte(hash),
		},
		{
			Key:   fieldWithRWSet,
			Value: []byte("false"),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_BY_HASH", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetBlockWithTxRWSetsByHash(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, hash string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("========get block with txRWsets by hash ", hash, "===============")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Printf("\n============ get block with txRWsets by hash [%s] ============\n", hash)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: []byte(hash),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_BLOCK_WITH_TXRWSETS_BY_HASH", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetLastConfigBlock(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("====================get last config block===================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   fieldWithRWSet,
			Value: []byte("true"),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_LAST_CONFIG_BLOCK", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetLastBlock(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("=======================get last block=======================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   fieldWithRWSet,
			Value: []byte("true"),
		},
	}

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_LAST_BLOCK", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, nil)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf(logTempUnmarshalBlockInfoFailed, err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message)
}

func testGetChainInfo(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	fmt.Println("=======================get chain info=======================")
	fmt.Println("============================================================")
	fmt.Println("============================================================")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}

	payload := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_CHAIN_INFO", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payload, nil)

	chainInfo := &discoveryPb.ChainInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, chainInfo)
	if err != nil {
		fmt.Printf("chainInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf(logTempSendBlock, resp.ContractResult.Code, resp.ContractResult.Message, chainInfo)
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

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return resp.TxId
}

var fileHash = "b4018d181b6f"

func testUpgradeInvokeSum(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[sum][%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "arg1",
			Value: []byte("1"),
		},
		{
			Key:   "arg2",
			Value: []byte("2"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "sum",
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload, nil)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return txId
}
func testInvokeFactSave(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[save] [%s] ============\n", contractName, txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
		{
			Key:   "time",
			Value: []byte("1615188470000"),
		},
		{
			Key:   "file_name",
			Value: []byte("长安链chainmaker"),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "save",
		Parameters:   pairs,
	}

	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	log.Fatalf(logTempMarshalPayLoadFailed, err.Error())
	//}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload, nil)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return txId
}

func testInvokeMethod(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, method string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[%s] [%s] ============\n", contractName, method, txId)

	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		CHAIN1, txId, payload, nil)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return txId
}
func testQueryMethod(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, method string) []byte {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[%s] [%s] ============\n", contractName, method, txId)

	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, txId, payload, nil)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return resp.ContractResult.Result
}

func testInvokeFunctionalVerify(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract %s[functional_verify] [%s] [functional_verify] ============\n", contractName, txId)

	// 构造Payload
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "contract_name",
			Value: []byte(contractName),
		},
	}
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "functional_verify",
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload, nil)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	return txId
}

func testQueryFindByHash(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) (string, []byte) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract %s[find_by_file_hash] fileHash=%s ============\n", contractName, fileHash)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "find_by_file_hash",
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload, nil)

	fmt.Printf(logTempSendTx, resp.Code, resp.Message, resp.TxId, resp.ContractResult)
	fmt.Println(string(resp.ContractResult.Result))
	//items := serialize.EasyUnmarshal(resp.ContractResult.Result)
	//for _, item := range items {
	//	fmt.Println(item.Key, item.Value)
	//}
	return txId, resp.ContractResult.Result
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

func testWaitTx(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, txId string) {
	fmt.Printf("\n============ testWaitTx [%s] ============\n", txId)
	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: "txId", Value: []byte(txId)}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := common.ConstructQueryPayload(syscontract.SystemContract_CHAIN_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := common.ProposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payloadBytes, nil)
	if resp == nil || resp.ContractResult == nil || strings.Contains(resp.Message, "no such transaction") {
		time.Sleep(time.Second * 2)
		testWaitTx(sk3, client, chainId, txId)
	} else if resp != nil && len(resp.Message) != 0 {
		fmt.Println(resp.Message)
	}
}

///////////////////////////////EVM/////////////////////////////

const (
	ByteCodeHexPath = "../../test/wasm/evm-token.hex"
	ByteCodePath    = "../../test/wasm/evm-token.bin"
	ABIPath         = "../../test/wasm/evm-token.abi"

	adminCrtPath = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"
)

var AbiJson = ""

func evmtest() {
	runtimeType = commonPb.RuntimeType_EVM

	contractAddr, _ := evmutils.MakeAddressFromString("cont_01")
	//contractName    = contractAddr.String()
	contractName = hex.EncodeToString(contractAddr.Bytes())
	fmt.Println("contractName:", contractName)

	flag.Parse()
	common.SetCertPathPrefix(certPathPrefix)

	conn, err := initGRPCConnect(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	c := apiPb.NewRpcNodeClient(conn)
	client := &c

	file, err := ioutil.ReadFile(userKeyPath)
	if err != nil {
		panic(err)
	}

	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("---------------A(User1) 创建ERC20合约-------------")
	txId := testCreateEvm(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)
	fmt.Println("---------------查询A(User1) B(Admin1)账户余额-------------")

	balanceA := testQueryBalance(sk3, client, CHAIN1, userCrtPath)
	if balanceA != "1000000000000000000000000000" {
		fmt.Println("balance A not equal 1000000000000000000000000000 will skip evmtest for later fix")
		return
	}
	balanceB := testQueryBalance(sk3, client, CHAIN1, adminCrtPath)
	if balanceB != "0" {
		panic("balance B not equal 0")
	}
	fmt.Println("---------------发起User1给Admin1的转账-------------")
	txId = testTransfer(sk3, client, CHAIN1)
	testWaitTx(sk3, client, CHAIN1, txId)

	fmt.Println("---------------查询AB账户余额-------------")
	balanceA = testQueryBalance(sk3, client, CHAIN1, userCrtPath)
	if balanceA != "999999999999999999999999990" {
		panic("balance A not equal 999999999999999999999999990")
	}
	balanceB = testQueryBalance(sk3, client, CHAIN1, adminCrtPath)
	if balanceB != "10" {
		panic("balance B not equal 10")
	}
}

func convertHex2Bin(hexPath, binPath string) error {
	hexBytes, err := ioutil.ReadFile(hexPath)
	if err != nil {
		return err
	}
	bin, err := hex.DecodeString(string(hexBytes))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(binPath, bin, 777)
}

func testCreateEvm(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	convertHex2Bin(ByteCodeHexPath, ByteCodePath)
	abi, err := ioutil.ReadFile(ABIPath)
	if err != nil {
		panic(err.Error())
	}
	AbiJson = string(abi)
	return common.CreateContract(sk3, client, chainId, contractName, ByteCodePath, runtimeType)
}

func testQueryBalance(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, certPath string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson))
	method0 := "balanceOf"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {

		method = method0

		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		client1Addr, err := getSKI(certPath)
		checkErr(err)
		fmt.Printf("User1 SKI:%s\n", client1Addr)
		addr, err := evm.MakeAddressFromHex(client1Addr)

		dataByte, err := myAbi.Pack(method, evm.BigToAddress(addr))

		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(data),
			},
		}

	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload)
	result := ""
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
		if len(v) > 0 {
			result = fmt.Sprintf("%v", v[0])
		}
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	return result
}
func getSKI(certPath string) (string, error) {
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("read cert file [%s] failed, %s", certPath, err)
	}

	block, rest := pem.Decode(certBytes)
	if len(rest) != 0 {
		return "", errors.New("pem.Decode failed, invalid cert")
	}
	cert, err := bcx509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parseCertificate cert failed, %s", err)
	}

	ski := hex.EncodeToString(cert.SubjectKeyId)
	return ski, nil
}

func testTransfer(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson))
	method0 := "transfer"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		method = method0
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		toSki, err := getSKI(adminCrtPath)
		checkErr(err)
		addr, err := evm.MakeAddressFromHex(toSki)
		checkErr(err)
		dataByte, err := myAbi.Pack(method, evm.BigToAddress(addr), big.NewInt(10))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(data),
			},
		}

	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func proposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payload *commonPb.Payload) *commonPb.TxResponse {

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
	sender := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		//IsFullCert: true,
		//MemberInfo: []byte(pubKeyString),
	}
	payload.ChainId = chainId
	payload.TxType = txType
	payload.TxId = txId
	payload.Timestamp = time.Now().Unix()

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
		if ok {
			if statusErr.Code() == codes.DeadlineExceeded {
				fmt.Println("WARN: client.call err: deadline")
				os.Exit(0)
			}
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

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
