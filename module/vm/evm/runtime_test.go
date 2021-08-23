package evm

import (
	"encoding/hex"
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/evm/test"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker/pb-go/common"
	"gotest.tools/assert"
)

const certFilePath = "./test/config/admin1.sing.crt"
const byteCodeFilePath = "./test/contracts/contract01/token.bin"

func TestRuntimeInstance_Install(t *testing.T) {
	//部署合约
	method := "init_contract"
	test.CertFilePath = certFilePath
	test.ByteCodeFile = byteCodeFilePath
	parameters := make(map[string][]byte)
	contractId, txContext, byteCode := test.InitContextTest(common.RuntimeType_EVM)

	runtimeInstance := &RuntimeInstance{
		ChainId:      "chain01",
		Log:          logger.GetLogger(logger.MODULE_VM),
		TxSimContext: txContext,
	}

	loggerByChain := logger.GetLoggerByChain(logger.MODULE_VM, "chain01")

	byteCode, _ = hex.DecodeString(string(byteCode))
	test.BaseParam(parameters)
	parameters["data"] = []byte("00000000000000000000000013f0c1639a9931b0ce17e14c83f96d4732865b58")
	contractResult := runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	loggerByChain.Infof("ContractResult Code:%+v", contractResult.Code)
	loggerByChain.Infof("ContractResult ContractEvent:%+v", contractResult.ContractEvent)
	loggerByChain.Infof("ContractResult GasUsed:%+v", contractResult.GasUsed)
	loggerByChain.Infof("ContractResult Message:%+v", contractResult.Message)
	loggerByChain.Infof("ContractResult Result:%+X", contractResult.Result)
}
func TestRuntimeInstance_Invoke(t *testing.T) {
	//调用合约
	method := "4f9d719e" //testEvent
	test.ByteCodeFile = "./test/contracts/contract01/token_body.bin"
	test.CertFilePath = "./test/config/admin1.sing.crt"
	parameters := make(map[string][]byte)
	contractId, txContext, byteCode := test.InitContextTest(common.RuntimeType_EVM)

	runtimeInstance := &RuntimeInstance{
		ChainId:      "chain01",
		Log:          logger.GetLogger(logger.MODULE_VM),
		TxSimContext: txContext,
	}

	loggerByChain := logger.GetLoggerByChain(logger.MODULE_VM, "chain01")

	byteCode, _ = hex.DecodeString(string(byteCode))
	test.BaseParam(parameters)
	parameters["data"] = []byte("4f9d719e")
	contractResult := runtimeInstance.Invoke(contractId, method, byteCode, parameters, txContext, 0)
	loggerByChain.Infof("ContractResult Code:%+v", contractResult.Code)
	loggerByChain.Infof("ContractResult ContractEvent:%+v", contractResult.ContractEvent)
	loggerByChain.Infof("ContractResult GasUsed:%+v", contractResult.GasUsed)
	loggerByChain.Infof("ContractResult Message:%+v", contractResult.Message)
	loggerByChain.Infof("ContractResult Result:%+X", contractResult.Result)
}
func TestConvertEvmContractName(t *testing.T) {
	name := "0x7162629f540a9e19eCBeEa163eB8e48eC898Ad00"
	addr, _ := contractNameToAddress(name)
	t.Logf("evm addr:%s", addr.Text(16))
	assert.Equal(t, strings.ToLower(name[2:]), addr.Text(16))
}
