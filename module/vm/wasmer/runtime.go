/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
)

// RuntimeInstance wasm runtime
type RuntimeInstance struct {
	pool    *vmPool
	log     *logger.CMLogger
	chainId string
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {

	logStr := fmt.Sprintf("wasmer runtime invoke[%s]: ", txContext.GetTx().GetHeader().GetTxId())
	startTime := utils.CurrentTimeMillisSeconds()

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_OK,
		Result:  nil,
		Message: "",
	}
	var instanceInfo *wrappedInstance
	defer func() {
		endTime := utils.CurrentTimeMillisSeconds()
		logStr = fmt.Sprintf("%s used time %d", logStr, endTime-startTime)
		r.log.Debugf(logStr)
		panicErr := recover()
		if panicErr != nil {
			contractResult.Code = commonPb.ContractResultCode_FAIL
			contractResult.Message = fmt.Sprint(panicErr)
			if instanceInfo != nil {
				instanceInfo.errCount++
			}
		}
	}()

	// if cross contract call, then new instance
	if txContext.GetDepth() > 0 {
		var err error
		instanceInfo, err = r.pool.NewInstance()
		defer r.pool.CloseInstance(instanceInfo)
		if err != nil {
			panic(err)
		}
	} else {
		instanceInfo = r.pool.GetInstance()
		defer r.pool.RevertInstance(instanceInfo)
	}

	instance := instanceInfo.wasmInstance
	instance.SetGasUsed(gasUsed)
	instance.SetGasLimit(protocol.GasLimit)

	var sc = NewSimContext(method, r.log)
	defer sc.removeCtxPointer()
	sc.ContractId = contractId
	sc.TxSimContext = txContext
	sc.ContractResult = contractResult
	sc.parameters = parameters
	sc.Instance = instance

	err := sc.CallMethod(instance)

	// gas Log
	gas := instance.GetGasUsed()
	if gas > protocol.GasLimit {
		if err != nil {
			err = fmt.Errorf("contract invoke failed, %s, out of gas %d/%d", err.Error(), gas, int64(protocol.GasLimit))
		} else {
			err = fmt.Errorf("contract invoke failed, out of gas %d/%d", gas, int64(protocol.GasLimit))
		}
	}
	logStr += fmt.Sprintf("used gas %d ", gas)
	contractResult.GasUsed = int64(gas)

	if err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		msg := fmt.Sprintf("contract invoke failed, %s", err.Error())
		r.log.Errorf(msg)
		contractResult.Message = msg
		instanceInfo.errCount++
		return contractResult
	}
	contractResult.ContractEvent = sc.ContractEvent
	contractResult.GasUsed = int64(gas)
	return contractResult
}
