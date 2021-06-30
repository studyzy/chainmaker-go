/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package evm

import (
	"encoding/hex"
	"fmt"
	"runtime/debug"

	"chainmaker.org/chainmaker/common/evmutils"
	evm_go "chainmaker.org/chainmaker-go/evm/evm-go"
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/opcodes"
	"chainmaker.org/chainmaker-go/evm/evm-go/storage"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	Method        string               // invoke contract method
	ChainId       string               // chain id
	Address       *evmutils.Int        //address
	ContractId    *commonPb.ContractId // contract info
	Log           *logger.CMLogger
	TxSimContext  protocol.TxSimContext
	ContractEvent []*commonPb.ContractEvent
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().GetHeader().TxId

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_FAIL,
		Result:  nil,
		Message: "",
	}

	defer func() {
		if err := recover(); err != nil {
			r.Log.Errorf("failed to invoke evm, tx id:%s, error:%s", txId, err)
			contractResult.Code = commonPb.ContractResultCode_FAIL
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

	messageData, err := hex.DecodeString(params)
	if err != nil {
		return r.errorResult(contractResult, err, "params is not hex encode string")
	}
	if isDeploy {
		messageData = append(byteCode, messageData...)
		byteCode = messageData
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
	r.ContractId = contractId
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
				Number:     evmutils.New(lastBlock.Header.BlockHeight), // height
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

	contractResult.Code = commonPb.ContractResultCode_OK
	contractResult.GasUsed = int64(gasLeft - result.GasLeft)
	contractResult.Result = result.ResultData
	contractResult.ContractEvent = r.ContractEvent
	return contractResult
}

func (r *RuntimeInstance) callback(result evm_go.ExecuteResult, err error) {
	if result.ExitOpCode == opcodes.REVERT {
		err = fmt.Errorf("revert instruction was encountered during execution")
		r.Log.Errorf("revert instruction encountered in contract [%s] execution，tx: [%s], error: [%s]",
			r.ContractId.ContractName, r.TxSimContext.GetTx().Header.TxId, err.Error())
		panic(err)
	}
	if err != nil {
		r.Log.Errorf("error encountered in contract [%s] execution，tx: [%s], error: [%s]",
			r.ContractId.ContractName, r.TxSimContext.GetTx().Header.TxId, err.Error())
		panic(err)
	}
	//emit  contract event
	err = r.emitContractEvent(result)
	if err != nil {
		r.Log.Debugf("emit contract event err:%s", err.Error())
		panic(err)
		return
	}
	for n, v := range result.StorageCache.CachedData {
		for k, val := range v {
			r.TxSimContext.Put(n, []byte(k), val.Bytes())
			//fmt.Println("n k val", n, k, val, val.String())
		}
	}
	if len(result.StorageCache.Destructs) > 0 {
		revokeKey := []byte(protocol.ContractRevoke + r.ContractId.ContractName)
		if err := r.TxSimContext.Put(commonPb.ContractName_SYSTEM_CONTRACT_STATE.String(), revokeKey, []byte(r.ContractId.ContractName)); err != nil {
			panic(err)
		}
		r.Log.Infof("destruction encountered in contract [%s] execution, tx: [%s]",
			r.ContractId.ContractName, r.TxSimContext.GetTx().Header.TxId)
	}
	// save address -> contractName,version
	if r.Method == protocol.ContractInitMethod || r.Method == protocol.ContractUpgradeMethod {
		if err := r.TxSimContext.Put(r.Address.String(), []byte(protocol.ContractAddress), []byte(r.ContractId.ContractName)); err != nil {
			r.Log.Errorf("failed to save contractName %s", err.Error())
			panic(err)
		}
		versionKey := []byte(protocol.ContractVersion + r.Address.String())
		//if err := r.TxSimContext.Put(r.Address.String(), []byte(protocol.ContractVersion), []byte(r.ContractId.ContractVersion)); err != nil {
		if err := r.TxSimContext.Put(commonPb.ContractName_SYSTEM_CONTRACT_STATE.String(), versionKey, []byte(r.ContractId.ContractVersion)); err != nil {
			r.Log.Errorf("failed to save ContractVersion %s", err.Error())
			panic(err)
		}
		// if is create/upgrade contract then override solidity byteCode
		if len(result.ByteCodeBody) > 0 && len(result.ByteCodeHead) > 0 {
			// save byteCodeBody
			versionedByteCodeKey := append([]byte(protocol.ContractByteCode+r.ContractId.ContractName), []byte(r.ContractId.ContractVersion)...)
			//if err := r.TxSimContext.Put(r.ContractId.ContractName, versionedByteCodeKey, result.ByteCodeBody); err != nil {
			if err := r.TxSimContext.Put(commonPb.ContractName_SYSTEM_CONTRACT_STATE.String(), versionedByteCodeKey, result.ByteCodeBody); err != nil {
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

func (r *RuntimeInstance) errorResult(contractResult *commonPb.ContractResult, err error, errMsg string) *commonPb.ContractResult {
	contractResult.Code = commonPb.ContractResultCode_FAIL
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
				TxId:            r.TxSimContext.GetTx().Header.TxId,
				ContractName:    r.ContractId.ContractName,
				ContractVersion: r.ContractId.ContractVersion,
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
