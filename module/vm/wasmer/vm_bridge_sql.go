/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
)

var rowIndex int32 = 0
var verifySql = &types.StandardSqlVerify{}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQuery() int32 {
	err := wacsi.ExecuteQuery(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.ChainId)
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
	data, err := wacsi.ExecuteQueryOne(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache, s.ChainId, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSHasNext return is there a next line, 1 is has next row, 0 is no next row
func (s *WaciInstance) RSHasNext() int32 {
	err := wacsi.RSHasNext(s.RequestBody, s.Sc.TxSimContext, s.Memory)
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
	data, err := wacsi.RSNext(s.RequestBody, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSClose close sql statement
func (s *WaciInstance) RSClose() int32 {
	err := wacsi.RSClose(s.RequestBody, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteUpdate execute update and insert sql, allow single row change
// as: update table set name = 'Tom' where uniqueKey='xxx'
func (s *WaciInstance) ExecuteUpdate() int32 {
	err := wacsi.ExecuteUpdate(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.ChainId)
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
	err := wacsi.ExecuteDDL(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.Sc.method)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
