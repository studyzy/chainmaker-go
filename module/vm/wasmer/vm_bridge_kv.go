/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/protocol"
)

// GetStateLen get state length from chain
func (s *WaciInstance) GetStateLen() int32 {
	data, err := wacsi.GetState(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess

}

// GetStateLen get state from chain
func (s *WaciInstance) GetState() int32 {
	return s.GetStateLen()
}

//func (s *WaciInstance) getStateCore(isGetLen bool) int32 {
//	req := serialize.EasyUnmarshal(s.RequestBody)
//	key, _ := serialize.GetValueFromItems(req, "key", serialize.EasyKeyType_USER)
//	field, _ := serialize.GetValueFromItems(req, "field", serialize.EasyKeyType_USER)
//	valuePtr, _ := serialize.GetValueFromItems(req, "value_ptr", serialize.EasyKeyType_USER)
//
//	if err := protocol.CheckKeyFieldStr(key.(string), field.(string)); err != nil {
//		return s.recordMsg(err.Error())
//	}
//
//	if isGetLen {
//		contractName := s.Sc.ContractId.ContractName
//		value, err := s.Sc.TxSimContext.Get(contractName, protocol.GetKeyStr(key.(string), field.(string)))
//		if err != nil {
//			msg := fmt.Sprintf("method getStateCore get fail. key=%s, field=%s, error:%s", key.(string), field.(string), err.Error())
//			return s.recordMsg(msg)
//		}
//		copy(s.Memory[valuePtr.(int32):valuePtr.(int32)+4], utils.IntToBytes(int32(len(value))))
//		s.Sc.GetStateCache = value
//	} else {
//		len := int32(len(s.Sc.GetStateCache))
//		if len != 0 {
//			copy(s.Memory[valuePtr.(int32):valuePtr.(int32)+len], s.Sc.GetStateCache)
//			s.Sc.GetStateCache = nil
//		}
//	}
//	return protocol.ContractSdkSignalResultSuccess
//}

// PutState put state to chain
func (s *WaciInstance) PutState() int32 {
	err := wacsi.PutState(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState delete state from chain
func (s *WaciInstance) DeleteState() int32 {
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
