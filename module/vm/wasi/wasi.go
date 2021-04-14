/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0

wasi: WebAssembly System Interface
*/
package wasi

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/statedb/statesqldb"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"sync/atomic"
)

func PutState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {
	req := serialize.EasyUnmarshal(requestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	value, _ := serialize.GetValueFromItems(req, "value", serialize.EasyKeyType_USER)
	if field == nil {
		field = ""
	}
	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return err
	}
	err := txSimContext.Put(contractName, protocol.GetKeyStr(key.(string), field.(string)), value.([]byte))
	if err != nil {
		return err
	}
	return nil
}

func GetState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error) {
	req := serialize.EasyUnmarshal(requestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	if field == nil {
		field = ""
	}

	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return nil, err
	}

	if data == nil {
		value, err := txSimContext.Get(contractName, protocol.GetKeyStr(key.(string), field.(string)))
		if err != nil {
			msg := fmt.Errorf("method getStateCore get fail. key=%s, field=%s, error:%s", key.(string), field.(string), err.Error())
			return nil, msg
		}
		if value == nil {
			value = make([]byte, 0)
		}
		copy(memory[valuePtr.(int32):valuePtr.(int32)+4], utils.IntToBytes(int32(len(value))))
		return value, nil
	} else {
		len := int32(len(data))
		if len != 0 {
			copy(memory[valuePtr.(int32):valuePtr.(int32)+len], data)
		}
	}
	return nil, nil
}

var verifySql = &types.StandardSqlVerify{}
var rowIndex int32 = 0

func ExecuteQuery(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error {
	req := serialize.EasyUnmarshal(requestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// verify
	if err := verifySql.VerifyDQLSql(sql); err != nil {
		return fmt.Errorf("verify query sql error, %s", err.Error())
	}

	// execute query
	rows, err := txSimContext.GetBlockchainStore().QueryMulti(contractName, sql)
	if err != nil {
		return fmt.Errorf("ctx query error, %s", err.Error())
	}

	index := atomic.AddInt32(&rowIndex, 1)
	txSimContext.SetStateSqlHandle(index, rows)
	copy(memory[ptr:ptr+4], utils.IntToBytes(index))
	return nil
}

func ExecuteQueryOne(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error) {
	req := serialize.EasyUnmarshal(requestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// verify
	if err := verifySql.VerifyDQLSql(sql); err != nil {
		return nil, fmt.Errorf("verify query one sql error, %s", err.Error())
	}

	// get len
	if data == nil {
		// execute
		row, err := txSimContext.GetBlockchainStore().QuerySingle(contractName, sql)
		if err != nil {
			return nil, fmt.Errorf("ctx query error, %s", err.Error())
		}

		var dataRow map[string]string
		if row.IsEmpty() {
			dataRow = make(map[string]string, 0)
		} else {
			dataRow, err = row.Data()
			if err != nil {
				return nil, fmt.Errorf("ctx query get data to map error, %s", err.Error())
			}
		}
		ec := serialize.NewEasyCodecWithMap(dataRow)
		rsBytes := ec.Marshal()
		copy(memory[ptr:ptr+4], utils.IntToBytes(int32(len(rsBytes))))
		return rsBytes, nil
	} else { // get data
		if data != nil && len(data) > 0 {
			copy(memory[ptr:ptr+int32(len(data))], data)
		}
		return nil, nil
	}
}

func RSHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	req := serialize.EasyUnmarshal(requestBody)
	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	rsIndex := rsIndexI.(int32)
	valuePtr := valuePtrI.(int32)

	// get
	rows, ok := txSimContext.GetStateSqlHandle(rsIndex)
	if !ok {
		return fmt.Errorf("ctx can not found rs_index[%d]", rsIndex)
	}
	var index int32 = 0
	if rows.Next() {
		index = 1
	}
	copy(memory[valuePtr:valuePtr+4], utils.IntToBytes(index))
	return nil
}

func RSNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error) {
	req := serialize.EasyUnmarshal(requestBody)
	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	rsIndex := rsIndexI.(int32)
	ptr := valuePtrI.(int32)

	// get handle
	rows, ok := txSimContext.GetStateSqlHandle(rsIndex)
	if !ok {
		return nil, fmt.Errorf("ctx can not found rs_index[%d]", rsIndex)
	}

	// get len
	if data == nil {
		var dataRow map[string]string
		var err error
		if rows == nil {
			dataRow = make(map[string]string, 0)
		} else {
			dataRow, err = rows.Data()
			if err != nil {
				return nil, fmt.Errorf("ctx query next data error, %s", err.Error())
			}
		}
		ec := serialize.NewEasyCodecWithMap(dataRow)
		rsBytes := ec.Marshal()
		copy(memory[ptr:ptr+4], utils.IntToBytes(int32(len(rsBytes))))
		return rsBytes, nil
	} else { // get data
		if len(data) > 0 {
			copy(memory[ptr:ptr+int32(len(data))], data)
		}
		return nil, nil
	}
}

func RSClose(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	req := serialize.EasyUnmarshal(requestBody)
	rsIndexI, _ := serialize.GetValueFromItems(req, "rs_index", serialize.EasyKeyType_USER)
	valuePtrI, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	rsIndex := rsIndexI.(int32)
	valuePtr := valuePtrI.(int32)

	// get
	rows, ok := txSimContext.GetStateSqlHandle(rsIndex)
	if !ok {
		return fmt.Errorf("ctx can not found rs_index[%d]", rsIndex)
	}
	var index int32 = 1
	if err := rows.Close(); err != nil {
		return fmt.Errorf("ctx close rows error, [%s]", err.Error())
	}
	copy(memory[valuePtr:valuePtr+4], utils.IntToBytes(index))
	return nil
}

func ExecuteUpdate(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, chainId string) error {
	req := serialize.EasyUnmarshal(requestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// verify
	if err := verifySql.VerifyDMLSql(sql); err != nil {
		return fmt.Errorf("verify update sql error, [%s]", err.Error())
	}

	txKey := common.GetTxKewWith(txSimContext.GetBlockProposer(), txSimContext.GetBlockHeight())
	transaction, err := txSimContext.GetBlockchainStore().GetDbTransaction(txKey)
	if err != nil {
		return fmt.Errorf("ctx get db transaction error, [%s]", err.Error())
	}

	// execute
	// todo 优化缓存currentDB  module map[chainId]?
	dbName := statesqldb.GetContractDbName(chainId, contractName)
	transaction.ChangeContextDb(dbName)
	affectedCount, err := transaction.ExecSql(sql)
	if err != nil {
		return fmt.Errorf("ctx execute update sql error, [%s], sql[%s]", err.Error(), sql)
	}
	copy(memory[ptr:ptr+4], utils.IntToBytes(int32(affectedCount)))
	return nil
}

func ExecuteDDL(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error {
	req := serialize.EasyUnmarshal(requestBody)
	sqlI, _ := serialize.GetValueFromItems(req, "sql", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
	sql := sqlI.(string)
	ptr := valuePtr.(int32)

	// verify
	if err := verifySql.VerifyDDLSql(sql); err != nil {
		return fmt.Errorf("verify ddl sql error,  [%s], sql[%s]", err.Error(), sql)
	}

	// execute
	if err := txSimContext.GetBlockchainStore().ExecDdlSql(contractName, sql); err != nil {
		return fmt.Errorf("ctx ExecDdlSql error, %s, sql[%s]", err.Error(), sql)
	}
	copy(memory[ptr:ptr+4], utils.IntToBytes(0))
	return nil
}
