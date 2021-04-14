/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package waci

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/wasi"
)

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQuery() int32 {
	err := wasi.ExecuteQuery(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory)
	if err == nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOneLen() int32 {
	data, err := wasi.ExecuteQueryOne(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory, s.GetStateCache)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOne() int32 {
	return s.ExecuteQueryOneLen()
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) RSHasNext() int32 {
	err := wasi.RSHasNext(s.RequestBody, s.TxSimContext, s.Vm.Memory)
	if err == nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSNextLen get result set length from chain
func (s *WaciInstance) RSNextLen() int32 {
	data, err := wasi.RSNext(s.RequestBody, s.TxSimContext, s.Vm.Memory, s.GetStateCache)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSNextLen get one row from result set
func (s *WaciInstance) RSNext() int32 {
	return s.RSNextLen()
}

// RSClose close sql statement
func (s *WaciInstance) RSClose() int32 {
	err := wasi.RSClose(s.RequestBody, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteUpdate execute update and insert sql, allow single row change
// as: update table set name = 'Tom' where uniqueKey='xxx'
func (s *WaciInstance) ExecuteUpdate() int32 {
	err := wasi.ExecuteUpdate(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory, s.ChainId)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteDDL execute DDL sql, for init_contract or upgrade method. allow table create/alter/drop/truncate
//
// allow:     [CREATE TABLE tableName] [ALTER TABLE tableName]
//            [DROP TABLE tableName]   [TRUNCATE TABLE tableName]
//
// not allow: [CREATE DATABASE dbName] [CREATE TABLE dbName.tableName] [ALTER TABLE dbName.tableName]
//			  [DROP DATABASE dbName]   [DROP TABLE dbName.tableName]   [TRUNCATE TABLE dbName.tableName]
//
// You must have a primary key to create a table
func (s *WaciInstance) ExecuteDDL() int32 {
	err := wasi.ExecuteDDL(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
