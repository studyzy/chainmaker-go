/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package exec

// #include "wxvm.h"
// #include "stdlib.h"
// extern wxvm_resolver_t make_resolver_t(void* env);
// #cgo !windows LDFLAGS: -ldl
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

// Code represents the wasm code object
type aotCode struct {
	code   *C.wxvm_code_t
	bridge *resolverBridge
	// 因为cgo不能持有go的pointer，这个指针是一个指向bridge的token，最后需要Delete
	bridgePointer uintptr
}

// NewAOTCode instances a Code object from file path of native shared library
func NewAOTCode(module string, resolver Resolver) (icode Code, err error) {
	code := &aotCode{}
	code.bridge = newResolverBridge(resolver)
	code.bridgePointer = PointerSave(code.bridge)
	// wxvm_new_code执行期间可能会抛出Trap，导致资源泄露
	// 如果CaptureTrap捕获了Trap则释放所有已经初始化的资源
	defer func() {
		if err != nil {
			code.Release()
			code = nil
		}
	}()
	defer CaptureTrap(&err)

	cpath := C.CString(module)
	defer C.free(unsafe.Pointer(cpath))
	resolvert := C.make_resolver_t(unsafe.Pointer(code.bridgePointer))
	code.code = C.wxvm_new_code(cpath, resolvert)

	if code.code == nil {
		err = fmt.Errorf("open module %s error", module)
		return
	}
	ret := C.wxvm_init_code(code.code)
	if ret == 0 {
		err = fmt.Errorf("init module %s error", module)
		return
	}
	icode = code
	runtime.SetFinalizer(code, (*aotCode).Release)
	return
}

// Release releases resources hold by Code
func (c *aotCode) Release() {
	if c.code != nil {
		C.wxvm_release_code(c.code)
	}
	if c.bridgePointer != 0 {
		PointerDelete(c.bridgePointer)
	}
	*c = aotCode{}
	runtime.SetFinalizer(c, nil)
}
