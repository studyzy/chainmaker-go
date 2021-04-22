package main

import (
	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/common/ca"
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	evm "chainmaker.org/chainmaker-go/common/evmutils"
	"chainmaker.org/chainmaker-go/common/helper"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	discoveryPb "chainmaker.org/chainmaker-go/pb/protogo/discovery"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	MAX_CNT        = 1
	CHAIN1         = "chain1"
	CHAIN2         = "chain2"
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "../../config"
	WasmPath       = "../../test/wasm/fact-rust-0.7.2.wasm"
	AbiJson        = "[{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newBalance\",\"type\":\"uint256\"},{\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"updateBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newBalance\",\"type\":\"uint256\"}],\"name\":\"updateMyBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_addressFounder\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"
	AbiJson1       = "[{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newBalance\",\"type\":\"uint256\"},{\"name\":\"_to\",\"type\":\"address\"}],\"name\":\"updateBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newBalance\",\"type\":\"uint256\"}],\"name\":\"updateMyBalance\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_addressFounder\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balances\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"
	ByteCodePath   = "../../test/wasm/evm-token.bin"
	ByteCodePath1  = "../../test/wasm/evm-token.bin"
	userKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key"
	userCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	adminKeyPath   = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.key"
	adminCrtPath   = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.tls.crt"

	orgId         = "wx-org1.chainmaker.org"
	contractName  = "contract92"
	contractName1 = "contract92"
	runtimeType   = commonPb.RuntimeType_EVM
	prePathFmt    = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"

	client1Addr = "1087848554046178479107522336262214072175637027873"
)

//var caPaths = []string{certPathPrefix + "/certs/wx-org1/ca"}
var caPaths = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}

func main() {
	flag.Parse()

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

	testCreate(sk3, &client, CHAIN1)
	time.Sleep(4 * time.Second)
	testQuery(sk3, &client, CHAIN1)
	testQuery2(sk3, &client, CHAIN1)

	testSet(sk3, &client, CHAIN1)
	time.Sleep(4 * time.Second)
	testQuery(sk3, &client, CHAIN1)

	testSet2(sk3, &client, CHAIN1)
	time.Sleep(4 * time.Second)
	testQuery2(sk3, &client, CHAIN1)

	testTransfer(sk3, &client, CHAIN1)
	time.Sleep(4 * time.Second)
	testQuery(sk3, &client, CHAIN1)
	testQuery2(sk3, &client, CHAIN1)

}

func testPerformanceModeTransfer(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	for j := 0; j < 5000; j++ {
		i := j % 1000
		txId := utils.GetRandTxId()
		// 构造Payload
		pairs := []*commonPb.KeyValuePair{
			{
				Key:   "from",
				Value: strconv.Itoa(i),
			},
			{
				Key:   "to",
				Value: strconv.Itoa(i + 1000),
			},
			{
				Key:   "amount",
				Value: "1",
			},
		}

		payload := &commonPb.TransactPayload{
			ContractName: contractName,
			Method:       "transfer",
			Parameters:   pairs,
		}

		payloadBytes, err := proto.Marshal(payload)
		if err != nil {
			log.Fatalf("marshal payload failed, %s", err.Error())
		}

		resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
			chainId, txId, payloadBytes)

		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	}
}

func testPerformanceModeBalance(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	for i := 0; i < 2000; i++ {
		txId := utils.GetRandTxId()
		// 构造Payload
		pairs := []*commonPb.KeyValuePair{
			{
				Key:   "from",
				Value: strconv.Itoa(i),
			},
		}

		payload := &commonPb.TransactPayload{
			ContractName: contractName,
			Method:       "balance",
			Parameters:   pairs,
		}

		payloadBytes, err := proto.Marshal(payload)
		if err != nil {
			log.Fatalf("marshal payload failed, %s", err.Error())
		}

		resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
			chainId, txId, payloadBytes)

		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	}
}

func testFreezeOrUnfreezeOrRevokeFlow(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	//执行合约
	testInvoke(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	// 升级合约
	testUpgrade(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)
	testInvoke(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	// 冻结
	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_FREEZE_CONTRACT.String())
	time.Sleep(5 * time.Second)
	testInvoke(sk3, &client, CHAIN1)
	//testInvoke2(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	// 解冻
	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_UNFREEZE_CONTRACT.String())
	time.Sleep(5 * time.Second)
	testInvoke(sk3, &client, CHAIN1)
	//testInvoke2(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	// 冻结
	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_FREEZE_CONTRACT.String())
	time.Sleep(5 * time.Second)
	testInvoke(sk3, &client, CHAIN1)
	//testInvoke2(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	// 解冻
	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_UNFREEZE_CONTRACT.String())
	time.Sleep(5 * time.Second)
	testInvoke(sk3, &client, CHAIN1)
	//testInvoke2(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	// 冻结
	//testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_FREEZE_CONTRACT.String())
	//time.Sleep(5 * time.Second)
	// 吊销
	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_REVOKE_CONTRACT.String())
	time.Sleep(5 * time.Second)
	testInvoke(sk3, &client, CHAIN1)
	//testInvoke2(sk3, &client, CHAIN1)
	time.Sleep(5 * time.Second)

	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_FREEZE_CONTRACT.String())
	time.Sleep(5 * time.Second)

	testFreezeOrUnfreezeOrRevoke(sk3, &client, CHAIN1, commonPb.ManageUserContractFunction_UNFREEZE_CONTRACT.String())
	time.Sleep(5 * time.Second)
}

func testGetTxByTxId(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txId, chainId string) {
	fmt.Printf("\n============ get tx by txId [%s] ============\n", txId)

	// 构造Payload
	pair := &commonPb.KeyValuePair{Key: "txId", Value: txId}
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, pair)

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_TX_BY_TX_ID", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
}

func testGetBlockByTxId(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txId, chainId string) {
	fmt.Printf("\n============ get block by txId [%s] ============\n", txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "txId",
			Value: txId,
		},
		{
			Key:   "withRWSet",
			Value: "false",
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_BLOCK_BY_TX_ID", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, txId, payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetBlockByHeight(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, height int64) string {
	fmt.Printf("\n============ get block by height [%d] ============\n", height)
	// 构造Payload

	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: strconv.FormatInt(height, 10),
		},
		{
			Key:   "withRWSet",
			Value: "false",
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_BLOCK_BY_HEIGHT", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)

	return hex.EncodeToString(blockInfo.Block.Header.BlockHash)
}

func testGetBlockWithTxRWSetsByHeight(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, height int64) string {
	fmt.Printf("\n============ get block with txRWsets by height [%d] ============\n", height)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHeight",
			Value: strconv.FormatInt(height, 10),
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_BLOCK_WITH_TXRWSETS_BY_HEIGHT", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)

	return hex.EncodeToString(blockInfo.Block.Header.BlockHash)
}

func testGetBlockByHash(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, hash string) {
	fmt.Printf("\n============ get block by hash [%s] ============\n", hash)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: hash,
		},
		{
			Key:   "withRWSet",
			Value: "false",
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_BLOCK_BY_HASH", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetBlockWithTxRWSetsByHash(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, hash string) {
	fmt.Printf("\n============ get block with txRWsets by hash [%s] ============\n", hash)
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "blockHash",
			Value: hash,
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_BLOCK_WITH_TXRWSETS_BY_HASH", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetLastConfigBlock(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Printf("\n============ get last config block ============\n")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "withRWSet",
			Value: "true",
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_LAST_CONFIG_BLOCK", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetLastBlock(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Printf("\n============ get last block ============\n")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "withRWSet",
			Value: "true",
		},
	}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_LAST_BLOCK", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	blockInfo := &commonPb.BlockInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, blockInfo)
	if err != nil {
		fmt.Printf("blockInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, blockInfo)
}

func testGetChainInfo(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	fmt.Printf("\n============ get chain info ============\n")
	// 构造Payload
	pairs := []*commonPb.KeyValuePair{}

	payloadBytes := constructPayload(commonPb.ContractName_SYSTEM_CONTRACT_QUERY.String(), "GET_CHAIN_INFO", pairs)

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_SYSTEM_CONTRACT,
		chainId, "", payloadBytes)

	chainInfo := &discoveryPb.ChainInfo{}
	err := proto.Unmarshal(resp.ContractResult.Result, chainInfo)
	if err != nil {
		fmt.Printf("chainInfo unmarshal error %s\n", err)
		os.Exit(0)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, blockInfo:%+v\n", resp.ContractResult.Code, resp.ContractResult.Message, chainInfo)
}

func testCreate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ create contract [%s] ============\n", txId)

	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson))
	var byteCodeBin []byte
	if runtimeType == commonPb.RuntimeType_EVM {
		byteCodeTmp, err := ioutil.ReadFile(ByteCodePath)
		bc := string(byteCodeTmp)
		checkErr(err)
		data := ""
		hasParam := true
		if hasParam {
			method := ""
			myAbi, err := abi.JSON(strings.NewReader(AbiJson))
			checkErr(err)
			addr := evm.BigToAddress(evm.FromDecimalString(client1Addr))
			dataByte, err := myAbi.Pack(method, addr)
			checkErr(err)
			data = hex.EncodeToString(dataByte)
		}
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}
		fmt.Println(data)
		//bc += data
		byteCodeBin, err = hex.DecodeString(bc)
	} else {
		var err error
		byteCodeBin, err = ioutil.ReadFile(WasmPath)
		checkErr(err)
		pairs = []*commonPb.KeyValuePair{}
	}

	method := commonPb.ManageUserContractFunction_INIT_CONTRACT.String()

	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName:    contractName,
			ContractVersion: "1.0.0",
			RuntimeType:     runtimeType,
		},
		Method:     method,
		Parameters: pairs,
		ByteCode:   byteCodeBin,
	}

	if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("failed to sign endorsement, %s", err.Error())
		os.Exit(0)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
		chainId, txId, payloadBytes)
	//result, _ := myAbi.Constructor.Outputs.Unpack(resp.ContractResult.Result)
	v := make(map[string]interface{})
	if resp.ContractResult != nil {
		myAbi.Constructor.Outputs.UnpackIntoMap(v, resp.ContractResult.Result)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
}

func testUpgrade(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ create contract [%s] ============\n", txId)

	var pairs []*commonPb.KeyValuePair

	var byteCodeBin []byte
	if runtimeType == commonPb.RuntimeType_EVM {
		byteCodeTmp, err := ioutil.ReadFile(ByteCodePath)
		checkErr(err)
		byteCodeBin, err = hex.DecodeString(string(byteCodeTmp))
		checkErr(err)

		data := ""
		hasParam := true
		if hasParam {
			method := ""
			myAbi, err := abi.JSON(strings.NewReader(AbiJson))
			checkErr(err)
			dataByte, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(2))
			checkErr(err)
			data = hex.EncodeToString(dataByte)
		}
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}
	} else {
		var err error
		byteCodeBin, err = ioutil.ReadFile(WasmPath)
		checkErr(err)
		pairs = []*commonPb.KeyValuePair{}
	}

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
		ByteCode:   byteCodeBin,
	}
	if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("failed to sign endorsement, %s", err.Error())
		os.Exit(0)
	}
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
}
func testFreezeOrUnfreezeOrRevoke(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, method string) {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ freeze contract [%s] ============\n", txId)

	payload := &commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName: contractName,
			RuntimeType:  runtimeType,
		},
		Method: method,
	}

	if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
		payload.Endorsement = endorsement
	} else {
		log.Fatalf("failed to sign endorsement, %s", err.Error())
		os.Exit(0)
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
		os.Exit(0)
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_MANAGE_USER_CONTRACT, chainId, txId, payloadBytes)

	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
}

func testInvoke(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson))
	method0 := "toSetData"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		method = method0
		//test1 := evm.StringToAddress("test1")
		//test2:=evm.StringToAddress("test2")
		//test3:=evm.StringToAddress("test3")
		//i:=evm.FromDecimalString("648297579190335911289253806050994198461092955663")
		d1 := evm.BigToAddress(evm.FromDecimalString("520910736052987994931930070646462332401959169580"))
		fmt.Println(d1)
		//dataByte, err := myAbi.Pack(method, big.NewInt(1234))
		dataByte, err := myAbi.Pack(method, d1, big.NewInt(123))
		fmt.Println("dataByte :", dataByte, err)
		data := hex.EncodeToString(dataByte)
		fmt.Println("data :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "save"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}

	fmt.Printf("send tx resp: code:%d, msg:%s, result:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	return txId
}

func testInvoke2(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair

	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		//method = "setAndMul"
		//myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		//checkErr(err)
		//dataByte, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(4))
		//checkErr(err)

		method = "mul"
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		dataByte, err := myAbi.Pack(method)
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "find_by_file_hash"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
		chainId, txId, payloadBytes)

	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

	return txId
}

func testQuery(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson1))
	//test1Addr, _ := myAbi.Pack("",big.NewInt(100000000),"test","test")
	//fmt.Println("test1Addr : ", hex.EncodeToString(test1Addr))
	//method0 := "balanceOfAddress"
	method0 := "balances"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		//method = "setAndMul"
		//myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		//checkErr(err)
		//dataByte, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(4))
		//checkErr(err)

		method = method0
		//test1 := evm.StringToAddress("test1")
		//test2 := evm.StringToAddress("test2")
		//test3:=evm.StringToAddress("test3")
		//test1, _ := evm.MakeAddressFromHex("aaaa1")
		//test2,_ := evm.MakeAddressFromHex("aaaa2")
		//test3,_ := evm.MakeAddressFromHex("aaaa3")
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		addr := evm.BigToAddress(evm.FromDecimalString(client1Addr))
		dataByte, err := myAbi.Pack(method, addr)

		//dataByte, err := myAbi.Pack(method,test1,big.NewInt(99999999999999))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "find_by_file_hash"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
		chainId, txId, payloadBytes)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

}

func testQuery2(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson1))
	//test1Addr, _ := myAbi.Pack("",big.NewInt(100000000),"test","test")
	//fmt.Println("test1Addr : ", hex.EncodeToString(test1Addr))
	//method0 := "balanceOfAddress"
	method0 := "balances"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		//method = "setAndMul"
		//myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		//checkErr(err)
		//dataByte, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(4))
		//checkErr(err)

		method = method0
		//test1 := evm.StringToAddress("test1")
		//test2 := evm.StringToAddress("test2")
		//test3:=evm.StringToAddress("test3")
		//test1, _ := evm.MakeAddressFromHex("aaaa1")
		//test2,_ := evm.MakeAddressFromHex("aaaa2")
		//test3,_ := evm.MakeAddressFromHex("aaaa3")
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		senderSki := "9dbf916da9f5ae892e0991d82b30e1366fe7aa76a6e18767783c9fa3c0921f3b"
		addr, err := evm.MakeAddressFromHex(senderSki)

		checkErr(err)
		dataByte, err := myAbi.Pack(method, evm.BigToAddress(addr))

		//dataByte, err := myAbi.Pack(method,test1,big.NewInt(99999999999999))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "find_by_file_hash"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_USER_CONTRACT,
		chainId, txId, payloadBytes)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

}

func testSet(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson1))
	method0 := "updateBalance"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		//method = "setAndMul"
		//myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		//checkErr(err)
		//dataByte, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(4))
		//checkErr(err)

		method = method0
		//test1 := evm.StringToAddress("test1")
		//test2 := evm.StringToAddress("test2")
		//test3:=evm.StringToAddress("test3")
		//test1, _ := evm.MakeAddressFromHex("aaaa1")
		//test2,_ := evm.MakeAddressFromHex("aaaa2")
		//test3,_ := evm.MakeAddressFromHex("aaaa3")
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		addr := evm.BigToAddress(evm.FromDecimalString(client1Addr))
		dataByte, err := myAbi.Pack(method, big.NewInt(100000002), addr)

		//dataByte, err := myAbi.Pack(method,test1,big.NewInt(99999999999999))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "find_by_file_hash"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

}
func testSet2(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson1))
	method0 := "updateMyBalance"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		//method = "setAndMul"
		//myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		//checkErr(err)
		//dataByte, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(4))
		//checkErr(err)

		method = method0
		//test1 := evm.StringToAddress("test1")
		//test2 := evm.StringToAddress("test2")
		//test3:=evm.StringToAddress("test3")
		//test1, _ := evm.MakeAddressFromHex("aaaa1")
		//test2,_ := evm.MakeAddressFromHex("aaaa2")
		//test3,_ := evm.MakeAddressFromHex("aaaa3")
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		dataByte, err := myAbi.Pack(method, big.NewInt(1000004))

		//dataByte, err := myAbi.Pack(method,test1,big.NewInt(99999999999999))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "find_by_file_hash"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

}
func testTransfer(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson1))
	method0 := "transfer"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		method = method0
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)

		addr := evm.BigToAddress(evm.FromDecimalString(client1Addr))
		dataByte, err := myAbi.Pack(method, addr, big.NewInt(10))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: data,
			},
		}

	} else {
		method = "find_by_file_hash"
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "file_hash",
				Value: "counter1",
			},
			{
				Key:   "file_name",
				Value: "counter1",
			},
		}
	}

	payload := &commonPb.TransactPayload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_USER_CONTRACT,
		chainId, txId, payloadBytes)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

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
		log.Fatalf("marshal payload failed, %s", err.Error())
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

	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, "SHA256")
}

func testUserContractFunctionalFlow(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {

}
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
