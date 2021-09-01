package xvm

import (
	"errors"

	"chainmaker.org/chainmaker-go/wxvm/xvm/exec"
	"chainmaker.org/chainmaker-go/wxvm/xvm/runtime/emscripten"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
)

func CreateInstance(contextId int64, code exec.Code, method string, contract *commonPb.Contract, gasUsed uint64, gasLimit int64) (*wxvmInstance, error) {
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
	instance := &wxvmInstance{
		method:  method,
		ExecCtx: execCtx,
	}
	return instance, nil
}

type wxvmInstance struct {
	method  string
	ExecCtx exec.Context
}

func (x *wxvmInstance) Exec() error {
	mem := x.ExecCtx.Memory()
	if mem == nil {
		return errors.New("bad contract, no memory")
	}

	function := "_" + x.method
	_, err := x.ExecCtx.Exec(function, []int64{})
	return err
}

func (x *wxvmInstance) ResourceUsed() Limits {
	limits := Limits{
		Cpu: x.ExecCtx.GasUsed(),
	}
	return limits
}

func (x *wxvmInstance) Release() {
	x.ExecCtx.Release()
}

func (x *wxvmInstance) Abort(msg string) {
	exec.Throw(exec.NewTrap(msg))
}
