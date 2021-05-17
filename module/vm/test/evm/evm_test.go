/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package evm

import (
	evm_go "chainmaker.org/chainmaker-go/evm/evm-go"
	pb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/common/evmutils"
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/storage"
	"chainmaker.org/chainmaker-go/vm/test"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func Test_main(t *testing.T) {
	//install()
	invoke()
}

var blockGasLimit = evmutils.New(1e10)

func constTransactionGasPrice() *evmutils.Int {
	return evmutils.New(1)
}
func constTransactionGasLimit() *evmutils.Int {
	return evmutils.New(100000000)
}

func contractAddress() *evmutils.Int {
	address := evmutils.Keccak256([]byte("contract1"))
	addr := hex.EncodeToString(address)[24:]
	fmt.Println("contract addr1 =", addr)
	return evmutils.FromHexString(addr)
}

func contractHash() *evmutils.Int {
	return evmutils.New(100000000)
}

func accountAddress1() *evmutils.Int {
	// fc12ad814631ba689f7abe671016f75c54c607f082ae6b0881fac0abeda21781
	// 1016f75c54c607f082ae6b0881fac0abeda21781
	pubKeyByte, _ := hex.DecodeString(pubKey1)
	address := evmutils.Keccak256(pubKeyByte)
	addr := hex.EncodeToString(address)[24:]
	fmt.Println("addr1 =", addr)
	return evmutils.FromHexString(addr)
}

func accountAddress2() *evmutils.Int {
	pubKeyByte, _ := hex.DecodeString(pubKey2)
	address := evmutils.Keccak256(pubKeyByte)
	addr := hex.EncodeToString(address)[24:]
	fmt.Println("addr2 =", addr)
	return evmutils.FromHexString(addr)
}

//topic0=hash("Transfer(address,uint256,uint256);")
//topic1=index0
//topic2=index1
func callback(result evm_go.ExecuteResult, err error) {
	fmt.Println("callback err info:", err)
	fmt.Println("callback result info ResultData:", result.ResultData)
	fmt.Println("callback result info ResultData:", hex.EncodeToString(result.ResultData))
	fmt.Println("callback result info GasLeft:", result.GasLeft)
	fmt.Println("callback result info StorageCache:", result.StorageCache)
	fmt.Println("callback result info ExitOpCode:", result.ExitOpCode)
	fmt.Println("callback result info ByteCodeHead:", result.ByteCodeHead)
	fmt.Println("callback result info ByteCodeBody:", result.ByteCodeBody)

}

func invoke() {
	method := "sum"
	myAbi, _ := abi.JSON(strings.NewReader(abiJSON))
	if _, ok := myAbi.Methods[method]; !ok {
		panic("expected 'balance' to be present")
	}

	//param := make([]string,0)
	//param = append(param, "3")
	//param = append(param, "2")
	//
	//params := make(map[string]string,0)
	//params["_value2"] = "2"
	//params["_value1"] = "3"

	data, err := myAbi.Pack(method, big.NewInt(3), big.NewInt(2))
	fmt.Println("err:", err)
	fmt.Println("callback result info data:", data)
	fmt.Println("callback result info data:", hex.EncodeToString(data))

	evmTransaction := environment.Transaction{
		TxHash:   []byte("0x1"),
		Origin:   evmutils.New(0), // creator address
		GasPrice: constTransactionGasPrice(),
		GasLimit: constTransactionGasLimit(),
	}

	code, err := hex.DecodeString(byteCodeInstall)

	if err != nil {
		panic(err)
	}
	contract := environment.Contract{
		Address: contractAddress(),
		Code:    code,
		Hash:    contractHash(),
	}

	//data, _ := hex.DecodeString(methodSum)

	_, txContext, _ := test.InitContextTest(pb.RuntimeType_EVM)
	evm := evm_go.New(evm_go.EVMParam{
		MaxStackDepth:  1024,
		ExternalStore:  &storage.ContractStorage{Ctx: txContext, Code: code},
		ResultCallback: callback,
		Context: &environment.Context{
			Block: environment.Block{
				Coinbase:   evmutils.BytesDataToEVMIntHash([]byte("0x1")), //proposer ski
				Timestamp:  evmutils.New(0),
				Number:     evmutils.New(0), // height
				Difficulty: evmutils.New(0),
				GasLimit:   blockGasLimit,
			},
			Contract:    contract,
			Transaction: evmTransaction,
			Message: environment.Message{
				Caller: accountAddress1(),
				Value:  evmutils.New(0),
				Data:   data,
				//Data:   nil,
			},
		},
	})

	//instructions.Load()

	evm.ExecuteContract(true)
}

func install() {
	evmTransaction := environment.Transaction{
		TxHash:   []byte("0x1"),
		Origin:   evmutils.New(0),
		GasPrice: constTransactionGasPrice(),
		GasLimit: constTransactionGasLimit(),
	}

	//code, err := hex.DecodeString("6080604052600436106100d0576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806306fdde03146100d557806307da68f514610165578063095ea7b31461017c57806318160ddd146101e157806323b872dd1461020c578063313ce5671461029157806342966c68146102bc57806370a08231146102e957806375f12b211461034057806395d89b411461036f578063a9059cbb146103ff578063be9a655514610464578063c47f00271461047b578063dd62ed3e146104e4575b600080fd5b3480156100e157600080fd5b506100ea61055b565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561012a57808201518184015260208101905061010f565b50505050905090810190601f1680156101575780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561017157600080fd5b5061017a6105f9565b005b34801561018857600080fd5b506101c7600480360381019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019092919050505061066f565b604051808215151515815260200191505060405180910390f35b3480156101ed57600080fd5b506101f6610833565b6040518082815260200191505060405180910390f35b34801561021857600080fd5b50610277600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610839565b604051808215151515815260200191505060405180910390f35b34801561029d57600080fd5b506102a6610b73565b6040518082815260200191505060405180910390f35b3480156102c857600080fd5b506102e760048036038101908080359060200190929190505050610b79565b005b3480156102f557600080fd5b5061032a600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610c9e565b6040518082815260200191505060405180910390f35b34801561034c57600080fd5b50610355610cb6565b604051808215151515815260200191505060405180910390f35b34801561037b57600080fd5b50610384610cc9565b6040518080602001828103825283818151815260200191508051906020019080838360005b838110156103c45780820151818401526020810190506103a9565b50505050905090810190601f1680156103f15780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561040b57600080fd5b5061044a600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610d67565b604051808215151515815260200191505060405180910390f35b34801561047057600080fd5b50610479610f8b565b005b34801561048757600080fd5b506104e2600480360381019080803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290505050611001565b005b3480156104f057600080fd5b50610545600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050611074565b6040518082815260200191505060405180910390f35b60008054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156105f15780601f106105c6576101008083540402835291602001916105f1565b820191906000526020600020905b8154815290600101906020018083116105d457829003601f168201915b505050505081565b3373ffffffffffffffffffffffffffffffffffffffff16600660019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561065257fe5b6001600660006101000a81548160ff021916908315150217905550565b6000600660009054906101000a900460ff1615151561068a57fe5b3373ffffffffffffffffffffffffffffffffffffffff166000141515156106ad57fe5b600082148061073857506000600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054145b151561074357600080fd5b81600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b60055481565b6000600660009054906101000a900460ff1615151561085457fe5b3373ffffffffffffffffffffffffffffffffffffffff1660001415151561087757fe5b81600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156108c557600080fd5b600360008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205482600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054011015151561095457600080fd5b81600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054101515156109df57600080fd5b81600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254019250508190555081600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555081600460008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825403925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3600190509392505050565b60025481565b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515610bc757600080fd5b80600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555080600360008073ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254019250508190555060003373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a350565b60036020528060005260406000206000915090505481565b600660009054906101000a900460ff1681565b60018054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610d5f5780601f10610d3457610100808354040283529160200191610d5f565b820191906000526020600020905b815481529060010190602001808311610d4257829003601f168201915b505050505081565b6000600660009054906101000a900460ff16151515610d8257fe5b3373ffffffffffffffffffffffffffffffffffffffff16600014151515610da557fe5b81600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205410151515610df357600080fd5b600360008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205482600360008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020540110151515610e8257600080fd5b81600360003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000828254039250508190555081600360008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a36001905092915050565b3373ffffffffffffffffffffffffffffffffffffffff16600660019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141515610fe457fe5b6000600660006101000a81548160ff021916908315150217905550565b3373ffffffffffffffffffffffffffffffffffffffff16600660019054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614151561105a57fe5b8060009080519060200190611070929190611099565b5050565b6004602052816000526040600020602052806000526040600020600091509150505481565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106110da57805160ff1916838001178555611108565b82800160010185558215611108579182015b828111156111075782518255916020019190600101906110ec565b5b5090506111159190611119565b5090565b61113b91905b8082111561113757600081600090555060010161111f565b5090565b905600a165627a7a72305820608f90a4c63380bbcdb2f422d2d0d99156d0d9d67a8e0f7feb7a3b14ba4749210029")
	code, err := hex.DecodeString("608060405234801561001057600080fd5b506101d4806100206000396000f30060806040526004361061006d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631ab06ee5146100725780634f2be91f146100a957806358931c46146100d4578063c54124be146100ff578063f9fa48c31461012a575b600080fd5b34801561007e57600080fd5b506100a76004803603810190808035906020019092919080359060200190929190505050610155565b005b3480156100b557600080fd5b506100be610167565b6040518082815260200191505060405180910390f35b3480156100e057600080fd5b506100e9610175565b6040518082815260200191505060405180910390f35b34801561010b57600080fd5b50610114610183565b6040518082815260200191505060405180910390f35b34801561013657600080fd5b5061013f610191565b6040518082815260200191505060405180910390f35b81600081905550806001819055505050565b600060015460005401905090565b600060015460005402905090565b600060015460005403905090565b60006001546000548115156101a257fe5b049050905600a165627a7a7230582046a0c736833d99bd1fa535aaac053604ff7e4f92d91b8b472bac6562c7e90ca70029")
	//code, err := hex.DecodeString(byteCodeAll)

	if err != nil {
		panic(err)
	}
	contract := environment.Contract{
		Address: contractAddress(),
		Code:    code,
		Hash:    contractHash(),
	}

	//data, _ := hex.DecodeString("be9a6555")
	//data, _ := hex.DecodeString("42966c68000000000000000000000000000000000000000000000000000000000000000")

	_, txContext, _ := test.InitContextTest(pb.RuntimeType_EVM)
	evm := evm_go.New(evm_go.EVMParam{
		MaxStackDepth:  1024,
		ExternalStore:  &storage.ContractStorage{Ctx: txContext, Code: code},
		ResultCallback: callback,
		Context: &environment.Context{
			Block: environment.Block{
				Coinbase:   evmutils.BytesDataToEVMIntHash([]byte("0x1")),
				Timestamp:  evmutils.New(0),
				Number:     evmutils.New(0),
				Difficulty: evmutils.New(0),
				GasLimit:   blockGasLimit,
			},
			Contract:    contract,
			Transaction: evmTransaction,
			Message: environment.Message{
				Caller: accountAddress1(),
				Value:  evmutils.New(0),
				//Data:   data,
				Data: nil,
			},
		},
	})

	//instructions.Load()

	evm.ExecuteContract(true)

}
