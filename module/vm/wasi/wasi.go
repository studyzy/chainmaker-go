package wasi

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
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
