/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker-go/wasi"
)

var rowIndex int32 = 0
var verifySql = &types.StandardSqlVerify{}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQuery() int32 {
	err := wasi.ExecuteQuery(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOneLen() int32 {
	data, err := wasi.ExecuteQueryOne(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache)
	s.Sc.GetStateCache = data // reset data
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

//func (s *WaciInstance) executeQueryOneCore(isGetLen bool) int32 {
//	req := serialize.EasyUnmarshal(s.RequestBody)
//	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
//	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
//	sql := sqlI.(string)
//	ptr := valuePtr.(int32)
//
//	// verify
//	if err := verifySql.VerifyDQLSql(sql); err != nil {
//		s.recordMsg("verify query one sql error, " + err.Error())
//		return protocol.ContractSdkSignalResultFail
//	}
//
//	if !isGetLen {
//		data := s.Sc.GetStateCache
//		if data != nil && len(data) > 0 {
//			copy(s.Memory[ptr:ptr+int32(len(data))], data)
//		}
//		s.Sc.GetStateCache = nil
//		return protocol.ContractSdkSignalResultSuccess
//	}
//
//	// execute
//	row, err := s.Sc.TxSimContext.GetBlockchainStore().QuerySingle(s.Sc.ContractId.ContractName, sql)
//	if err != nil {
//		s.recordMsg("ctx query error, " + err.Error())
//		return protocol.ContractSdkSignalResultFail
//	}
//
//	var data map[string]string
//	if row.IsEmpty() {
//		data = make(map[string]string, 0)
//	} else {
//		data, err = row.Data()
//		if err != nil {
//			s.recordMsg("ctx query get data to map error, " + err.Error())
//			return protocol.ContractSdkSignalResultFail
//		}
//	}
//	ec := serialize.NewEasyCodecWithMap(data)
//	bytes := ec.Marshal()
//	copy(s.Memory[ptr:ptr+4], IntToBytes(int32(len(bytes))))
//	s.Sc.GetStateCache = bytes
//
//	return protocol.ContractSdkSignalResultSuccess
//}

// RSHasNext return is there a next line, 1 is has next row, 0 is no next row
func (s *WaciInstance) RSHasNext() int32 {
	err := wasi.RSHasNext(s.RequestBody, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// RSNextLen get result set length from chain
func (s *WaciInstance) RSNextLen() int32 {
	data, err := wasi.RSNext(s.RequestBody, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache)
	s.Sc.GetStateCache = data // reset data
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

//func (s *WaciInstance) rsNextCore(isGetLen bool) int32 {
//	req := serialize.EasyUnmarshal(s.RequestBody)
//	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
//	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
//
//	rsIndex := rsIndexI.(int32)
//	ptr := valuePtrI.(int32)
//
//	// get handle
//	rows, ok := s.Sc.TxSimContext.GetStateSqlHandle(rsIndex)
//	if !ok {
//		s.recordMsg("ctx can not found rs_index[" + strconv.Itoa(int(rsIndex)) + "]")
//		return protocol.ContractSdkSignalResultFail
//	}
//
//	// get data
//	if !isGetLen {
//		data := s.Sc.GetStateCache
//		if data != nil && len(data) > 0 {
//			copy(s.Memory[ptr:ptr+int32(len(data))], data)
//		}
//		s.Sc.GetStateCache = nil
//		return protocol.ContractSdkSignalResultSuccess
//	}
//
//	// get len
//	data, err := rows.Data()
//	if err != nil {
//		s.recordMsg("ctx query next data error, " + err.Error())
//		return protocol.ContractSdkSignalResultFail
//	}
//
//	ec := serialize.NewEasyCodecWithMap(data)
//	bytes := ec.Marshal()
//	copy(s.Memory[ptr:ptr+4], IntToBytes(int32(len(bytes))))
//	s.Sc.GetStateCache = bytes
//
//	return protocol.ContractSdkSignalResultSuccess
//}

// RSClose close sql statement
func (s *WaciInstance) RSClose() int32 {
	err := wasi.RSClose(s.RequestBody, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteUpdate execute update and insert sql, allow single row change
// as: update table set name = 'Tom' where uniqueKey='xxx'
func (s *WaciInstance) ExecuteUpdate() int32 {
	err := wasi.ExecuteUpdate(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.ChainId)
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
	err := wasi.ExecuteDDL(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
