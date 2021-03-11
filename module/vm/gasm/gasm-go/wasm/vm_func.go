package wasm

import (
	"math"
	"reflect"
	"runtime"
)

type (
	VirtualMachineFunction interface {
		Call(vm *VirtualMachine)
		FunctionType() *FunctionType
	}
	HostFunction struct {
		ClosureGenerator func(vm *VirtualMachine) reflect.Value
		function         reflect.Value // should be set at the time of VM creation
		Signature        *FunctionType
	}
	NativeFunction struct {
		Signature *FunctionType
		NumLocal  uint32
		Body      []byte
		Blocks    map[uint64]*NativeFunctionBlock
	}
	NativeFunctionBlock struct {
		HasElse                bool
		StartAt, ElseAt, EndAt uint64
		BlockType              *FunctionType
		BlockTypeBytes         uint64
	}
)

var (
	_ VirtualMachineFunction = &HostFunction{}
	_ VirtualMachineFunction = &NativeFunction{}
)

func (h *HostFunction) FunctionType() *FunctionType {
	return h.Signature
}

func (n *NativeFunction) FunctionType() *FunctionType {
	return n.Signature
}

func (h *HostFunction) Call(vm *VirtualMachine) {
	tp := h.function.Type()
	in := make([]reflect.Value, tp.NumIn())
	for i := len(in) - 1; i >= 0; i-- {
		val := reflect.New(tp.In(i)).Elem()
		raw := vm.OperandStack.Pop()
		kind := tp.In(i).Kind()

		switch kind {
		case reflect.Float64, reflect.Float32:
			val.SetFloat(math.Float64frombits(raw))
		case reflect.Uint32, reflect.Uint64:
			val.SetUint(raw)
		case reflect.Int32, reflect.Int64:
			val.SetInt(int64(raw))
		default:
			panic("invalid input type")
		}
		in[i] = val
	}

	funcName := runtime.FuncForPC(h.function.Pointer()).Name()
	if v, ok := vm.GasMap[funcName]; ok {
		vm.Gas += v
	} else {
		vm.Gas += tableGas[OptCodeCallIndirect]
	}

	for _, ret := range h.function.Call(in) {
		switch ret.Kind() {
		case reflect.Float64, reflect.Float32:
			vm.OperandStack.Push(math.Float64bits(ret.Float()))
		case reflect.Uint32, reflect.Uint64:
			vm.OperandStack.Push(ret.Uint())
		case reflect.Int32, reflect.Int64:
			vm.OperandStack.Push(uint64(ret.Int()))
		default:
			panic("invalid return type")
		}
	}
}

func (n *NativeFunction) Call(vm *VirtualMachine) {
	al := len(n.Signature.InputTypes)
	locals := make([]uint64, n.NumLocal+uint32(al))
	for i := 0; i < al; i++ {
		locals[al-1-i] = vm.OperandStack.Pop()
	}

	prev := vm.ActiveContext
	vm.ActiveContext = &NativeFunctionContext{
		Function:   n,
		Locals:     locals,
		LabelStack: NewVirtualMachineLabelStack(),
	}
	vm.Gas += tableGas[OptCodeCall]
	vm.execNativeFunction()
	vm.ActiveContext = prev
}

func (vm *VirtualMachine) execNativeFunction() {
	for ; int(vm.ActiveContext.PC) < len(vm.ActiveContext.Function.Body); vm.ActiveContext.PC++ {

		switch op := vm.ActiveContext.Function.Body[vm.ActiveContext.PC]; OptCode(op) {
		case OptCodeReturn:
			return
		default:
			virtualMachineInstructions[op](vm)
			vm.AddGas(OptCode(op))
		}
	}
}

func getName(i int) string {
	setName := make(map[int32]string, 200)
	setName[0x00] = "OptCodeUnreachable   "
	setName[0x01] = "OptCodeNop           "
	setName[0x02] = "OptCodeBlock         "
	setName[0x03] = "OptCodeLoop          "
	setName[0x04] = "OptCodeIf            "
	setName[0x05] = "OptCodeElse          "
	setName[0x0b] = "OptCodeEnd           "
	setName[0x0c] = "OptCodeBr            "
	setName[0x0d] = "OptCodeBrIf          "
	setName[0x0e] = "OptCodeBrTable       "
	setName[0x0f] = "OptCodeReturn        "
	setName[0x10] = "OptCodeCall          "
	setName[0x11] = "OptCodeCallIndirect  "

	// parametric instruction
	setName[0x1a] = "OptCodeDrop    "
	setName[0x1b] = "OptCodeSelect  "

	// variable instruction
	setName[0x20] = "OptCodeLocalGet   "
	setName[0x21] = "OptCodeLocalSet   "
	setName[0x22] = "OptCodeLocalTee   "
	setName[0x23] = "OptCodeGlobalGet  "
	setName[0x24] = "OptCodeGlobalSet  "

	// memory instruction
	setName[0x28] = "OptCodeI32Load     "
	setName[0x29] = "OptCodeI64Load     "
	setName[0x2a] = "OptCodeF32Load     "
	setName[0x2b] = "OptCodeF64Load     "
	setName[0x2c] = "OptCodeI32Load8s   "
	setName[0x2d] = "OptCodeI32Load8u   "
	setName[0x2e] = "OptCodeI32Load16s  "
	setName[0x2f] = "OptCodeI32Load16u  "
	setName[0x30] = "OptCodeI64Load8s   "
	setName[0x31] = "OptCodeI64Load8u   "
	setName[0x32] = "OptCodeI64Load16s  "
	setName[0x33] = "OptCodeI64Load16u  "
	setName[0x34] = "OptCodeI64Load32s  "
	setName[0x35] = "OptCodeI64Load32u  "
	setName[0x36] = "OptCodeI32Store    "
	setName[0x37] = "OptCodeI64Store    "
	setName[0x38] = "OptCodeF32Store    "
	setName[0x39] = "OptCodeF64Store    "
	setName[0x3a] = "OptCodeI32Store8   "
	setName[0x3b] = "OptCodeI32Store16  "
	setName[0x3c] = "OptCodeI64Store8   "
	setName[0x3d] = "OptCodeI64Store16  "
	setName[0x3e] = "OptCodeI64Store32  "
	setName[0x3f] = "OptCodeMemorySize  "
	setName[0x40] = "OptCodeMemoryGrow  "

	// numeric instruction
	setName[0x41] = "OptCodeI32Const  "
	setName[0x42] = "OptCodeI64Const  "
	setName[0x43] = "OptCodeF32Const  "
	setName[0x44] = "OptCodeF64Const  "

	setName[0x45] = "OptCodeI32eqz  "
	setName[0x46] = "OptCodeI32eq   "
	setName[0x47] = "OptCodeI32ne   "
	setName[0x48] = "OptCodeI32lts  "
	setName[0x49] = "OptCodeI32ltu  "
	setName[0x4a] = "OptCodeI32gts  "
	setName[0x4b] = "OptCodeI32gtu  "
	setName[0x4c] = "OptCodeI32les  "
	setName[0x4d] = "OptCodeI32leu  "
	setName[0x4e] = "OptCodeI32ges  "
	setName[0x4f] = "OptCodeI32geu  "

	setName[0x50] = "OptCodeI64eqz  "
	setName[0x51] = "OptCodeI64eq   "
	setName[0x52] = "OptCodeI64ne   "
	setName[0x53] = "OptCodeI64lts  "
	setName[0x54] = "OptCodeI64ltu  "
	setName[0x55] = "OptCodeI64gts  "
	setName[0x56] = "OptCodeI64gtu  "
	setName[0x57] = "OptCodeI64les  "
	setName[0x58] = "OptCodeI64leu  "
	setName[0x59] = "OptCodeI64ges  "
	setName[0x5a] = "OptCodeI64geu  "

	setName[0x5b] = "OptCodeF32eq  "
	setName[0x5c] = "OptCodeF32ne  "
	setName[0x5d] = "OptCodeF32lt  "
	setName[0x5e] = "OptCodeF32gt  "
	setName[0x5f] = "OptCodeF32le  "
	setName[0x60] = "OptCodeF32ge  "

	setName[0x61] = "OptCodeF64eq  "
	setName[0x62] = "OptCodeF64ne  "
	setName[0x63] = "OptCodeF64lt  "
	setName[0x64] = "OptCodeF64gt  "
	setName[0x65] = "OptCodeF64le  "
	setName[0x66] = "OptCodeF64ge  "

	setName[0x67] = "OptCodeI32clz     "
	setName[0x68] = "OptCodeI32ctz     "
	setName[0x69] = "OptCodeI32popcnt  "
	setName[0x6a] = "OptCodeI32add     "
	setName[0x6b] = "OptCodeI32sub     "
	setName[0x6c] = "OptCodeI32mul     "
	setName[0x6d] = "OptCodeI32divs    "
	setName[0x6e] = "OptCodeI32divu    "
	setName[0x6f] = "OptCodeI32rems    "
	setName[0x70] = "OptCodeI32remu    "
	setName[0x71] = "OptCodeI32and     "
	setName[0x72] = "OptCodeI32or      "
	setName[0x73] = "OptCodeI32xor     "
	setName[0x74] = "OptCodeI32shl     "
	setName[0x75] = "OptCodeI32shrs    "
	setName[0x76] = "OptCodeI32shru    "
	setName[0x77] = "OptCodeI32rotl    "
	setName[0x78] = "OptCodeI32rotr    "

	setName[0x79] = "OptCodeI64clz     "
	setName[0x7a] = "OptCodeI64ctz     "
	setName[0x7b] = "OptCodeI64popcnt  "
	setName[0x7c] = "OptCodeI64add     "
	setName[0x7d] = "OptCodeI64sub     "
	setName[0x7e] = "OptCodeI64mul     "
	setName[0x7f] = "OptCodeI64divs    "
	setName[0x80] = "OptCodeI64divu    "
	setName[0x81] = "OptCodeI64rems    "
	setName[0x82] = "OptCodeI64remu    "
	setName[0x83] = "OptCodeI64and     "
	setName[0x84] = "OptCodeI64or      "
	setName[0x85] = "OptCodeI64xor     "
	setName[0x86] = "OptCodeI64shl     "
	setName[0x87] = "OptCodeI64shrs    "
	setName[0x88] = "OptCodeI64shru    "
	setName[0x89] = "OptCodeI64rotl    "
	setName[0x8a] = "OptCodeI64rotr    "

	setName[0x8b] = "OptCodeF32abs       "
	setName[0x8c] = "OptCodeF32neg       "
	setName[0x8d] = "OptCodeF32ceil      "
	setName[0x8e] = "OptCodeF32floor     "
	setName[0x8f] = "OptCodeF32trunc     "
	setName[0x90] = "OptCodeF32nearest   "
	setName[0x91] = "OptCodeF32sqrt      "
	setName[0x92] = "OptCodeF32add       "
	setName[0x93] = "OptCodeF32sub       "
	setName[0x94] = "OptCodeF32mul       "
	setName[0x95] = "OptCodeF32div       "
	setName[0x96] = "OptCodeF32min       "
	setName[0x97] = "OptCodeF32max       "
	setName[0x98] = "OptCodeF32copysign  "

	setName[0x99] = "OptCodeF64abs       "
	setName[0x9a] = "OptCodeF64neg       "
	setName[0x9b] = "OptCodeF64ceil      "
	setName[0x9c] = "OptCodeF64floor     "
	setName[0x9d] = "OptCodeF64trunc     "
	setName[0x9e] = "OptCodeF64nearest   "
	setName[0x9f] = "OptCodeF64sqrt      "
	setName[0xa0] = "OptCodeF64add       "
	setName[0xa1] = "OptCodeF64sub       "
	setName[0xa2] = "OptCodeF64mul       "
	setName[0xa3] = "OptCodeF64div       "
	setName[0xa4] = "OptCodeF64min       "
	setName[0xa5] = "OptCodeF64max       "
	setName[0xa6] = "OptCodeF64copysign  "

	setName[0xa7] = "OptCodeI32wrapI64    "
	setName[0xa8] = "OptCodeI32truncf32s  "
	setName[0xa9] = "OptCodeI32truncf32u  "
	setName[0xaa] = "OptCodeI32truncf64s  "
	setName[0xab] = "OptCodeI32truncf64u  "

	setName[0xac] = "OptCodeI64Extendi32s  "
	setName[0xad] = "OptCodeI64Extendi32u  "
	setName[0xae] = "OptCodeI64TruncF32s   "
	setName[0xaf] = "OptCodeI64TruncF32u   "
	setName[0xb0] = "OptCodeI64Truncf64s   "
	setName[0xb1] = "OptCodeI64Truncf64u   "

	setName[0xb2] = "OptCodeF32Converti32s  "
	setName[0xb3] = "OptCodeF32Converti32u  "
	setName[0xb4] = "OptCodeF32Converti64s  "
	setName[0xb5] = "OptCodeF32Converti64u  "
	setName[0xb6] = "OptCodeF32Demotef64    "

	setName[0xb7] = "OptCodeF64Converti32s  "
	setName[0xb8] = "OptCodeF64Converti32u  "
	setName[0xb9] = "OptCodeF64Converti64s  "
	setName[0xba] = "OptCodeF64Converti64u  "
	setName[0xbb] = "OptCodeF64Promotef32   "

	setName[0xbc] = "OptCodeI32reinterpretf32  "
	setName[0xbd] = "OptCodeI64reinterpretf64  "
	setName[0xbe] = "OptCodeF32reinterpreti32  "
	setName[0xbf] = "OptCodeF64reinterpreti64  "

	return setName[int32(i)]
}
