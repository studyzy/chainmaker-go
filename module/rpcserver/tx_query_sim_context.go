/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpcserver

import (
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/protocol"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
)

// Storage interface for smart contracts, implement TxSimContext
type txQuerySimContextImpl struct {
	tx               *commonPb.Transaction
	txResult         *commonPb.Result
	txReadKeyMap     map[string]*commonPb.TxRead
	txWriteKeyMap    map[string]*commonPb.TxWrite
	txWriteKeySql    []*commonPb.TxWrite
	txWriteKeyDdlSql []*commonPb.TxWrite
	blockchainStore  protocol.BlockchainStore
	vmManager        protocol.VmManager
	gasUsed          uint64 // only for callContract
	currentDepth     int
	currentResult    []byte
	hisResult        []*callContractResult
	sqlRowCache      map[int32]protocol.SqlRows
	kvRowCache       map[int32]protocol.StateIterator
}

type callContractResult struct {
	contractName string
	method       string
	param        map[string]string
	depth        int
	gasUsed      uint64
	result       []byte
}

// StateDB & ReadWriteSet
func (s *txQuerySimContextImpl) Get(contractName string, key []byte) ([]byte, error) {
	// Get from write set
	value, done := s.getFromWriteSet(contractName, key)
	if done {
		s.putIntoReadSet(contractName, key, value)
		return value, nil
	}

	// Get from read set
	value, done = s.getFromReadSet(contractName, key)
	if done {
		return value, nil
	}

	// Get from db
	value, err := s.blockchainStore.ReadObject(contractName, key)
	if err != nil {
		return nil, err
	}

	// if get from db success, put into read set
	s.putIntoReadSet(contractName, key, value)
	return value, nil
}

func (s *txQuerySimContextImpl) Put(contractName string, key []byte, value []byte) error {
	s.putIntoWriteSet(contractName, key, value)
	return nil
}

func (s *txQuerySimContextImpl) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
	txWrite := &commonPb.TxWrite{
		Key:          nil,
		Value:        value,
		ContractName: contractName,
	}
	s.txWriteKeySql = append(s.txWriteKeySql, txWrite)
	if sqlType == protocol.SqlTypeDdl {
		s.txWriteKeyDdlSql = append(s.txWriteKeyDdlSql, txWrite)
	}
}

func (s *txQuerySimContextImpl) Del(contractName string, key []byte) error {
	s.putIntoWriteSet(contractName, key, nil)
	return nil
}

func (s *txQuerySimContextImpl) Select(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	return s.blockchainStore.SelectObject(contractName, startKey, limit)
}

func (s *txQuerySimContextImpl) GetCreator(contractName string) *acPb.SerializedMember {
	if creatorByte, err := s.Get(commonPb.ContractName_SYSTEM_CONTRACT_STATE.String(), []byte(protocol.ContractCreator+contractName)); err != nil {
		return nil
	} else {
		creator := &acPb.SerializedMember{}
		if err = proto.Unmarshal(creatorByte, creator); err != nil {
			return nil
		}
		return creator
	}
}

func (s *txQuerySimContextImpl) GetSender() *acPb.SerializedMember {
	return s.tx.Header.Sender
}

func (s *txQuerySimContextImpl) GetBlockHeight() int64 {
	if lastBlock, err := s.blockchainStore.GetLastBlock(); err != nil {
		return 0
	} else {
		return lastBlock.Header.BlockHeight
	}
}

func (s *txQuerySimContextImpl) GetBlockProposer() []byte {
	if lastBlock, err := s.blockchainStore.GetLastBlock(); err != nil {
		return nil
	} else {
		return lastBlock.Header.Proposer
	}
}

func (s *txQuerySimContextImpl) putIntoReadSet(contractName string, key []byte, value []byte) {
	s.txReadKeyMap[constructKey(contractName, key)] = &commonPb.TxRead{
		Key:          key,
		Value:        value,
		ContractName: contractName,
		Version:      nil,
	}
}

func (s *txQuerySimContextImpl) putIntoWriteSet(contractName string, key []byte, value []byte) {
	s.txWriteKeyMap[constructKey(contractName, key)] = &commonPb.TxWrite{
		Key:          key,
		Value:        value,
		ContractName: contractName,
	}
}

func (s *txQuerySimContextImpl) getFromReadSet(contractName string, key []byte) ([]byte, bool) {
	if txRead, ok := s.txReadKeyMap[constructKey(contractName, key)]; ok {
		return txRead.Value, true
	}
	return nil, false
}

func (s *txQuerySimContextImpl) getFromWriteSet(contractName string, key []byte) ([]byte, bool) {
	if txWrite, ok := s.txWriteKeyMap[constructKey(contractName, key)]; ok {
		return txWrite.Value, true
	}
	return nil, false
}

func (s *txQuerySimContextImpl) GetTx() *commonPb.Transaction {
	return s.tx
}

func (s *txQuerySimContextImpl) GetBlockchainStore() protocol.BlockchainStore {
	return s.blockchainStore
}

// Get access control service
func (s *txQuerySimContextImpl) GetAccessControl() (protocol.AccessControlProvider, error) {
	if s.vmManager.GetAccessControl() == nil {
		return nil, errors.New("access control for tx sim context is nil")
	}
	return s.vmManager.GetAccessControl(), nil
}

// Get organization service
func (s *txQuerySimContextImpl) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	if s.vmManager.GetChainNodesInfoProvider() == nil {
		return nil, errors.New("chainNodesInfoProvider for tx sim context is nil")
	}
	return s.vmManager.GetChainNodesInfoProvider(), nil
}

func (s *txQuerySimContextImpl) GetTxRWSet(runVmSuccess bool) *commonPb.TxRWSet {
	txRwSet := &commonPb.TxRWSet{
		TxId:     s.tx.Header.TxId,
		TxReads:  nil,
		TxWrites: nil,
	}
	if !runVmSuccess {
		// Query does not contain DDL
		// txRwSet.TxWrites = append(txRwSet.TxWrites, s.txWriteKeyDdlSql...)
		return txRwSet
	}
	for _, txRead := range s.txReadKeyMap {
		txRwSet.TxReads = append(txRwSet.TxReads, txRead)
	}
	for _, txWrite := range s.txWriteKeyMap {
		txRwSet.TxWrites = append(txRwSet.TxWrites, txWrite)
	}
	txRwSet.TxWrites = append(txRwSet.TxWrites, s.txWriteKeySql...)
	return txRwSet
}

func (s *txQuerySimContextImpl) GetTxExecSeq() int {
	return 0
}

func (s *txQuerySimContextImpl) SetTxExecSeq(int) {
	return
}

// Get the tx result
func (s *txQuerySimContextImpl) GetTxResult() *commonPb.Result {
	return s.txResult
}

// Set the tx result
func (s *txQuerySimContextImpl) SetTxResult(txResult *commonPb.Result) {
	s.txResult = txResult
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}

func (s *txQuerySimContextImpl) CallContract(contractId *commonPb.ContractId, method string, byteCode []byte, parameter map[string]string, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult, commonPb.TxStatusCode) {
	s.gasUsed = gasUsed
	s.currentDepth = s.currentDepth + 1
	if s.currentDepth > protocol.CallContractDepth {
		contractResult := &commonPb.ContractResult{
			Code:    commonPb.ContractResultCode_FAIL,
			Result:  nil,
			Message: fmt.Sprintf("CallContract too depth %d", s.currentDepth),
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
	r, code := s.vmManager.RunContract(contractId, method, byteCode, parameter, s, s.gasUsed, refTxType)

	result := callContractResult{
		depth:        s.currentDepth,
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

func (s *txQuerySimContextImpl) GetCurrentResult() []byte {
	return s.currentResult
}

func (s *txQuerySimContextImpl) GetDepth() int {
	return s.currentDepth
}

func (s *txQuerySimContextImpl) SetStateSqlHandle(index int32, rows protocol.SqlRows) {
	s.sqlRowCache[index] = rows
}

func (s *txQuerySimContextImpl) GetStateSqlHandle(index int32) (protocol.SqlRows, bool) {
	data, ok := s.sqlRowCache[index]
	return data, ok
}

func (s *txQuerySimContextImpl) SetStateKvHandle(index int32, rows protocol.StateIterator) {
	s.kvRowCache[index] = rows
}

func (s *txQuerySimContextImpl) GetStateKvHandle(index int32) (protocol.StateIterator, bool) {
	data, ok := s.kvRowCache[index]
	return data, ok
}
