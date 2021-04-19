/*
 * Copyright (c) 2021.  BAEC.ORG.CN All Rights Reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package evm_go

import (
	"chainmaker.org/chainmaker-go/common/evmutils"
	"chainmaker.org/chainmaker-go/evm/evm-go/environment"
	"chainmaker.org/chainmaker-go/evm/evm-go/instructions"
	"chainmaker.org/chainmaker-go/evm/evm-go/memory"
	"chainmaker.org/chainmaker-go/evm/evm-go/opcodes"
	"chainmaker.org/chainmaker-go/evm/evm-go/precompiledContracts"
	"chainmaker.org/chainmaker-go/evm/evm-go/stack"
	"chainmaker.org/chainmaker-go/evm/evm-go/storage"
	"chainmaker.org/chainmaker-go/evm/evm-go/utils"
	"chainmaker.org/chainmaker-go/protocol"
)

type EVMResultCallback func(result ExecuteResult, err error)
type EVMParam struct {
	MaxStackDepth  int
	ExternalStore  storage.IExternalStorage
	ResultCallback EVMResultCallback
	Context        *environment.Context
}

type EVM struct {
	stack        *stack.Stack
	memory       *memory.Memory
	storage      *storage.Storage
	context      *environment.Context
	instructions instructions.IInstructions
	resultNotify EVMResultCallback
}

type ExecuteResult struct {
	ResultData   []byte
	GasLeft      uint64
	StorageCache storage.ResultCache
	ExitOpCode   opcodes.OpCode
	ByteCodeHead []byte
	ByteCodeBody []byte
}

func init() {
	Load()
}
func Load() {
	instructions.Load()
}

func New(param EVMParam) *EVM {
	if param.Context.Block.GasLimit.Cmp(param.Context.Transaction.GasLimit.Int) < 0 {
		param.Context.Transaction.GasLimit = evmutils.FromBigInt(param.Context.Block.GasLimit.Int)
	}

	evm := &EVM{
		stack:        stack.New(param.MaxStackDepth),
		memory:       memory.New(),
		storage:      storage.New(param.ExternalStore),
		context:      param.Context,
		instructions: nil,
		resultNotify: param.ResultCallback,
	}

	evm.instructions = instructions.New(evm, evm.stack, evm.memory, evm.storage, evm.context, nil, closure)

	return evm
}

func (e *EVM) subResult(result ExecuteResult, err error) {
	if err == nil && result.ExitOpCode != opcodes.REVERT {
		storage.MergeResultCache(&result.StorageCache, &e.storage.ResultCache)
	}
}

func (e *EVM) executePreCompiled(addr uint64, input []byte) (ExecuteResult, error) {
	contract := precompiledContracts.Contracts[addr]
	switch addr {
	case 10:
		input = []byte(e.context.Parameters[protocol.ContractSenderOrgIdParam])
		//contract.SetValue(e.context.Parameters[protocol.ContractSenderOrgIdParam])
	case 12:
		input = []byte(e.context.Parameters[protocol.ContractSenderRoleParam])
		//contract.SetValue(e.context.Parameters[protocol.ContractSenderRoleParam])
	case 13:
		input = []byte(e.context.Parameters[protocol.ContractSenderPkParam])
		//contract.SetValue(e.context.Parameters[protocol.ContractSenderPkParam])
	case 14:
		input = []byte(e.context.Parameters[protocol.ContractCreatorOrgIdParam])
		//contract.SetValue(e.context.Parameters[protocol.ContractCreatorOrgIdParam])
	case 15:
		input = []byte(e.context.Parameters[protocol.ContractCreatorRoleParam])
		//contract.SetValue(e.context.Parameters[protocol.ContractCreatorRoleParam])
	case 16:
		input = []byte(e.context.Parameters[protocol.ContractCreatorPkParam])
		//contract.SetValue(e.context.Parameters[protocol.ContractCreatorPkParam])
	}
	gasCost := contract.GasCost(input)
	gasLeft := e.instructions.GetGasLeft()

	result := ExecuteResult{
		ResultData:   nil,
		GasLeft:      gasLeft,
		StorageCache: e.storage.ResultCache,
	}

	if gasLeft < gasCost {
		return result, utils.ErrOutOfGas
	}

	execRet, err := contract.Execute(input)
	gasLeft -= gasCost
	e.instructions.SetGasLimit(gasLeft)
	result.ResultData = execRet
	return result, err
}

func (e *EVM) ExecuteContract(isCreate bool) (ExecuteResult, error) {
	contractAddr := e.context.Contract.Address
	gasLeft := e.instructions.GetGasLeft()

	result := ExecuteResult{
		ResultData:   nil,
		GasLeft:      gasLeft,
		StorageCache: e.storage.ResultCache,
	}
	//return e.executePreCompiled(1, e.context.Message.Data)
	if contractAddr != nil {
		if contractAddr.IsUint64() {
			addr := contractAddr.Uint64()
			if addr < precompiledContracts.ContractsMaxAddress {
				return e.executePreCompiled(addr, e.context.Message.Data)
			}
		}
	}

	execRet, gasLeft, byteCodeHead, byteCodeBody, err := e.instructions.ExecuteContract(isCreate)
	result.ResultData = execRet
	result.GasLeft = gasLeft
	result.ExitOpCode = e.instructions.ExitOpCode()
	result.ByteCodeBody = byteCodeBody
	result.ByteCodeHead = byteCodeHead

	if e.resultNotify != nil {
		e.resultNotify(result, err)
	}
	return result, err
}

func (e *EVM) getClosureDefaultEVM(param instructions.ClosureParam) *EVM {
	newEVM := New(EVMParam{
		MaxStackDepth:  1024,
		ExternalStore:  e.storage.ExternalStorage,
		ResultCallback: e.subResult,
		Context: &environment.Context{
			Block:       e.context.Block,
			Transaction: e.context.Transaction,
			Message: environment.Message{
				Data: param.CallData,
			},
		},
	})

	newEVM.context.Contract = environment.Contract{
		Address: param.ContractAddress,
		Code:    param.ContractCode,
		Hash:    param.ContractHash,
	}

	return newEVM
}

func (e *EVM) commonCall(param instructions.ClosureParam) ([]byte, error) {
	newEVM := e.getClosureDefaultEVM(param)

	//set storage address and call value
	switch param.OpCode {
	case opcodes.CALLCODE:
		newEVM.context.Contract.Address = e.context.Contract.Address
		newEVM.context.Message.Value = param.CallValue
		newEVM.context.Message.Caller = e.context.Contract.Address

	case opcodes.DELEGATECALL:
		newEVM.context.Contract.Address = e.context.Contract.Address
		newEVM.context.Message.Value = e.context.Message.Value
		newEVM.context.Message.Caller = e.context.Message.Caller
	case opcodes.CALL:
		newEVM.context.Contract.Address = param.ContractAddress
		newEVM.context.Message.Value = e.context.Message.Value
		newEVM.context.Message.Caller = e.context.Message.Caller
	}
	if param.OpCode == opcodes.STATICCALL || e.instructions.IsReadOnly() {
		newEVM.instructions.SetReadOnly()
	}

	ret, err := newEVM.ExecuteContract(false)
	//ret, err := newEVM.ExecuteContract(opcodes.CALL == param.OpCode)

	e.instructions.SetGasLimit(ret.GasLeft)
	return ret.ResultData, err
}

func (e *EVM) commonCreate(param instructions.ClosureParam) ([]byte, error) {
	//var addr *utils.Int
	//if opcodes.CREATE == param.OpCode {
	//	addr = e.storage.ExternalStorage.CreateAddress(e.context.Message.Caller, e.context.Transaction)
	//} else {
	//	addr = e.storage.ExternalStorage.CreateFixedAddress(e.context.Message.Caller, param.CreateSalt, e.context.Transaction)
	//}

	newEVM := e.getClosureDefaultEVM(param)

	//newEVM.context.Contract.Address = addr
	newEVM.context.Message.Value = param.CallValue
	newEVM.context.Message.Caller = e.context.Contract.Address

	ret, err := newEVM.ExecuteContract(true)
	e.instructions.SetGasLimit(ret.GasLeft)
	return ret.ResultData, err
}

func closure(param instructions.ClosureParam) ([]byte, error) {
	evm, ok := param.VM.(*EVM)
	if !ok {
		return nil, utils.ErrInvalidEVMInstance
	}

	switch param.OpCode {
	case opcodes.CALL, opcodes.CALLCODE, opcodes.DELEGATECALL, opcodes.STATICCALL:
		return evm.commonCall(param)
	case opcodes.CREATE, opcodes.CREATE2:
		return evm.commonCreate(param)
	}

	return nil, nil
}
