package test

import (
	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/vm"
	"chainmaker.org/chainmaker-go/wasmer"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
	"chainmaker.org/chainmaker-go/wxvm/xvm"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
)

var testOrgId = "wx-org1.chainmaker.org"

var CertFilePath = "../../../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"
var WasmFile = "D:\\develop\\workspace\\chainMaker\\chainmaker-contract-sdk-rust\\target\\wasm32-unknown-unknown\\release\\chainmaker_contract.wasm"

var txType = commonPb.TxType_INVOKE_USER_CONTRACT
var pool *wasmer.VmPoolManager

const (
	ContractNameTest    = "contract01"
	ContractVersionTest = "v1.0.0"
	ChainIdTest         = "chain01"
)

func GetVmPoolManager() *wasmer.VmPoolManager {
	if pool == nil {
		pool = wasmer.NewVmPoolManager(ChainIdTest)
	}
	return pool
}

var bytes []byte
var file []byte

// 初始化上下文和wasm字节码
func InitContextTest(runtimeType commonPb.RuntimeType) (*commonPb.ContractId, *TxContextMockTest, []byte) {
	if bytes == nil {
		bytes, _ = wasm.ReadBytes(WasmFile)
		fmt.Printf("Wasm file size=%d\n", len(bytes))
	}

	contractId := commonPb.ContractId{
		ContractName:    ContractNameTest,
		ContractVersion: ContractVersionTest,
		RuntimeType:     runtimeType,
	}

	wxvmCodeManager := xvm.NewCodeManager(ChainIdTest, "tmp/wxvm-data")
	wxvmContextService := xvm.NewContextService(ChainIdTest)
	log := logger.GetLoggerByChain(logger.MODULE_VM, ChainIdTest)

	if file == nil {
		var err error
		file, err = ioutil.ReadFile(CertFilePath)
		if err != nil {
			panic("file is nil" + err.Error())
		}
	}
	sender := &acPb.SerializedMember{
		OrgId:      testOrgId,
		MemberInfo: file,
		IsFullCert: true,
	}

	txContext := TxContextMockTest{
		lock: &sync.Mutex{},
		vmManager: &vm.ManagerImpl{
			WasmerVmPoolManager:    GetVmPoolManager(),
			WxvmCodeManager:        wxvmCodeManager,
			WxvmContextService:     wxvmContextService,
			SnapshotManager:        nil,
			AccessControl:          accesscontrol.MockAccessControl(),
			ChainNodesInfoProvider: nil,
			ChainId:                ChainIdTest,
			Log:                    log,
		},
		hisResult: make([]*callContractResult, 0),
		creator:   sender,
		sender:    sender,
		cacheMap:  make(map[string][]byte),
	}

	versionKey := []byte(protocol.ContractVersion)
	runtimeTypeKey := []byte(protocol.ContractRuntimeType)
	versionedByteCodeKey := append([]byte(protocol.ContractByteCode), []byte(contractId.ContractVersion)...)

	txContext.Put(contractId.ContractName, versionedByteCodeKey, bytes)
	txContext.Put(contractId.ContractName, versionKey, []byte(contractId.ContractVersion))
	txContext.Put(contractId.ContractName, runtimeTypeKey, []byte(strconv.Itoa(int(runtimeType))))

	txContext.Put(contractId.ContractName+"a", versionedByteCodeKey, bytes)
	txContext.Put(contractId.ContractName+"a", versionKey, []byte(contractId.ContractVersion))
	txContext.Put(contractId.ContractName+"a", runtimeTypeKey, []byte(strconv.Itoa(int(runtimeType))))

	return &contractId, &txContext, bytes
}

// test
// test
// test
// test

type TxContextMockTest struct {
	lock          *sync.Mutex
	vmManager     protocol.VmManager
	gasUsed       uint64 // only for callContract
	currentDepth  int
	currentResult []byte
	hisResult     []*callContractResult

	sender   *acPb.SerializedMember
	creator  *acPb.SerializedMember
	cacheMap map[string][]byte
}

type callContractResult struct {
	contractName string
	method       string
	param        map[string]string
	deep         int
	gasUsed      uint64
	result       []byte
}

func (s *TxContextMockTest) Get(name string, key []byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	k := string(key)
	if name != "" {
		k = name + "::" + k
	}
	//println("【get】 key:" + k)
	//fms.Println("【get】 key:", k, "val:", cacheMap[k])
	return s.cacheMap[k], nil
	//return nil,nil
	//data := "hello"
	//for i := 0; i < 70; i++ {
	//	for i := 0; i < 100; i++ {//1k
	//		data += "1234567890"
	//	}
	//}
	//return []byte(data), nil
}

func (s *TxContextMockTest) Put(name string, key []byte, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	k := string(key)
	//v := string(value)
	if name != "" {
		k = name + "::" + k
	}
	//println("【put】 key:" + k)
	//fmt.Println("【put】 key:", k, "val:", value)
	s.cacheMap[k] = value
	return nil
}

func (s *TxContextMockTest) Del(name string, key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	k := string(key)
	//v := string(value)
	if name != "" {
		k = name + "::" + k
	}
	//println("【put】 key:" + k)
	s.cacheMap[k] = nil
	return nil
}

func (*TxContextMockTest) Select(name string, startKey []byte, limit []byte) (protocol.Iterator, error) {
	panic("implement me")
}

func (s *TxContextMockTest) CallContract(contractId *commonPb.ContractId, method string, byteCode []byte,
	parameter map[string]string, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
	s.gasUsed = gasUsed
	s.currentDepth = s.currentDepth + 1
	if s.currentDepth > protocol.CallContractDepth {
		contractResult := &commonPb.ContractResult{
			Code:    commonPb.ContractResultCode_FAIL,
			Result:  nil,
			Message: fmt.Sprintf("CallContract too deep %d", s.currentDepth),
		}
		return contractResult, commonPb.TxStatusCode_CONTRACT_TOO_DEEP_FAILED
	}
	if s.gasUsed > protocol.GasLimit {
		contractResult := &commonPb.ContractResult{
			Code:    commonPb.ContractResultCode_FAIL,
			Result:  nil,
			Message: fmt.Sprintf("There is not enough gas, gasUsed %d GasLimit %d ", gasUsed, int64(protocol.GasLimit)),
		}
		return contractResult, commonPb.TxStatusCode_CONTRACT_FAIL
	}
	r, code := s.vmManager.RunContract(contractId, method, byteCode, parameter,s, s.gasUsed, refTxType)

	result := callContractResult{
		deep:         s.currentDepth,
		gasUsed:      s.gasUsed,
		result:       r.Result,
		contractName: contractId.ContractName,
		method:       method,
		param:        parameter,
	}
	s.hisResult = append(s.hisResult, &result)
	s.currentResult = r.Result
	s.currentDepth = s.currentDepth - 1
	return r, code
}

func (s *TxContextMockTest) GetCurrentResult() []byte {
	return s.currentResult
}

func (s *TxContextMockTest) GetTx() *commonPb.Transaction {
	return &commonPb.Transaction{
		Header: &commonPb.TxHeader{
			ChainId:        ChainIdTest,
			Sender:         s.GetSender(),
			TxType:         txType,
			TxId:           "1234567890123456789012345678901234567890123456789012345678901234",
			Timestamp:      0,
			ExpirationTime: 0,
		},
		RequestPayload:   nil,
		RequestSignature: nil,
		Result:           nil,
	}
}

func (*TxContextMockTest) GetBlockHeight() int64 {
	return 0
}
func (s *TxContextMockTest) GetTxResult() *commonPb.Result {
	panic("implement me")
}

func (s *TxContextMockTest) SetTxResult(txResult *commonPb.Result) {
	panic("implement me")
}

func (TxContextMockTest) GetTxRWSet() *commonPb.TxRWSet {
	return &commonPb.TxRWSet{
		TxId:     "txId",
		TxReads:  nil,
		TxWrites: nil,
	}
}

func (s *TxContextMockTest) GetCreator(namespace string) *acPb.SerializedMember {
	return s.creator
}

func (s *TxContextMockTest) GetSender() *acPb.SerializedMember {
	return s.sender
}

func (*TxContextMockTest) GetBlockchainStore() protocol.BlockchainStore {
	panic("implement me")
}

func (*TxContextMockTest) GetAccessControl() (protocol.AccessControlProvider, error) {
	panic("implement me")
}

func (s *TxContextMockTest) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	panic("implement me")
}

func (*TxContextMockTest) GetTxExecSeq() int {
	panic("implement me")
}

func (*TxContextMockTest) SetTxExecSeq(i int) {
	panic("implement me")
}

func (s *TxContextMockTest) GetDepth() int {
	return s.currentDepth
}

func BaseParam(parameters map[string]string) {
	parameters[protocol.ContractTxIdParam] = "TX_ID"
	parameters[protocol.ContractCreatorOrgIdParam] = "CREATOR_ORG_ID"
	parameters[protocol.ContractCreatorRoleParam] = "CREATOR_ROLE"
	parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
	parameters[protocol.ContractSenderOrgIdParam] = "SENDER_ORG_ID"
	parameters[protocol.ContractSenderRoleParam] = "SENDER_ROLE"
	parameters[protocol.ContractSenderPkParam] = "SENDER_PK"
	parameters[protocol.ContractBlockHeightParam] = "111"
}
