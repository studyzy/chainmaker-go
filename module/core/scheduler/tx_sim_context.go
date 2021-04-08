/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	acpb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"sort"
)

// Storage interface for smart contracts
type txSimContextImpl struct {
	txExecSeq     int
	txResult      *commonpb.Result
	txRWSet       *commonpb.TxRWSet
	tx            *commonpb.Transaction
	txReadKeyMap  map[string]*commonpb.TxRead
	txWriteKeyMap map[string]*commonpb.TxWrite
	txWriteKeySql []*commonpb.TxWrite
	snapshot      protocol.Snapshot
	vmManager     protocol.VmManager
	gasUsed       uint64 // only for callContract
	currentDeep   int
	currentResult []byte
	hisResult     []*callContractResult
	sqlRowCache   map[int32]protocol.SqlRows
}

type callContractResult struct {
	contractName string
	method       string
	param        map[string]string
	deep         int
	gasUsed      uint64
	result       []byte
}

func (s *txSimContextImpl) Get(contractName string, key []byte) ([]byte, error) {
	// Get from write set
	if value, done := s.getFromWriteSet(contractName, key); done {
		s.putIntoReadSet(contractName, key, value)
		return value, nil
	}

	// Get from read set
	if value, done := s.getFromReadSet(contractName, key); done {
		return value, nil
	}

	// Get from db
	if value, err := s.snapshot.GetKey(s.txExecSeq, contractName, key); err != nil {
		return nil, err
	} else {
		// if get from db success, put into read set
		s.putIntoReadSet(contractName, key, value)
		return value, nil
	}
}

func (s *txSimContextImpl) Put(contractName string, key []byte, value []byte) error {
	s.putIntoWriteSet(contractName, key, value)
	return nil
}

func (s *txSimContextImpl) PutSql(contractName string, value []byte) {
	txWrite := &commonpb.TxWrite{
		Key:          nil,
		Value:        value,
		ContractName: contractName,
	}
	s.txWriteKeySql = append(s.txWriteKeySql, txWrite)
}

func (s *txSimContextImpl) Del(contractName string, key []byte) error {
	s.putIntoWriteSet(contractName, key, nil)
	return nil
}

func (s *txSimContextImpl) Select(contractName string, startKey []byte, limit []byte) (protocol.Iterator, error) {
	// 将来需要把txRwSet的最新状态填充到Iter中去，覆盖或者替换，才是完整的最新的Iter，否则就只是数据库的状态
	return s.snapshot.GetBlockchainStore().SelectObject(contractName, startKey, limit), nil
}

func (s *txSimContextImpl) GetCreator(contractName string) *acpb.SerializedMember {
	if creatorByte, err := s.Get(contractName, []byte(protocol.ContractCreator)); err != nil {
		return nil
	} else {
		creator := &acpb.SerializedMember{}
		if err = proto.Unmarshal(creatorByte, creator); err != nil {
			return nil
		}
		return creator
	}
}

func (s *txSimContextImpl) GetSender() *acpb.SerializedMember {
	return s.tx.Header.Sender
}

func (s *txSimContextImpl) putIntoReadSet(contractName string, key []byte, value []byte) {
	s.txReadKeyMap[constructKey(contractName, key)] = &commonpb.TxRead{
		Key:          key,
		Value:        value,
		ContractName: contractName,
		Version:      nil,
	}
}

func (s *txSimContextImpl) putIntoWriteSet(contractName string, key []byte, value []byte) {
	s.txWriteKeyMap[constructKey(contractName, key)] = &commonpb.TxWrite{
		Key:          key,
		Value:        value,
		ContractName: contractName,
	}
}

func (s *txSimContextImpl) getFromReadSet(contractName string, key []byte) ([]byte, bool) {
	if txRead, ok := s.txReadKeyMap[constructKey(contractName, key)]; ok {
		return txRead.Value, true
	}
	return nil, false
}

func (s *txSimContextImpl) getFromWriteSet(contractName string, key []byte) ([]byte, bool) {
	if txWrite, ok := s.txWriteKeyMap[constructKey(contractName, key)]; ok {
		return txWrite.Value, true
	}
	return nil, false
}

// Get the corresponding transaction
func (s *txSimContextImpl) GetTx() *commonpb.Transaction {
	return s.tx
}

// Get blockchain storage
func (s *txSimContextImpl) GetBlockchainStore() protocol.BlockchainStore {
	return s.snapshot.GetBlockchainStore()
}

// GetAccessControl get access control service
func (s *txSimContextImpl) GetAccessControl() (protocol.AccessControlProvider, error) {
	if s.vmManager.GetAccessControl() == nil {
		return nil, errors.New("access control for tx sim context is nil")
	}
	return s.vmManager.GetAccessControl(), nil
}

// Get organization service
func (s *txSimContextImpl) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	if s.vmManager.GetChainNodesInfoProvider() == nil {
		return nil, errors.New("chainNodesInfoProvider for tx sim context is nil, may be running in singleton mode")
	}
	return s.vmManager.GetChainNodesInfoProvider(), nil
}

// GetTxRWSet return current transaction read write set
func (s *txSimContextImpl) GetTxRWSet() *commonpb.TxRWSet {
	if s.txRWSet != nil {
		return s.txRWSet
	}
	s.txRWSet = &commonpb.TxRWSet{
		TxId:     s.tx.Header.TxId,
		TxReads:  make([]*commonpb.TxRead, 0, len(s.txReadKeyMap)),
		TxWrites: make([]*commonpb.TxWrite, 0, len(s.txWriteKeyMap)),
	}
	{
		txIds := make([]string, 0, len(s.txReadKeyMap))
		for txId := range s.txReadKeyMap {
			txIds = append(txIds, txId)
		}
		sort.Strings(txIds)
		for _, k := range txIds {
			s.txRWSet.TxReads = append(s.txRWSet.TxReads, s.txReadKeyMap[k])
		}
	}
	{
		txIds := make([]string, 0, len(s.txWriteKeyMap))
		for txId := range s.txWriteKeyMap {
			txIds = append(txIds, txId)
		}
		sort.Strings(txIds)
		for _, k := range txIds {
			s.txRWSet.TxWrites = append(s.txRWSet.TxWrites, s.txWriteKeyMap[k])
		}
		// sql nil key tx writes
		s.txRWSet.TxWrites = append(s.txRWSet.TxWrites, s.txWriteKeySql...)
	}
	return s.txRWSet
}

// Get the height of the corresponding block
func (s *txSimContextImpl) GetBlockHeight() int64 {
	return s.snapshot.GetBlockHeight()
}

func (s *txSimContextImpl) GetBlockProposer() []byte {
	return s.snapshot.GetBlockProposer()
}

// Obtain the corresponding transaction execution sequence
func (s *txSimContextImpl) GetTxExecSeq() int {
	return s.txExecSeq
}

// set the corresponding transaction execution sequence
func (s *txSimContextImpl) SetTxExecSeq(txExecSeq int) {
	s.txExecSeq = txExecSeq
}

// Get the tx result
func (s *txSimContextImpl) GetTxResult() *commonpb.Result {
	return s.txResult
}

// Set the tx result
func (s *txSimContextImpl) SetTxResult(txResult *commonpb.Result) {
	s.txResult = txResult
}

// Cross contract call
func (s *txSimContextImpl) CallContract(contractId *commonpb.ContractId, method string, byteCode []byte, parameter map[string]string, gasUsed uint64, refTxType commonpb.TxType) (*commonpb.ContractResult, commonpb.TxStatusCode) {
	s.gasUsed = gasUsed
	s.currentDeep = s.currentDeep + 1
	if s.currentDeep > protocol.CallContractDeep {
		contractResult := &commonpb.ContractResult{
			Code:    commonpb.ContractResultCode_FAIL,
			Result:  nil,
			Message: fmt.Sprintf("CallContract too deep %d", s.currentDeep),
		}
		return contractResult, commonpb.TxStatusCode_CONTRACT_TOO_DEEP_FAILED
	}
	if s.gasUsed > protocol.GasLimit {
		contractResult := &commonpb.ContractResult{
			Code:    commonpb.ContractResultCode_FAIL,
			Result:  nil,
			Message: fmt.Sprintf("There is not enough gas, gasUsed %d GasLimit %d ", gasUsed, int64(protocol.GasLimit)),
		}
		return contractResult, commonpb.TxStatusCode_CONTRACT_FAIL
	}
	r, code := s.vmManager.RunContract(contractId, method, byteCode, parameter, s, s.gasUsed, refTxType)

	result := callContractResult{
		deep:         s.currentDeep,
		gasUsed:      s.gasUsed,
		result:       r.Result,
		contractName: contractId.ContractName,
		method:       method,
		param:        parameter,
	}
	s.hisResult = append(s.hisResult, &result)
	s.currentResult = r.Result
	s.currentDeep = s.currentDeep - 1
	return r, code
}

// Obtain the execution result of current contract (cross contract)
func (s *txSimContextImpl) GetCurrentResult() []byte {
	return s.currentResult
}

// Get contract call depth
func (s *txSimContextImpl) GetDepth() int {
	return s.currentDeep
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}

func (s *txSimContextImpl) SetStateSqlHandle(index int32, rows protocol.SqlRows) {
	s.sqlRowCache[index] = rows
}

func (s *txSimContextImpl) GetStateSqlHandle(index int32) (protocol.SqlRows, bool) {
	data, ok := s.sqlRowCache[index]
	return data, ok
}
