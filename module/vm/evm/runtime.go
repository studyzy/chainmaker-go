/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package evm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"

	evm_go "chainmaker.org/chainmaker-go/evm/evm-go"
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/opcodes"
	"chainmaker.org/chainmaker-go/evm/evm-go/storage"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker/common/evmutils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	Method        string             // invoke contract method
	ChainId       string             // chain id
	Address       *evmutils.Int      //address
	Contract      *commonPb.Contract // contract info
	Log           *logger.CMLogger
	TxSimContext  protocol.TxSimContext
	ContractEvent []*commonPb.ContractEvent
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string,
	byteCode []byte, parameters map[string][]byte,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().Payload.TxId

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    uint32(1),
		Result:  nil,
		Message: "",
	}

	defer func() {
		if err := recover(); err != nil {
			r.Log.Errorf("failed to invoke evm, tx id:%s, error:%s", txId, err)
			contractResult.Code = 1
			if e, ok := err.(error); ok {
				contractResult.Message = e.Error()
			} else if e, ok := err.(string); ok {
				contractResult.Message = e
			}
			debug.PrintStack()
		}
	}()

	// merge evm param
	//todo sdk常量
	params := string(parameters[protocol.ContractEvmParamKey])
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

	messageData, err := hex.DecodeString(params)
	if err != nil {
		return r.errorResult(contractResult, err, "params is not hex encode string")
	}
	if isDeploy {
		messageData = append(byteCode, messageData...)
		byteCode = messageData
	}

	// evmTransaction
	creatorAddress, err := evmutils.MakeAddressFromHex(string(parameters[protocol.ContractCreatorPkParam]))
	if err != nil {
		return r.errorResult(contractResult, err, "get creator pk fail")
	}
	senderAddress, err := evmutils.MakeAddressFromHex(string(parameters[protocol.ContractSenderPkParam]))
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
	//address, err := evmutils.MakeAddressFromString(contract.Name) // reference vm_factory.go RunContract
	address, err := contractNameHexToAddress(contract.Name)
	if err != nil {
		return r.errorResult(contractResult, err, "make address fail")
	}
	codeHash := evmutils.BytesDataToEVMIntHash(byteCode)
	eContract := environment.Contract{
		Address: address,
		Code:    byteCode,
		Hash:    codeHash,
	}
	r.Address = address
	r.Contract = contract
	// new evm instance
	lastBlock, _ := txSimContext.GetBlockchainStore().GetLastBlock()
	externalStore := &storage.ContractStorage{Ctx: txSimContext}
	evm := evm_go.New(evm_go.EVMParam{
		MaxStackDepth:  protocol.EvmMaxStackDepth,
		ExternalStore:  externalStore,
		ResultCallback: r.callback,
		Context: &environment.Context{
			Block: environment.Block{
				Coinbase:   creatorAddress, //proposer ski
				Timestamp:  evmutils.New(lastBlock.Header.BlockTimestamp),
				Number:     evmutils.New(int64(lastBlock.Header.BlockHeight)), // height
				Difficulty: evmutils.New(0),
				GasLimit:   evmutils.New(protocol.GasLimit),
			},
			Contract:    eContract,
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

	contractResult.Code = 0
	contractResult.GasUsed = gasLeft - result.GasLeft
	contractResult.Result = result.ResultData
	contractResult.ContractEvent = r.ContractEvent
	return contractResult
}

//func contractNameDecimalToAddress(cname string) (*evmutils.Int, error) {
//	// hexStr2 == hexStr2
//	// hexStr := hex.EncodeToString(evmutils.Keccak256([]byte("contractName")))[24:]
//	// hexStr2 := hex.EncodeToString(evmutils.Keccak256([]byte("contractName"))[12:])
//	// 为什么使用十进制字符串转换，因为在./evm-go中，使用的是 address.String()作为key，也就是说数据库的名称是十进制字符串。
//	evmAddr := evmutils.FromDecimalString(cname)
//	if evmAddr == nil {
//		return nil, errors.New("contractName[%s] not DecimalString,
//		you can use evmutils.MakeAddressFromString(\"contractName\").String() get a decimal string")
//	}
//	return evmAddr, nil
//}
func contractNameHexToAddress(cname string) (*evmutils.Int, error) {
	evmAddr := evmutils.FromHexString(cname)
	if evmAddr == nil {
		return nil, errors.New("contractName[%s] not HexString, you can use hex.EncodeToString(" +
			"evmutils.MakeAddressFromString(\"contractName\").Bytes()) get a hex string address")
	}
	return evmAddr, nil
}
func (r *RuntimeInstance) callback(result evm_go.ExecuteResult, err error) {
	if result.ExitOpCode == opcodes.REVERT {
		err = fmt.Errorf("revert instruction was encountered during execution")
		r.Log.Errorf("revert instruction encountered in contract [%s] execution，tx: [%s], error: [%s]",
			r.Contract.Name, r.TxSimContext.GetTx().Payload.TxId, err.Error())
		panic(err)
	}
	if err != nil {
		r.Log.Errorf("error encountered in contract [%s] execution，tx: [%s], error: [%s]",
			r.Contract.Name, r.TxSimContext.GetTx().Payload.TxId, err.Error())
		panic(err)
	}
	//emit  contract event
	err = r.emitContractEvent(result)
	if err != nil {
		r.Log.Errorf("emit contract event err:%s", err.Error())
		panic(err)
	}
	for n, v := range result.StorageCache.CachedData {
		for k, val := range v {
			err := r.TxSimContext.Put(n, []byte(k), val.Bytes())
			if err != nil {
				r.Log.Errorf("callback txSimContext put err:%s", err.Error())
			}
			//fmt.Println("n k val", n, k, val, val.String())
		}
	}
	r.Log.Debug("result:", result.ResultData)
}

func (r *RuntimeInstance) errorResult(
	contractResult *commonPb.ContractResult, err error, errMsg string) *commonPb.ContractResult {
	contractResult.Code = 1
	if err != nil {
		errMsg += ", " + err.Error()
	}
	contractResult.Message = errMsg
	r.Log.Error(errMsg)
	return contractResult
}
func (r *RuntimeInstance) emitContractEvent(result evm_go.ExecuteResult) error {
	//parse log
	var contractEvents []*commonPb.ContractEvent
	logsMap := result.StorageCache.Logs
	for _, logs := range logsMap {
		for _, log := range logs {
			if len(log.Topics) > protocol.EventDataMaxCount-1 {
				return fmt.Errorf("too many event data")
			}
			contractEvent := &commonPb.ContractEvent{
				TxId:            r.TxSimContext.GetTx().Payload.TxId,
				ContractName:    r.Contract.Name,
				ContractVersion: r.Contract.Version,
			}
			topics := log.Topics
			for index, topic := range topics {
				//the first topic in log as contract event topic,others as event data.
				//in ChainMaker contract event,only has one topic filed.
				if index == 0 && topic != nil {
					topicHexStr := hex.EncodeToString(topic)
					if err := protocol.CheckTopicStr(topicHexStr); err != nil {
						return fmt.Errorf(err.Error())
					}
					contractEvent.Topic = topicHexStr
					r.Log.Debugf("topicHexString: %s", topicHexStr)
					continue
				}
				//topic marked by 'index' in ethereum as contract event data
				topicIndexHexStr := hex.EncodeToString(topic)
				r.Log.Debugf("topicIndexString: %s", topicIndexHexStr)
				contractEvent.EventData = append(contractEvent.EventData, topicIndexHexStr)
			}
			data := log.Data
			dataHexStr := hex.EncodeToString(data)
			if len(dataHexStr) > protocol.EventDataMaxLen {
				return fmt.Errorf("event data too long,longer than %v", protocol.EventDataMaxLen)
			}
			contractEvent.EventData = append(contractEvent.EventData, dataHexStr)
			contractEvents = append(contractEvents, contractEvent)
			r.Log.Debugf("dataHexStr: %s", dataHexStr)
		}
	}
	r.ContractEvent = contractEvents
	return nil
}
