package main

import (
	"os"

	"chainmaker.org/chainmaker-go/gasm"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/vm/test"
)

func main() {
	runtimeInstance := &gasm.RuntimeInstance{
		ChainId: "chain01",
		Log:     logger.GetLogger(logger.MODULE_VM),
	}

	logger := logger.GetLoggerByChain(logger.MODULE_VM, "chain01")
	wasmFilePath := os.Args[1]
	test.WasmFile = wasmFilePath
	contractId, txContext, byteCode := test.InitContextTest(common.RuntimeType_GASM)
	method := os.Args[2]
	parameters := make(map[string]string)
	for i := 3; i < len(os.Args); i += 2 {
		parameters[os.Args[i]] = os.Args[i+1]
	}
	contractResult := runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	logger.Infof("contractResult :%+v\n", contractResult)
}
