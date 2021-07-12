/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package waci

import (
	"chainmaker.org/chainmaker/protocol"
)

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQuery() int32 {
	err := wacsi.ExecuteQuery(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory, s.ChainId)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOneLen() int32 {
	return s.executeQueryOneCore(true)
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOne() int32 {
	return s.executeQueryOneCore(false)
}

func (s *WaciInstance) executeQueryOneCore(isLen bool) int32 {
	data, err := wacsi.ExecuteQueryOne(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory, s.GetStateCache, s.ChainId, isLen)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) RSHasNext() int32 {
	err := wacsi.RSHasNext(s.RequestBody, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSNextLen get result set length from chain
func (s *WaciInstance) RSNextLen() int32 {
	return s.rsNextCore(true)
}

// RSNextLen get one row from result set
func (s *WaciInstance) RSNext() int32 {
	return s.rsNextCore(false)
}

func (s *WaciInstance) rsNextCore(isLen bool) int32 {
	data, err := wacsi.RSNext(s.RequestBody, s.TxSimContext, s.Vm.Memory, s.GetStateCache, isLen)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSClose close sql statement
func (s *WaciInstance) RSClose() int32 {
	err := wacsi.RSClose(s.RequestBody, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteUpdate execute update and insert sql, allow single row change
// as: update table set name = 'Tom' where uniqueKey='xxx'
func (s *WaciInstance) ExecuteUpdate() int32 {
	err := wacsi.ExecuteUpdate(s.RequestBody, s.ContractId.Name, s.Method, s.TxSimContext, s.Vm.Memory, s.ChainId)
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
	//wacsi.IsManageContract()
	err := wacsi.ExecuteDDL(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory, s.Method)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
