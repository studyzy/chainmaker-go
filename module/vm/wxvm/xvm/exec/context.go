/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package exec

// #include "wxvm.h"
// #include "stdlib.h"
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

type ErrFuncNotFound struct {
	Name string
}

func (e *ErrFuncNotFound) Error() string {
	return fmt.Sprintf("%s not found", e.Name)
}

// ContextConfig configures an execution context
type ContextConfig struct {
	GasLimit int64
}

// Context hold the context data when running a wasm instance
type Context interface {
	Exec(name string, param []int64) (ret int64, err error)
	GasUsed() uint64
	ResetGasUsed()
	SetGasUsed(uint64)
	Memory() []byte
	StaticTop() uint32
	SetUserData(key string, value interface{})
	GetUserData(key string) interface{}
	Release()
}

type Code interface {
	NewContext(cfg *ContextConfig) (ictx Context, err error)
	Release()
}

type aotContext struct {
	context  C.wxvm_context_t
	//gasUsed  uint64
	cfg      ContextConfig
	userData map[string]interface{}
	code     *aotCode
}

// NewContext instances a Context from Code
func (code *aotCode) NewContext(cfg *ContextConfig) (ictx Context, err error) {
	ctx := &aotContext{
		cfg:      *cfg,
		userData: make(map[string]interface{}),
		code:     code,
	}
	defer func() {
		if err != nil {
			ctx.Release()
			ctx = nil
		}
	}()
	defer CaptureTrap(&err)
	ret := C.wxvm_init_context(&ctx.context, code.code, C.uint64_t(cfg.GasLimit))
	if ret == 0 {
		return nil, errors.New("init context error")
	}
	ictx = ctx
	runtime.SetFinalizer(ctx, (*aotContext).Release)
	return ictx, nil
}

// Release releases resources hold by Context
func (c *aotContext) Release() {
	C.wxvm_release_context(&c.context)
	runtime.SetFinalizer(c, nil)
}

func isalpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isalnum(c byte) bool {
	return isalpha(c) || (c >= '0' && c <= '9')
}

// legalizeName makes a name a legail c identifier
func legalizeName(name string) string {
	if len(name) == 0 {
		return "_"
	}
	result := make([]byte, 1, len(name))
	result[0] = name[0]
	if !isalpha(name[0]) {
		result[0] = '_'
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !isalnum(c) {
			c = '_'
		}
		result = append(result, c)
	}
	return string(result)

}

// Exec executes a wasm function by given name and param
func (c *aotContext) Exec(name string, param []int64) (ret int64, err error) {
	defer CaptureTrap(&err)

	exportName := "export_" + legalizeName(name)
	cname := C.CString(exportName)
	defer C.free(unsafe.Pointer(cname))

	var args *C.int64_t
	if len(param) != 0 {
		args = (*C.int64_t)(unsafe.Pointer(&param[0]))
	}
	var cret C.int64_t
	ok := C.wxvm_call(&c.context, cname, args, C.int64_t(len(param)), &cret)
	if ok == 0 {
		return 0, &ErrFuncNotFound{
			Name: name,
		}
	}
	ret = int64(cret)
	return
}

// gasUsed returns the gas used by Exec
func (c *aotContext) GasUsed() uint64 {
	return uint64(C.wxvm_gas_used(&c.context))
}

// ResetGasUsed reset the gas counter
func (c *aotContext) ResetGasUsed() {
	C.wxvm_reset_gas_used(&c.context)
}

// SetGasUsed reset the gas counter
func (c *aotContext) SetGasUsed(gasUsed uint64) {
	C.wxvm_set_gas_used(&c.context, C.uint64_t(gasUsed))
}

// Memory returns the memory of current context, nil will be returned if wasm code has no memory
func (c *aotContext) Memory() []byte {
	if c.context.mem == nil || c.context.mem.size == 0 {
		return nil
	}
	ptr := c.context.mem.data
	n := int(c.context.mem.size)
	return (*[1 << 30]byte)(unsafe.Pointer(ptr))[:n:n]
}

// StaticTop returns the static data's top offset of memory
func (c *aotContext) StaticTop() uint32 {
	return uint32(C.wxvm_mem_static_top(&c.context))
}

// SetUserData store key-value pair to Context which can be retrieved by GetUserData
func (c *aotContext) SetUserData(key string, value interface{}) {
	c.userData[key] = value
}

// GetUserData retrieves user data stored by SetUserData
func (c *aotContext) GetUserData(key string) interface{} {
	return c.userData[key]
}
