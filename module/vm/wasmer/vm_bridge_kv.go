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

// GetStateLen get state length from chain
func (s *sdkRequestCtx) GetStateLen() int32 {
	return s.getStateCore(true)
}

// GetStateLen get state from chain
func (s *sdkRequestCtx) GetState() int32 {
	return s.getStateCore(false)
}

func (s *sdkRequestCtx) getStateCore(isGetLen bool) int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)

	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return s.recordMsg(err.Error())
	}

	if isGetLen {
		contractName := s.Sc.ContractId.ContractName
		value, err := s.Sc.TxSimContext.Get(contractName, protocol.GetKeyStr(key.(string), field.(string)))
		if err != nil {
			msg := fmt.Sprintf("method getStateCore get fail. key=%s, field=%s, error:%s", key.(string), field.(string), err.Error())
			return s.recordMsg(msg)
		}
		copy(s.Memory[valuePtr.(int32):valuePtr.(int32)+4], IntToBytes(int32(len(value))))
		s.Sc.GetStateCache = value
	} else {
		len := int32(len(s.Sc.GetStateCache))
		if len != 0 {
			copy(s.Memory[valuePtr.(int32):valuePtr.(int32)+len], s.Sc.GetStateCache)
			s.Sc.GetStateCache = nil
		}
	}
	return protocol.ContractSdkSignalResultSuccess
}

// PutState put state to chain
func (s *sdkRequestCtx) PutState() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
	value, _ := serialize.GetValueFromItems(req, "value", serialize.EasyKeyType_USER)
	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return s.recordMsg(err.Error())
	}
	contractName := s.Sc.ContractId.ContractName
	err := s.Sc.TxSimContext.Put(contractName, protocol.GetKeyStr(key.(string), field.(string)), value.([]byte))
	if err != nil {
		return s.recordMsg("method PutState put fail. " + err.Error())
	}
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState delete state from chain
func (s *sdkRequestCtx) DeleteState() int32 {
	req := serialize.EasyUnmarshal(s.RequestBody)
	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)

	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
		return s.recordMsg(err.Error())
	}

	contractName := s.Sc.ContractId.ContractName
	err := s.Sc.TxSimContext.Del(contractName, protocol.GetKeyStr(key.(string), field.(string)))
	if err != nil {
		return s.recordMsg(err.Error())
	}

	return protocol.ContractSdkSignalResultSuccess
}
