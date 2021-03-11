package wasmertest

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/vm/test"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/wasmer"

	// pprof 的init函数会将pprof里的一些handler注册到http.DefaultServeMux上
	// 当不使用http.DefaultServeMux来提供http api时，可以查阅其init函数，自己注册handler
	_ "net/http/pprof"
)

// 转账合约
func TestCallWallet(t *testing.T) {
	test.WasmFile = "../../../../test/wasm/asset-rust-0.7.2_v1.0.0.wasm"
	contractId, txContext, bytes := test.InitContextTest(commonPb.RuntimeType_WASMER)
	//bytes, _ = wasm.ReadBytes("../../../../test/wasm/wallet-rust-0.6.2.wasm")
	println("bytes len", len(bytes))

	pool := wasmer.NewVmPoolManager("Wallet")
	start := time.Now().UnixNano() / 1e6

	// 安装
	invokeWalletInit(contractId, txContext, pool, bytes)
	invokeWalletBalanceOfCreator(contractId, txContext, pool, bytes)

	invokeWalletBalanceOf1(contractId, txContext, pool, bytes)
	invokeWalletRegister1(contractId, txContext, pool, bytes)
	invokeWalletBalanceOf1(contractId, txContext, pool, bytes)

	invokeWalletRegister2(contractId, txContext, pool, bytes)

	invokeWalletEmitAmountTo1(contractId, txContext, pool, bytes)
	invokeWalletEmitAmountTo2(contractId, txContext, pool, bytes)

	invokeWalletTransfer1to2(contractId, txContext, pool, bytes)
	invokeWalletBalanceOf1(contractId, txContext, pool, bytes)
	invokeWalletBalanceOf2(contractId, txContext, pool, bytes)

	invokeWalletTransfer2to1(contractId, txContext, pool, bytes)
	invokeWalletBalanceOf1(contractId, txContext, pool, bytes)
	invokeWalletBalanceOf2(contractId, txContext, pool, bytes)

	invokeWalletTransfer1to1(contractId, txContext, pool, bytes)

	invokeWalletQueryAddress1(contractId, txContext, pool, bytes)
	invokeWalletQueryAddress3(contractId, txContext, pool, bytes)

	invokeWalletRegister3(contractId, txContext, pool, bytes)
	invokeWalletApprove1to3(contractId, txContext, pool, bytes)
	invokeWallet3TransferFrom1to2(contractId, txContext, pool, bytes)
	invokeWallet3TransferFrom1to2(contractId, txContext, pool, bytes)

	invokeWalletAllowance1to3(contractId, txContext, pool, bytes)
	invokeWalletAllowance1to2(contractId, txContext, pool, bytes)

	invokeWalletTransfer1to2ErrorAmount(contractId, txContext, pool, bytes)
	invokeWalletTransfer1to2ErrorPk(contractId, txContext, pool, bytes)
	invokeWalletTransfer2to1NoEnough(contractId, txContext, pool, bytes)
	invokeWalletEmitAmountTo1OutOfLimit(contractId, txContext, pool, bytes)
	invokeWalletEmitAmountTo1OutOfInt(contractId, txContext, pool, bytes)

	end := time.Now().UnixNano() / 1e6
	println("end 【spend】", end-start)
	// time.Sleep(time.Second * 5)
}

func invokeWalletInit(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "init_contract"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["issue_limit"] = "1000"
	parameters["total_supply"] = "50000"
	parameters["manager_pk"] = "CREATOR_PK,mpk1,mpk2,mpk3,mpk4"
	parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
	parameters[protocol.ContractSenderPkParam] = "CREATOR_PK"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletRegister1(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "register"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletRegister2(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "register"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk2"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletRegister3(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "register"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk3"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletEmitAmountTo1(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "issue_amount"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["to"] = "pk1"
	parameters["amount"] = "150"
	parameters[protocol.ContractSenderPkParam] = "CREATOR_PK"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletEmitAmountTo2(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "issue_amount"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["to"] = "pk2"
	parameters["amount"] = "100"
	parameters[protocol.ContractSenderPkParam] = "CREATOR_PK"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletEmitAmountTo1OutOfLimit(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "issue_amount"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["to"] = "pk2"
	parameters["amount"] = "100111111"
	parameters[protocol.ContractSenderPkParam] = "CREATOR_PK"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletEmitAmountTo1OutOfInt(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "issue_amount"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["to"] = "pk2"
	parameters["amount"] = "1001111111111111111"
	parameters[protocol.ContractSenderPkParam] = "CREATOR_PK"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletTransfer1to2(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"
	parameters["to"] = "pk2"
	parameters["amount"] = "10"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletTransfer1to1(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"
	parameters["to"] = "pk1"
	parameters["amount"] = "10"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletTransfer1to2ErrorPk(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"
	parameters["to"] = "pk2222"
	parameters["amount"] = "10"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletTransfer1to2ErrorAmount(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"
	parameters["to"] = "pk2222"
	parameters["amount"] = "10dd"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletTransfer2to1(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk2"
	parameters["to"] = "pk1"
	parameters["amount"] = "5"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletTransfer2to1NoEnough(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk2"
	parameters["to"] = "pk1"
	parameters["amount"] = "5000"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletBalanceOf1(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "balance_of"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["owner"] = "pk1"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletBalanceOf2(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "balance_of"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["owner"] = "pk2"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletBalanceOfCreator(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "balance_of"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["owner"] = "CREATOR_PK"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletQueryAddress1(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "query_address"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
func invokeWalletQueryAddress3(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "query_address"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk3"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletApprove1to3(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "approve"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk1"
	parameters["spender"] = "pk3"
	parameters["amount"] = "50"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWallet3TransferFrom1to2(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "transfer_from"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters[protocol.ContractSenderPkParam] = "pk3"
	parameters["from"] = "pk1"
	parameters["to"] = "pk2"
	parameters["amount"] = "40"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletAllowance1to3(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "allowance"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["spender"] = "pk3"
	parameters["owner"] = "pk1"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeWalletAllowance1to2(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "allowance"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["spender"] = "pk2"
	parameters["owner"] = "pk1"

	runtime, _ := pool.NewRuntimeInstance(contractId, txContext, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}
