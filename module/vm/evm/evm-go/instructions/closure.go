/*
 * Copyright 2020 The SealEVM Authors
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

package instructions

import (
	"chainmaker.org/chainmaker-go/common/evmutils"
	"chainmaker.org/chainmaker-go/evm/evm-go/opcodes"
)

type ClosureExecute func(ClosureParam) ([]byte, error)

type ClosureParam struct {
	VM           interface{}
	OpCode       opcodes.OpCode
	GasRemaining *evmutils.Int

	ContractAddress *evmutils.Int
	ContractHash    *evmutils.Int
	ContractCode    []byte

	CallData   []byte
	CallValue  *evmutils.Int
	CreateSalt *evmutils.Int
}

func loadClosure() {
	instructionTable[opcodes.CALL] = opCodeInstruction{
		action:            callAction,
		requireStackDepth: 7,
		enabled:           true,
		returns:           true,
	}

	instructionTable[opcodes.CALLCODE] = opCodeInstruction{
		action:            callCodeAction,
		requireStackDepth: 7,
		enabled:           true,
		returns:           true,
	}

	instructionTable[opcodes.DELEGATECALL] = opCodeInstruction{
		action:            delegateCallAction,
		requireStackDepth: 6,
		enabled:           true,
		returns:           true,
	}

	instructionTable[opcodes.STATICCALL] = opCodeInstruction{
		action:            staticCallAction,
		requireStackDepth: 6,
		enabled:           true,
		returns:           true,
	}

	instructionTable[opcodes.CREATE] = opCodeInstruction{
		action:            createAction,
		requireStackDepth: 3,
		enabled:           true,
		returns:           true,
		isWriter:          true,
	}

	instructionTable[opcodes.CREATE2] = opCodeInstruction{
		action:            create2Action,
		requireStackDepth: 2,
		enabled:           true,
		returns:           true,
		isWriter:          true,
	}
}

func commonCall(ctx *instructionsContext, opCode opcodes.OpCode) ([]byte, error) {
	_ = ctx.stack.Pop()
	addr := ctx.stack.Pop()
	var v *evmutils.Int = nil
	if opCode != opcodes.DELEGATECALL && opCode != opcodes.STATICCALL {
		v = ctx.stack.Pop()
	}
	dOffset := ctx.stack.Pop()
	dLen := ctx.stack.Pop()
	rOffset := ctx.stack.Pop()
	rLen := ctx.stack.Pop()

	//gas check
	offset, size, _, err := ctx.memoryGasCostAndMalloc(dOffset, dLen)
	if err != nil {
		return nil, err
	}

	data, err := ctx.memory.Copy(offset, size)
	//view data 查看data是damo1的入参
	if err != nil {
		return nil, err
	}

	contractCode, err := ctx.storage.GetCode(addr)
	if err != nil {
		return nil, err
	}

	contractCodeHash, err := ctx.storage.GetCodeHash(addr)
	if err != nil {
		return nil, err
	}
	cParam := ClosureParam{
		VM:              ctx.vm,
		OpCode:          opCode,
		GasRemaining:    ctx.gasRemaining,
		ContractAddress: addr,
		ContractCode:    contractCode,
		ContractHash:    contractCodeHash,
		CallData:        data,
		CallValue:       v,
	}

	callRet, err := ctx.closureExec(cParam)
	if err != nil {
		return nil, err
	}

	//gas check
	offset, size, _, err = ctx.memoryGasCostAndMalloc(rOffset, rLen)
	if err != nil {
		return nil, err
	}

	err = ctx.memory.StoreNBytes(offset, size, callRet)
	if err != nil {
		ctx.stack.Push(evmutils.New(0))
	} else {
		ctx.stack.Push(evmutils.New(1))
	}
	return callRet, err
}

func callAction(ctx *instructionsContext) ([]byte, error) {
	return commonCall(ctx, opcodes.CALL)
}

func callCodeAction(ctx *instructionsContext) ([]byte, error) {
	return commonCall(ctx, opcodes.CALLCODE)
}

func delegateCallAction(ctx *instructionsContext) ([]byte, error) {
	return commonCall(ctx, opcodes.DELEGATECALL)
}

func staticCallAction(ctx *instructionsContext) ([]byte, error) {
	return commonCall(ctx, opcodes.STATICCALL)
}

func commonCreate(ctx *instructionsContext, opCode opcodes.OpCode) ([]byte, error) {
	v := ctx.stack.Pop()
	mOffset := ctx.stack.Pop()
	mSize := ctx.stack.Pop()
	//rand.Seed(time.Now().UnixNano())
	var salt *evmutils.Int = evmutils.New(0)

	if opCode == opcodes.CREATE2 {
		salt = salt.Add(ctx.stack.Pop())
	}
	solt0 := evmutils.New(0).SetBytes(ctx.environment.Contract.Code[0:ctx.pc])
	salt.Add(evmutils.FromBigInt(solt0))
	//gas check
	offset, size, _, err := ctx.memoryGasCostAndMalloc(mOffset, mSize)
	if err != nil {
		return nil, err
	}

	code, err := ctx.memory.Copy(offset, size)
	if err != nil {
		return nil, err
	}
	addr := ctx.storage.CreateFixedAddress(ctx.environment.Message.Caller, salt, ctx.environment.Transaction)
	ctx.storage.SetCode(addr, code)
	hash := evmutils.Keccak256(code)
	i := evmutils.New(0)
	i.SetBytes(hash)
	ctx.storage.SetCodeHash(addr, i)
	ctx.storage.SetCodeSize(addr, evmutils.New(int64(len(code))))
	cParam := ClosureParam{
		VM:              ctx.vm,
		OpCode:          opCode,
		GasRemaining:    ctx.gasRemaining,
		ContractAddress: addr,
		ContractCode:    code,
		CallData:        []byte{},
		CallValue:       v,
		CreateSalt:      salt,
	}

	ret, err := ctx.closureExec(cParam)
	if err != nil {
		ctx.stack.Push(evmutils.New(0))
	} else {
		//addr:=ctx.storage.CreateFixedAddress(ctx.environment.Message.Caller,salt,ctx.environment.Transaction)
		ctx.stack.Push(addr)
	}
	return ret, err
}

func createAction(ctx *instructionsContext) ([]byte, error) {
	return commonCreate(ctx, opcodes.CREATE)
}

func create2Action(ctx *instructionsContext) ([]byte, error) {
	return commonCreate(ctx, opcodes.CREATE2)
}
