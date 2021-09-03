package evm_go

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/storage"
	"chainmaker.org/chainmaker-go/evm/test"
	"chainmaker.org/chainmaker/common/v2/evmutils"
	pb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	blockGasLimit = evmutils.New(1e10)
	abiJSON       = "[ { \"inputs\": [], \"name\": \"retrieve\", \"outputs\": [ { \"internalType\": \"uint256\", \"name\": \"\", \"type\": \"uint256\" } ], \"stateMutability\": \"view\", \"type\": \"function\" }, { \"inputs\": [ { \"internalType\": \"uint256\", \"name\": \"num\", \"type\": \"uint256\" } ], \"name\": \"store\", \"outputs\": [], \"stateMutability\": \"nonpayable\", \"type\": \"function\" } ]"
	contractName  = "contract1"
	userSki       = "08E6253A8BF02BBBED033471B424D0A5F1C402CCB6446E4230AB51763A34C37B"
)

func TestEVM_ExecuteContract(t *testing.T) {
	test.CertFilePath = "../test/config/admin1.sing.crt"
	test.ByteCodeFile = "../test/contracts/contract02/storage.bin"
	method := "store"
	myAbi, _ := abi.JSON(strings.NewReader(abiJSON))
	if _, ok := myAbi.Methods[method]; !ok {
		panic("expected 'balance' to be present")
	}

	data, err := myAbi.Pack(method, big.NewInt(100))
	fmt.Println("err:", err)
	fmt.Println("callback result info data:", data)
	fmt.Println("callback result info data:", hex.EncodeToString(data))

	evmTransaction := environment.Transaction{
		TxHash:   []byte("0x1"),
		Origin:   userAddress(userSki), // creator address
		GasPrice: constTransactionGasPrice(),
		GasLimit: constTransactionGasLimit(),
	}
	_, txContext, code := test.InitContextTest(pb.RuntimeType_EVM)

	code, err = hex.DecodeString(string(code))

	if err != nil {
		panic(err)
	}
	contract := environment.Contract{
		Address: contractAddress(contractName),
		Code:    code,
		Hash:    contractHash(code),
	}

	//data, _ := hex.DecodeString(methodSum)

	evm := New(EVMParam{
		MaxStackDepth:  1024,
		ExternalStore:  &storage.ContractStorage{Ctx: txContext},
		ResultCallback: callback,
		Context: &environment.Context{
			Block: environment.Block{
				Coinbase:   userAddress(userSki), //proposer ski
				Timestamp:  evmutils.New(0),
				Number:     evmutils.New(0), // height
				Difficulty: evmutils.New(0),
				GasLimit:   blockGasLimit,
			},
			Contract:    contract,
			Transaction: evmTransaction,
			Message: environment.Message{
				Caller: userAddress(userSki),
				Value:  evmutils.New(0),
				Data:   data,
			},
		},
	})

	//instructions.Load()

	evm.ExecuteContract(true)

}

func constTransactionGasPrice() *evmutils.Int {
	return evmutils.New(1)
}
func constTransactionGasLimit() *evmutils.Int {
	return evmutils.New(100000000)
}

func contractAddress(contractName string) *evmutils.Int {
	address := evmutils.Keccak256([]byte(contractName))
	addr := hex.EncodeToString(address)[24:]
	fmt.Println("contract addr1 =", addr)
	return evmutils.FromHexString(addr)
}
func userAddress(ski string) *evmutils.Int {
	fromHex, _ := evmutils.MakeAddressFromHex(ski)
	return fromHex
}
func contractHash(code []byte) *evmutils.Int {
	return evmutils.BytesDataToEVMIntHash(code)
}

func callback(result ExecuteResult, err error) {
	fmt.Println("callback err info:", err)
	fmt.Println("callback result info ResultData:", result.ResultData)
	fmt.Println("callback result info ResultData:", hex.EncodeToString(result.ResultData))
	fmt.Println("callback result info GasLeft:", result.GasLeft)
	fmt.Println("callback result info StorageCache:", result.StorageCache)
	fmt.Println("callback result info ExitOpCode:", result.ExitOpCode)
	fmt.Println("callback result info ByteCodeHead:", result.ByteCodeHead)
	fmt.Println("callback result info ByteCodeBody:", result.ByteCodeBody)

}
