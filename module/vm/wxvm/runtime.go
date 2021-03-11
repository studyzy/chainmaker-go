/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wxvm

import (
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/wxvm/xvm"
	"runtime/debug"
)

type RuntimeInstance struct {
	ChainId     string
	CodeManager *xvm.CodeManager
	CtxService  *xvm.ContextService
	Log         *logger.CMLogger
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {

	tx := txContext.GetTx()

	defer func() {
		if err := recover(); err != nil {
			r.Log.Errorf("invoke wxvm panic, tx id:%s, error:%s", tx.Header.TxId, err)
			contractResult.Code = commonPb.ContractResultCode_FAIL
			if e, ok := err.(error); ok {
				contractResult.Message = e.Error()
			} else if e, ok := err.(string); ok {
				contractResult.Message = e
			}
			debug.PrintStack()
		}
	}()

	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_OK,
		Result:  nil,
		Message: "",
	}

	execCode, err := r.CodeManager.GetExecCode(r.ChainId, contractId, byteCode, r.CtxService)
	if err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		return
	}

	//var contextId int64 = 0
	context := r.CtxService.MakeContext(contractId, txContext, contractResult, parameters)
	defer r.CtxService.DestroyContext(context)

	if inst, err := xvm.CreateInstance(context.ID, execCode, method, contractId, gasUsed, int64(protocol.GasLimit)); err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		return
	} else if err = inst.Exec(); err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		return
	}
	return
}
