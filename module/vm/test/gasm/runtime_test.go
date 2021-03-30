package gasmtest

import (
	"chainmaker.org/chainmaker-go/gasm"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestContract_Fact(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/go-func-verify-1.0.0.wasm"
	contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_GASM)

	if len(byteCode) == 0 {
		panic("error byteCode==0")
	}
	start := time.Now().UnixNano() / 1e6
	x := 0
	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		for j := 0; j < 1; j++ {
			x += 1
			y := int32(x)
			wg.Add(1)
			go func() {
				defer wg.Done()
				invokeCallContractTestSave("functional_verify", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("save", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("find_by_file_hash", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("increase", int32(i), contractId, txContext, byteCode)
				invokeCallContractTestSave("query", int32(i), contractId, txContext, byteCode)
				end := time.Now().UnixNano() / 1e6
				if (end-start)/1000 > 0 && y%100 == 0 {
					fmt.Printf("【tps】 %d 【spend】%d i = %d, count=%d \n", int(y)/int((end-start)/1000), end-start, i+1, y)
				}
			}()
		}
		wg.Wait()
	}

	end := time.Now().UnixNano() / 1e6
	println("end 【spend】", end-start)
}

func invokeCallContractTestSave(method string, id int32, contractId *commonPb.ContractId, txContext protocol.TxSimContext, byteCode []byte) {
	parameters := make(map[string]string)
	test.BaseParam(parameters)
	parameters["app_id"] = "app_id"
	parameters["file_hash"] = "staticVal2"
	parameters["file_name"] = "staticVal3"
	parameters["contract_name"] = test.ContractNameTest
	parameters["method"] = "save"
	parameters["time"] = "12"

	runtimeInstance := &gasm.RuntimeInstance{
		Log: logger.GetLogger(logger.MODULE_VM),
	}
	runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
