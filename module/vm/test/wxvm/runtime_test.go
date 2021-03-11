package wxvm

import (
	"chainmaker.org/chainmaker-go/common/random/uuid"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wxvm"
	"chainmaker.org/chainmaker-go/wxvm/xvm"
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_invoke_cpp(t *testing.T) {
	runtimeInstance := &wxvm.RuntimeInstance{
		ChainId:     "chain01",
		CtxService:  xvm.NewContextService(""),
		CodeManager: xvm.NewCodeManager("chain01", "C:\\tmp\\wxvm-data"),
	}

	logger := logger.GetLoggerByChain(logger.MODULE_VM, "chain01")

	method := "call_contract"
	count := 1
	start := time.Now()
	var wg sync.WaitGroup

	parameters := make(map[string]string)
	parameters[protocol.ContractCreatorOrgIdParam] = "CREATOR_ORG_ID"
	parameters[protocol.ContractCreatorRoleParam] = "CREATOR_ROLE"
	parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
	parameters[protocol.ContractSenderOrgIdParam] = "SENDER_ORG_ID"
	parameters[protocol.ContractSenderRoleParam] = "SENDER_ROLE"
	parameters[protocol.ContractSenderPkParam] = "SENDER_PK"
	parameters[protocol.ContractBlockHeightParam] = "1"
	parameters[protocol.ContractTxIdParam] = uuid.GetUUID()
	parameters[protocol.ContractContextPtrParam] = "0"

	parameters["name"] = "微芯"
	parameters["num"] = "100"
	parameters["num1"] = "220"
	parameters["num2"] = "0"
	parameters["time"] = time.Now().String()
	parameters["file_hash"] = uuid.GetUUID()
	parameters["file_name"] = uuid.GetUUID()
	parameters["tx_id"] = uuid.GetUUID()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			contractId, txContext, byteCode := test.InitContextTest(commonPb.RuntimeType_WXVM)
			//runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
			contractResult := runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
			logger.Infof("contractResult :%+v\n", contractResult)
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Printf("method [%+v], tx count %+v, time cost %+v\n", method, count, time.Since(start))
}
