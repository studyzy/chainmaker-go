/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package xvm

import (
	"errors"

	"chainmaker.org/chainmaker-go/wxvm/xvm/exec"
	"chainmaker.org/chainmaker-go/wxvm/xvm/runtime/emscripten"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
)

func CreateInstance(contextId int64, code exec.Code, method string, contract *commonPb.Contract, gasUsed uint64,
	gasLimit int64) (*WxvmInstance, error) {
	execCtx, err := code.NewContext(&exec.ContextConfig{
		GasLimit: gasLimit,
	})
	if err != nil {
		return nil, err
	}

	if err = emscripten.Init(execCtx); err != nil {
		return nil, err
	}

	execCtx.SetGasUsed(gasUsed)
	execCtx.SetUserData(contextIDKey, contextId)
	instance := &WxvmInstance{
		method:  method,
		ExecCtx: execCtx,
	}
	return instance, nil
}

type WxvmInstance struct {
	method  string
	ExecCtx exec.Context
}

func (x *WxvmInstance) Exec() error {
	mem := x.ExecCtx.Memory()
	if mem == nil {
		return errors.New("bad contract, no memory")
	}

	function := "_" + x.method
	_, err := x.ExecCtx.Exec(function, []int64{})
	return err
}

func (x *WxvmInstance) ResourceUsed() Limits {
	limits := Limits{
		Cpu: x.ExecCtx.GasUsed(),
	}
	return limits
}

func (x *WxvmInstance) Release() {
	x.ExecCtx.Release()
}

func (x *WxvmInstance) Abort(msg string) {
	exec.Throw(exec.NewTrap(msg))
}
