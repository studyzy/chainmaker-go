/*
 * Copyright 2020 ChainMaker.org
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package evm

import (
	"chainmaker.org/chainmaker-go/common/evmutils"
	"chainmaker.org/chainmaker-go/evm/evm-go"
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/opcodes"
	"chainmaker.org/chainmaker-go/evm/evm-go/storage"
	"chainmaker.org/chainmaker-go/logger"
	pb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"
	"fmt"
)

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	Method       string         // invoke contract method
	ChainId      string         // chain id
	Address      *evmutils.Int  //address
	ContractId   *pb.ContractId // contract info
	Log          *logger.CMLogger
	TxSimContext protocol.TxSimContext
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contractId *pb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *pb.ContractResult) {
	txId := txSimContext.GetTx().GetHeader().GetTxId()

	logStr := fmt.Sprintf("evm runtime invoke[%s]:", txId)
	startTime := utils.CurrentTimeMillisSeconds()

	// contract response
	contractResult = &pb.ContractResult{
		Code:    pb.ContractResultCode_FAIL,
		Result:  nil,
		Message: "",
	}

	// record log add panic
	defer func() {
		endTime := utils.CurrentTimeMillisSeconds()
		logStr = fmt.Sprintf("%s, used time %d", logStr, endTime-startTime)
		r.Log.Debugf(logStr)
		panicErr := recover()
		if panicErr != nil {
			r.errorResult(contractResult, nil, fmt.Sprint("panicErr:", panicErr))
		}
	}()

	// merge evm param
	//todo sdk常量
	params := parameters[protocol.ContractEvmParamKey]
	isDeploy := false
	if method == protocol.ContractInitMethod || method == protocol.ContractUpgradeMethod {
		isDeploy = true
	} else {
		if evmutils.Has0xPrefix(method) {
			method = method[2:]
		}
		if len(method) != 8 {
			return r.errorResult(contractResult, nil, "contract verify failed, method length is not 8")
		}
	}
	if evmutils.Has0xPrefix(params) {
		params = params[2:]
	}
	if len(params)%2 == 1 {
		params = "0" + params
	}

	// evmTransaction
	creatorAddress, err := evmutils.MakeAddressFromHex(parameters[protocol.ContractCreatorPkParam])
	if err != nil {
		return r.errorResult(contractResult, err, "get creator pk fail")
	}
	senderAddress, err := evmutils.MakeAddressFromHex(parameters[protocol.ContractSenderPkParam])
	if err != nil {
		return r.errorResult(contractResult, err, "get sender pk fail")
	}

	gasLeft := protocol.GasLimit - gasUsed
	evmTransaction := environment.Transaction{
		TxHash:   []byte(txId),
		Origin:   senderAddress,
		GasPrice: evmutils.New(protocol.EvmGasPrice),
		GasLimit: evmutils.New(int64(gasLeft)),
	}

	// contract
	address, err := evmutils.MakeAddressFromString(contractId.ContractName) // reference vm_factory.go RunContract
	logStr = logStr + " address->" + address.String() + " name ->" + contractId.ContractName

	if err != nil {
		return r.errorResult(contractResult, err, "make address fail")
	}
	codeHash := evmutils.BytesDataToEVMIntHash(byteCode)
	contract := environment.Contract{
		Address: address,
		Code:    byteCode,
		Hash:    codeHash,
	}
	r.Address = address

	messageData, err := hex.DecodeString(params)
	if err != nil {
		return r.errorResult(contractResult, err, "params is not hex encode string")
	}
	if isDeploy {
		messageData = append(byteCode, messageData...)
		byteCode = messageData
	}
	// new evm instance
	externalStore := &storage.ContractStorage{Ctx: txSimContext}
	evm := evm_go.New(evm_go.EVMParam{
		MaxStackDepth:  protocol.EvmMaxStackDepth,
		ExternalStore:  externalStore,
		ResultCallback: r.callback,
		Context: &environment.Context{
			Block: environment.Block{
				Coinbase:   creatorAddress, //proposer ski
				Timestamp:  evmutils.New(startTime),
				Number:     evmutils.New(txSimContext.GetBlockHeight()), // height
				Difficulty: evmutils.New(0),
				GasLimit:   evmutils.New(protocol.GasLimit),
			},
			Contract:    contract,
			Transaction: evmTransaction,
			Message: environment.Message{
				Caller: senderAddress,
				Value:  evmutils.New(0),
				Data:   messageData,
			},
			Parameters: parameters,
		},
	})
	// init memory and env
	evm_go.Load()
	// execute method
	result, err := evm.ExecuteContract(isDeploy)
	if err != nil {
		return r.errorResult(contractResult, err, "failed to execute evm contract")
	}

	contractResult.Code = pb.ContractResultCode_OK
	contractResult.GasUsed = int64(gasLeft - result.GasLeft)
	contractResult.Result = result.ResultData
	return contractResult
}

func (r *RuntimeInstance) callback(result evm_go.ExecuteResult, err error) {
	if result.ExitOpCode == opcodes.REVERT {
		err = fmt.Errorf("revert instruction was encountered during execution")
		r.Log.Errorf("failed to run evm []: %s", r.TxSimContext.GetTx().Header.TxId, err.Error())
		panic(err)
		return
	}
	if err != nil {
		r.Log.Errorf("call back do nothing for err: %s", err.Error())
		panic(err)
		return
	}
	for n, v := range result.StorageCache.CachedData {
		for k, val := range v {
			r.TxSimContext.Put(n, []byte(k), val.Bytes())
		}
	}
	// save address -> contractName,version
	if r.Method == protocol.ContractInitMethod || r.Method == protocol.ContractUpgradeMethod {
		if err := r.TxSimContext.Put(r.Address.String(), []byte(protocol.ContractAddress), []byte(r.ContractId.ContractName)); err != nil {
			r.Log.Errorf("failed to save contractName %s", err.Error())
			panic(err)
		}
		if err := r.TxSimContext.Put(r.Address.String(), []byte(protocol.ContractVersion), []byte(r.ContractId.ContractVersion)); err != nil {
			r.Log.Errorf("failed to save ContractVersion %s", err.Error())
			panic(err)
		}
		// if is create/upgrade contract then override solidity byteCode
		if len(result.ByteCodeBody) > 0 && len(result.ByteCodeHead) > 0 {
			// save byteCodeBody
			versionedByteCodeKey := append([]byte(protocol.ContractByteCode), []byte(r.ContractId.ContractVersion)...)
			if err := r.TxSimContext.Put(r.ContractId.ContractName, versionedByteCodeKey, result.ByteCodeBody); err != nil {
				r.Log.Errorf("failed to save byte code body %s", err.Error())
				panic(err)
			}
		} else {
			r.Log.Errorf("failed to parse evm byte code, head length = %d, body length = %d", len(result.ByteCodeHead), len(result.ByteCodeBody))
			panic(err)
		}
	}

	r.Log.Debug("result:", result.ResultData)
}

func (r *RuntimeInstance) errorResult(contractResult *pb.ContractResult, err error, errMsg string) *pb.ContractResult {
	contractResult.Code = pb.ContractResultCode_FAIL
	if err != nil {
		errMsg += ", " + err.Error()
	}
	contractResult.Message = errMsg
	r.Log.Error(errMsg)
	return contractResult
}
