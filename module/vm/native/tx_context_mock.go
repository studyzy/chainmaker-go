package native

import (
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"sync"
)

type dataStore map[string][]byte

type TxContextMock struct {
	lock          *sync.Mutex
	cacheMap dataStore
}

func newTxContextMock(cache dataStore) *TxContextMock {
	return &TxContextMock {
		lock: &sync.Mutex{},
		cacheMap: cache,
	}
}

func (mock *TxContextMock) Get(name string, key []byte) ([]byte, error) {
	mock.lock.Lock()
	defer mock.lock.Unlock()

	k := string(key)
	if name != "" {
		k = name + "::" + k
	}

	return mock.cacheMap[k], nil
}

func (mock *TxContextMock) Put(name string, key []byte, value []byte) error {
	mock.lock.Lock()
	defer mock.lock.Unlock()

	k := string(key)
	if name != "" {
		k = name + "::" + k
	}

	mock.cacheMap[k] = value
	return nil
}

func (mock *TxContextMock) Del(name string, key []byte) error {
	mock.lock.Lock()
	defer mock.lock.Unlock()

	k := string(key)
	if name != "" {
		k = name + "::" + k
	}

	mock.cacheMap[k] = nil
	return nil
}

func (*TxContextMock) CallContract(contractId *commonPb.ContractId,
	method string,
	byteCode []byte,
	parameter map[string]string,
	gasUsed uint64,
	refTxType commonPb.TxType,
	) (*commonPb.ContractResult, commonPb.TxStatusCode) {

	panic("implement me")
}


func (*TxContextMock) GetCurrentResult() []byte {
	panic("implement me")
}

func (*TxContextMock) GetTx() *commonPb.Transaction {
	panic("implement me")
}


func (mock *TxContextMock) GetBlockHeight() int64 {
	return 0
}

func (mock *TxContextMock) GetTxResult() *commonPb.Result {
	panic("implement me")
}

func (mock *TxContextMock) SetTxResult(txResult *commonPb.Result) {
	panic("implement me")
}

func (mock *TxContextMock) GetTxRWSet() *commonPb.TxRWSet {
	panic("implement me")
}

func (mock *TxContextMock) GetCreator(namespace string) *acPb.SerializedMember {
	panic("implement me")
}

func (mock *TxContextMock) GetSender() *acPb.SerializedMember {
	panic("implement me")
}

func (mock *TxContextMock) GetBlockchainStore() protocol.BlockchainStore {
	panic("implement me")
}

func (mock *TxContextMock) GetAccessControl() (protocol.AccessControlProvider, error) {
	panic("implement me")
}

func (mock *TxContextMock) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	panic("implement me")
}

func (mock *TxContextMock) GetTxExecSeq() int {
	panic("implement me")
}

func (mock *TxContextMock) SetTxExecSeq(i int) {
	panic("implement me")
}

func (mock *TxContextMock) GetDepth() int {
	panic("implement me")
}

func (mock *TxContextMock) GetBlockProposer() []byte {
	panic("implement me")
}


func (mock *TxContextMock) PutRecord(contractName string, value []byte) {
	panic("implement me")
}

func (mock *TxContextMock) Select(name string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	panic("implement me")
}

func (mock *TxContextMock) SetStateSqlHandle(i int32, rows protocol.SqlRows) {
	panic("implement me")
}

func (mock *TxContextMock) GetStateSqlHandle(i int32) (protocol.SqlRows, bool) {
	panic("implement me")
}
