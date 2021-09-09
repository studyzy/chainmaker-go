/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"

	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

// RuntimeInstance wasm runtime
type RuntimeInstance struct {
	pool    *vmPool
	log     *logger.CMLogger
	chainId string
}

func (r *RuntimeInstance) Pool() *vmPool {
	return r.pool
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(
	contract *commonPb.Contract, method string,
	byteCode []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {

	logStr := fmt.Sprintf("wasmer runtime invoke[%s]: ", txContext.GetTx().Payload.TxId)
	startTime := utils.CurrentTimeMillisSeconds()

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    uint32(0),
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
			contractResult.Code = 1
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

	var sc = NewSimContext(method, r.log, r.chainId)
	defer sc.removeCtxPointer()
	sc.Contract = contract
	sc.TxSimContext = txContext
	sc.ContractResult = contractResult
	sc.parameters = parameters
	sc.Instance = instance

	err := sc.CallMethod(instance)
	if err != nil {
		r.log.Warnf("contract[%s] invoke failed, %s", contract.Name, err.Error())
	}

	// gas Log
	gas := instance.GetGasUsed()
	if gas > protocol.GasLimit {
		err = fmt.Errorf("out of gas %d/%d", gas, int64(protocol.GasLimit))
	}
	logStr += fmt.Sprintf("used gas %d ", gas)
	contractResult.GasUsed = gas
	if err != nil {
		contractResult.Code = 1
		msg := fmt.Sprintf("contract[%s] invoke failed, %s", contract.Name, err.Error())
		r.log.Errorf(msg)
		contractResult.Message = msg
		instanceInfo.errCount++
		return contractResult
	}
	contractResult.ContractEvent = sc.ContractEvent
	contractResult.GasUsed = gas
	return contractResult
}
