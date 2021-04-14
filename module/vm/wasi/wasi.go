package wasi

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/protocol"
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

		var data map[string]string
		if row.IsEmpty() {
			data = make(map[string]string, 0)
		} else {
			data, err = row.Data()
			if err != nil {
				return nil, fmt.Errorf("ctx query get data to map error, %s", err.Error())
			}
		}
		ec := serialize.NewEasyCodecWithMap(data)
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

func RSHasNext(requestBody []byte, contractName string, txSimContext protocol.TxSimContext, memory []byte) error {
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
