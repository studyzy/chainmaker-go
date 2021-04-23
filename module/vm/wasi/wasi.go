/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0

Wacsi WebAssembly chainmaker system interface
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
	"regexp"
	"sync/atomic"
)

var ErrorNotManageContract = fmt.Errorf("method not init_contract or upgrade")

// Wacsi WebAssembly chainmaker system interface
type Wacsi interface {
	PutState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error
	GetState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error)
	DeleteState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error
	CallContract(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte, gasUsed uint64) ([]byte, error, uint64)
	SuccessResult(contractResult *common.ContractResult, data []byte) int32
	ErrorResult(contractResult *common.ContractResult, data []byte) int32

	ExecuteQuery(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error
	ExecuteQueryOne(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error)
	ExecuteUpdate(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, chainId string) error
	ExecuteDDL(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, method string) error

	RSHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error
	RSNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error)
	RSClose(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error
}

type WacsiImpl struct {
	verifySql *types.StandardSqlVerify
	rowIndex  int32
}

func NewWacsi() Wacsi {
	return &WacsiImpl{
		verifySql: &types.StandardSqlVerify{},
		rowIndex:  0,
	}
}

func (*WacsiImpl) PutState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, _ := ec.GetString("key")
	field, _ := ec.GetString("field")
	value, _ := ec.GetBytes("value")
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return err
	}
	err := txSimContext.Put(contractName, protocol.GetKeyStr(key, field), value)
	return err
}

func (*WacsiImpl) GetState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, _ := ec.GetString("key")
	field, _ := ec.GetString("field")
	valuePtr, _ := ec.GetInt32("value_ptr")
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return nil, err
	}

	if data == nil {
		value, err := txSimContext.Get(contractName, protocol.GetKeyStr(key, field))
		if err != nil {
			msg := fmt.Errorf("method getStateCore get fail. key=%s, field=%s, error:%s", key, field, err.Error())
			return nil, msg
		}
		if value == nil {
			value = make([]byte, 0)
		}
		copy(memory[valuePtr:valuePtr+4], utils.IntToBytes(int32(len(value))))
		return value, nil
	} else {
		len := int32(len(data))
		if len != 0 {
			copy(memory[valuePtr:valuePtr+len], data)
		}
	}
	return nil, nil
}

func (*WacsiImpl) DeleteState(requestBody []byte, contractName string, txSimContext protocol.TxSimContext) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	key, _ := ec.GetString("key")
	field, _ := ec.GetString("field")
	if err := protocol.CheckKeyFieldStr(key, field); err != nil {
		return err
	}

	err := txSimContext.Del(contractName, protocol.GetKeyStr(key, field))
	if err != nil {
		return err
	}
	return nil
}
func (*WacsiImpl) CallContract(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte, gasUsed uint64) ([]byte, error, uint64) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	valuePtr, _ := ec.GetInt32("value_ptr")
	contractName, _ := ec.GetString("contract_name")
	method, _ := ec.GetString("method")
	param, _ := ec.GetBytes("param")

	ecData := serialize.NewEasyCodecWithBytes(param)
	paramItem := ecData.GetItems()

	if data != nil { // get value from cache
		result := txSimContext.GetCurrentResult()
		copy(memory[valuePtr:valuePtr+int32(len(result))], result)
		return nil, nil, gasUsed
	}

	// check param
	if len(contractName) == 0 {
		return nil, fmt.Errorf("CallContract contractName is null"), gasUsed
	}
	if len(method) == 0 {
		return nil, fmt.Errorf("CallContract method is null"), gasUsed
	}
	if len(paramItem) > protocol.ParametersKeyMaxCount {
		return nil, fmt.Errorf("expect less than %d parameters, but get %d", protocol.ParametersKeyMaxCount, len(paramItem)), gasUsed
	}
	for _, item := range paramItem {
		if len(item.Key) > protocol.DefaultStateLen {
			return nil, fmt.Errorf("CallContract param expect Key length less than %d, but get %d", protocol.DefaultStateLen, len(item.Key)), gasUsed
		}
		match, err := regexp.MatchString(protocol.DefaultStateRegex, item.Key)
		if err != nil || !match {
			return nil, fmt.Errorf("CallContract param expect Key no special characters, but get %s. letter, number, dot and underline are allowed", item.Key), gasUsed
		}
		if len(item.Value.(string)) > protocol.ParametersValueMaxLength {
			return nil, fmt.Errorf("expect Value length less than %d, but get %d", protocol.ParametersValueMaxLength, len(item.Value.(string))), gasUsed
		}
	}
	if err := protocol.CheckKeyFieldStr(contractName, method); err != nil {
		return nil, err, gasUsed
	}

	// call contract
	gasUsed += protocol.CallContractGasOnce
	paramMap := ecData.ToMap()
	result, code := txSimContext.CallContract(&common.ContractId{ContractName: contractName}, method, nil, paramMap, gasUsed, common.TxType_INVOKE_USER_CONTRACT)
	gasUsed += uint64(result.GasUsed)
	if code != common.TxStatusCode_SUCCESS {
		return nil, fmt.Errorf("CallContract %s, , msg: %s", code.String(), result.Message), gasUsed
	}
	// set value length to memory
	l := utils.IntToBytes(int32(len(result.Result)))
	copy(memory[valuePtr:valuePtr+4], l)
	return result.Result, nil, gasUsed
}

func (*WacsiImpl) SuccessResult(contractResult *common.ContractResult, data []byte) int32 {
	if contractResult.Code == common.ContractResultCode_FAIL {
		return protocol.ContractSdkSignalResultFail
	}
	contractResult.Code = common.ContractResultCode_OK
	contractResult.Result = data
	return protocol.ContractSdkSignalResultSuccess
}

func (*WacsiImpl) ErrorResult(contractResult *common.ContractResult, data []byte) int32 {
	contractResult.Code = common.ContractResultCode_FAIL
	contractResult.Message += string(data)
	return protocol.ContractSdkSignalResultSuccess
}

func (w *WacsiImpl) ExecuteQuery(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDQLSql(sql); err != nil {
		return fmt.Errorf("verify query sql error, %s", err.Error())
	}

	// execute query
	rows, err := txSimContext.GetBlockchainStore().QueryMulti(contractName, sql)
	if err != nil {
		return fmt.Errorf("ctx query error, %s", err.Error())
	}

	index := atomic.AddInt32(&w.rowIndex, 1)
	txSimContext.SetStateSqlHandle(index, rows)
	copy(memory[ptr:ptr+4], utils.IntToBytes(index))
	return nil
}

func (w *WacsiImpl) ExecuteQueryOne(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDQLSql(sql); err != nil {
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

func (*WacsiImpl) RSHasNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, _ := ec.GetInt32("rs_index")
	valuePtr, _ := ec.GetInt32("value_ptr")

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

func (*WacsiImpl) RSNext(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte, data []byte) ([]byte, error) {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, _ := ec.GetInt32("rs_index")
	ptr, _ := ec.GetInt32("value_ptr")

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

func (*WacsiImpl) RSClose(requestBody []byte, txSimContext protocol.TxSimContext, memory []byte) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	rsIndex, _ := ec.GetInt32("rs_index")
	valuePtr, _ := ec.GetInt32("value_ptr")

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

func (w *WacsiImpl) ExecuteUpdate(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, chainId string) error {
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDMLSql(sql); err != nil {
		return fmt.Errorf("verify update sql error, [%s]", err.Error())
	}

	txKey := common.GetTxKewWith(txSimContext.GetBlockProposer(), txSimContext.GetBlockHeight())
	transaction, err := txSimContext.GetBlockchainStore().GetDbTransaction(txKey)
	if err != nil {
		return fmt.Errorf("ctx get db transaction error, [%s]", err.Error())
	}

	// execute
	changeCurrentDB(chainId, contractName, transaction)
	affectedCount, err := transaction.ExecSql(sql)
	if err != nil {
		return fmt.Errorf("ctx execute update sql error, [%s], sql[%s]", err.Error(), sql)
	}
	copy(memory[ptr:ptr+4], utils.IntToBytes(int32(affectedCount)))
	return nil
}

func (w *WacsiImpl) ExecuteDDL(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte, method string) error {
	if !w.isManageContract(method) {
		return ErrorNotManageContract
	}
	ec := serialize.NewEasyCodecWithBytes(requestBody)
	sql, _ := ec.GetString("sql")
	ptr, _ := ec.GetInt32("value_ptr")

	// verify
	if err := w.verifySql.VerifyDDLSql(sql); err != nil {
		return fmt.Errorf("verify ddl sql error,  [%s], sql[%s]", err.Error(), sql)
	}

	// execute
	if err := txSimContext.GetBlockchainStore().ExecDdlSql(contractName, sql); err != nil {
		return fmt.Errorf("ctx ExecDdlSql error, %s, sql[%s]", err.Error(), sql)
	}
	copy(memory[ptr:ptr+4], utils.IntToBytes(0))
	return nil
}
func (w *WacsiImpl) isManageContract(method string) bool {
	return method == protocol.ContractInitMethod || method == protocol.ContractUpgradeMethod
}

func changeCurrentDB(chainId string, contractName string, transaction protocol.SqlDBTransaction) {
	dbName := statesqldb.GetContractDbName(chainId, contractName)
	//currentDbName := getCurrentDb(chainId)
	//if contractName != "" && dbName != currentDbName {
	transaction.ChangeContextDb(dbName)
	//setCurrentDb(chainId, dbName)
	//}
}
