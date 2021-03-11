package wasi

import (
	"reflect"
	"strconv"

	"chainmaker.org/chainmaker-go/gasm/gasm-go/wasm"
)

const WasiUnstableModuleName = "wasi_unstable"
const WasiModuleName = "wasi_snapshot_preview1"

type WasiInstance struct {
	Modules map[string]*wasm.Module
}

func (w *WasiInstance) FdWrite(vm *wasm.VirtualMachine) reflect.Value {
	body := func(fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
		//if fd != 1 {
		//	panic(fmt.Errorf("invalid file descriptor: %d", fd))
		//}
		//
		//var nwritten uint32
		//for i := int32(0); i < iovsLen; i++ {
		//	iovPtr := iovsPtr + i*8
		//	offset := binary.LittleEndian.Uint32(vm.Memory[iovPtr:])
		//	l := binary.LittleEndian.Uint32(vm.Memory[iovPtr+4:])
		//	n, err := os.Stdout.Write(vm.Memory[offset : offset+l])
		//	if err != nil {
		//		panic(err)
		//	}
		//	nwritten += uint32(n)
		//}
		//binary.LittleEndian.PutUint32(vm.Memory[nwrittenPtr:], nwritten)
		return 0
	}
	return reflect.ValueOf(body)
}

func (w *WasiInstance) FdRead(machine *wasm.VirtualMachine) reflect.Value {
	body := func(fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
		return 0
	}
	return reflect.ValueOf(body)
}

func (w *WasiInstance) FdClose(machine *wasm.VirtualMachine) reflect.Value {
	body := func(fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
		return 0
	}
	return reflect.ValueOf(body)
}

func (w *WasiInstance) FdSeek(machine *wasm.VirtualMachine) reflect.Value {
	body := func(fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
		return 0
	}
	return reflect.ValueOf(body)
}

func (w *WasiInstance) ProcExit(machine *wasm.VirtualMachine) reflect.Value {
	body := func(exitCode int32) {
		panic("exit called by contract, code:" + strconv.Itoa(int(exitCode)))
	}
	return reflect.ValueOf(body)
}
