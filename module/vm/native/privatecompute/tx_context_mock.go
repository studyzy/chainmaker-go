/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
package privatecompute

import (
	"sync"

	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
)

type dataStore map[string][]byte

type TxContextMock struct {
	lock     *sync.Mutex
	cacheMap dataStore
}

func newTxContextMock(cache dataStore) *TxContextMock {
	return &TxContextMock{
		lock:     &sync.Mutex{},
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

func (*TxContextMock) CallContract(contract *commonPb.Contract,
	method string,
	byteCode []byte,
	parameter map[string][]byte,
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

func (mock *TxContextMock) GetBlockHeight() uint64 {
	return 0
}

func (mock *TxContextMock) GetTxResult() *commonPb.Result {
	panic("implement me")
}

func (mock *TxContextMock) SetTxResult(txResult *commonPb.Result) {
	panic("implement me")
}

func (mock *TxContextMock) GetTxRWSet(runVmSuccess bool) *commonPb.TxRWSet {
	panic("implement me")
}

func (mock *TxContextMock) GetCreator(namespace string) *acPb.Member {
	panic("implement me")
}

func (mock *TxContextMock) GetSender() *acPb.Member {
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

func (mock *TxContextMock) GetBlockProposer() *acPb.Member {
	panic("implement me")
}

func (mock *TxContextMock) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
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

func (mock *TxContextMock) GetStateKvHandle(int32) (protocol.StateIterator, bool) {
	panic("implement me")
}

func (mock *TxContextMock) SetStateKvHandle(int32, protocol.StateIterator) {
	panic("implement me")
}
