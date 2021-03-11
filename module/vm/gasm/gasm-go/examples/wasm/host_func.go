//go:generate tinygo build -opt=s -o host_func.wasm -target wasm host_func.go
package main

import (
	"strconv"
	"unsafe"
)

//export log_message
func logMessage(string)

//export get_state_len
func getStateLen(key string, name string, valueLenPtr uintptr) uint32

//export get_state
func getState(key string, name string, valuePtr uintptr, valueLen int32)

//export put_state
func putState(key string, name string, value string)

//export delete_state
func deleteState(key string, name string)

//export success_result
func successResult(msg string)

//export error_result
func errorResult(msg string)

var args []byte
var argsMap = make(map[string]string)

//export allocate
func allocate(size int32) uintptr {
	args = make([]byte, size)
	return uintptr(unsafe.Pointer(&args[0]))
}

// sdk for user
func GetState(key string, name string) string {
	var valueLen int32
	a := getStateLen(key, name, uintptr(unsafe.Pointer(&valueLen)))
	logMessage(strconv.Itoa(int(a)))
	valueByte := make([]byte, valueLen)
	getState(key, name, uintptr(unsafe.Pointer(&valueByte[0])), valueLen)
	return string(valueByte)
}

//func Arg(key string) string {
//	if argsMap == nil {
//		err := json.Unmarshal(args, &argsMap)
//		if err != nil {
//			logMessage("failed to un marshal args")
//		}
//	}
//	var data Args
//	err := json.Unmarshal([]byte(args), &data)
//	if err != nil {
//		logMessage("arg unmarshal error")
//		return ""
//	}
//
//	value, ok := data.Args[key]
//	if ok {
//		return value
//	} else {
//		logMessage("cannot find arg " + key)
//		return ""
//	}
//}

//export init
func init() {

}

//export call_host_func
func callHostFunc(cnt int32) {
	putState("Key", "Name", "Value")
	value := GetState("Key", "Name")
	logMessage("callHostFunc: " + value)

	putState("Key", "Name", "Value1")
	value = GetState("Key", "Name")
	logMessage("callHostFunc: " + value)

	deleteState("Key", "Name")
	value = GetState("Key", "Name")
	logMessage("callHostFunc: " + value)

	successResult("ok")
	errorResult("no")
}

func main() {}
