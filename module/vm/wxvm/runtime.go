/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wxvm

import (
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/wxvm/xvm"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"runtime/debug"
)

type RuntimeInstance struct {
	ChainId     string
	CodeManager *xvm.CodeManager
	CtxService  *xvm.ContextService
	Log         *logger.CMLogger
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string, byteCode []byte, parameters map[string][]byte,
	txContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {

	tx := txContext.GetTx()

	defer func() {
		if err := recover(); err != nil {
			r.Log.Errorf("invoke wxvm panic, tx id:%s, error:%s", tx.Payload.TxId, err)
			contractResult.Code = 1
			if e, ok := err.(error); ok {
				contractResult.Message = e.Error()
			} else if e, ok := err.(string); ok {
				contractResult.Message = e
			}
			debug.PrintStack()
		}
	}()

	contractResult = &commonPb.ContractResult{
		Code:    uint32(protocol.ContractResultCode_OK),
		Result:  nil,
		Message: "",
	}

	context := r.CtxService.MakeContext(contract, txContext, contractResult, parameters)
	execCode, err := r.CodeManager.GetExecCode(r.ChainId, contract, byteCode, r.CtxService)
	defer r.CtxService.DestroyContext(context)

	if err != nil {
		contractResult.Code = 1
		contractResult.Message = err.Error()
		return
	}

	if inst, err := xvm.CreateInstance(context.ID, execCode, method, contract, gasUsed, int64(protocol.GasLimit)); err != nil {
		contractResult.Code = 1
		contractResult.Message = err.Error()
		return
	} else if err = inst.Exec(); err != nil {
		contractResult.Code = 1
		contractResult.Message = err.Error()
		return
	} else {
		contractResult.GasUsed = inst.ExecCtx.GasUsed()
		contractResult.ContractEvent = context.ContractEvent
	}

	return
}
