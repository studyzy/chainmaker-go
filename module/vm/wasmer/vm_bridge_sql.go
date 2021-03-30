/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/protocol"
	"fmt"
)

// ExecuteQueryMulti execute query sql, return result set index
func (s *sdkRequestCtx) ExecuteQueryMulti() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// TODO get query sql result set index
	{
		index := 12312
		fmt.Println("index:", index, "ptr:", ptr, "sql:", sql)

		copy(s.Memory[ptr:ptr+4], IntToBytes(int32(index)))
	}
	return protocol.ContractSdkSignalResultSuccess
}

// QueryIteratorHasNext 1 is has next row, 0 is no next row
func (s *sdkRequestCtx) QueryIteratorHasNext() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	rsIndex := rsIndexI.(int32)
	valuePtr := valuePtrI.(int32)

	// TODO get query sql result set index
	{
		index := 12312
		fmt.Println("rsIndex:", rsIndex, "valuePtr:", valuePtr, "sql:", valuePtr)

		copy(s.Memory[valuePtr:valuePtr+4], IntToBytes(int32(index)))
	}
	return protocol.ContractSdkSignalResultSuccess
}

// QueryIteratorNextLen get result set length from chain
func (s *sdkRequestCtx) QueryIteratorNextLen() int32 {
	return s.queryIteratorNextCore(true)
}

// QueryIteratorNextLen get one row from result set
func (s *sdkRequestCtx) QueryIteratorNext() int32 {
	return s.queryIteratorNextCore(true)
}

func (s *sdkRequestCtx) queryIteratorNextCore(isGetLen bool) int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	rsIndex := rsIndexI.(int32)
	valuePtr := valuePtrI.(int32)

	if isGetLen {
		// TODO get next row
		length := 100
		value := make([]byte, 0)
		fmt.Println("rsIndex", rsIndex, "valuePtr", valuePtr)
		//contractName := s.Sc.ContractId.ContractName
		copy(s.Memory[valuePtr:valuePtr+4], IntToBytes(int32(length)))
		s.Sc.GetStateCache = value
	} else {
		len := int32(len(s.Sc.GetStateCache))
		if len != 0 {
			copy(s.Memory[valuePtr:valuePtr+len], s.Sc.GetStateCache)
			s.Sc.GetStateCache = nil
		}
	}
	return protocol.ContractSdkSignalResultSuccess
}

// QueryIteratorClose close sql statement
func (s *sdkRequestCtx) QueryIteratorClose() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	rsIndex := rsIndexI.(int32)
	valuePtr := valuePtrI.(int32)

	// TODO get query sql result set index
	{
		index := 1 // 1 success, 0 error
		fmt.Println("rsIndex:", rsIndex, "valuePtr:", valuePtr, "sql:", valuePtr)

		copy(s.Memory[valuePtr:valuePtr+4], IntToBytes(int32(index)))
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteUpdateSql execute update and insert sql, allow single row change
// as: update table set name = 'Tom' where uniqueKey='xxx'
func (s *sdkRequestCtx) ExecuteUpdateSql() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// TODO get query sql result set index
	{
		affectedCount := 1 //
		fmt.Println("affectedCount:", affectedCount, "ptr:", ptr, "sql:", sql)

		copy(s.Memory[ptr:ptr+4], IntToBytes(int32(affectedCount)))
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteDDLSql execute DDL sql, for init_contract or upgrade method. allow table create/alter/drop/truncate
//
// allow:     [CREATE TABLE tableName] [ALTER TABLE tableName]
//            [DROP TABLE tableName]   [TRUNCATE TABLE tableName]
//
// not allow: [CREATE DATABASE dbName] [CREATE TABLE dbName.tableName] [ALTER TABLE dbName.tableName]
//			  [DROP DATABASE dbName]   [DROP TABLE dbName.tableName]   [TRUNCATE TABLE dbName.tableName]
//
// You must have a primary key to create a table
func (s *sdkRequestCtx) ExecuteDDLSql() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// TODO get query sql result set index
	{
		affectedCount := 1 //
		fmt.Println("affectedCount:", affectedCount, "ptr:", ptr, "sql:", sql)

		copy(s.Memory[ptr:ptr+4], IntToBytes(int32(affectedCount)))
	}
	return protocol.ContractSdkSignalResultSuccess
}
