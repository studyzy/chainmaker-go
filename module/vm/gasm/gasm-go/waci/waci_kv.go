/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package waci

import (
	"chainmaker.org/chainmaker/protocol"
)

// GetStateLen get state length from chain
func (s *WaciInstance) GetStateLen() int32 {
	return s.getStateCore(true)
}

// GetStateLen get state from chain
func (s *WaciInstance) GetState() int32 {
	return s.getStateCore(false)
}

func (s *WaciInstance) getStateCore(isLen bool) int32 {
	data, err := wacsi.GetState(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory, s.GetStateCache, isLen)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess

}

// PutState put state to chain
func (s *WaciInstance) PutState() int32 {
	err := wacsi.PutState(s.RequestBody, s.ContractId.Name, s.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// DeleteState delete state from chain
func (s *WaciInstance) DeleteState() int32 {
	err := wacsi.DeleteState(s.RequestBody, s.ContractId.Name, s.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIterator Select kv statement
func (s *WaciInstance) KvIterator() int32 {
	err := wacsi.KvIterator(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

func (s *WaciInstance) KvPreIterator() int32 {
	err := wacsi.KvPreIterator(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

//KvIteratorHasNext to determine whether db has next statement
func (s *WaciInstance) KvIteratorHasNext() int32 {
	err := wacsi.KvIteratorHasNext(s.RequestBody, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

func (s *WaciInstance) KvIteratorNextLen() int32 {
	return s.kvIteratorNextCore(true)
}

//KvIteratorNext to get kv statement
func (s *WaciInstance) KvIteratorNext() int32 {
	return s.kvIteratorNextCore(false)
}

func (s *WaciInstance) kvIteratorNextCore(isLen bool) int32 {
	data, err := wacsi.KvIteratorNext(s.RequestBody, s.TxSimContext, s.Vm.Memory, s.GetStateCache, s.ContractId.Name, isLen)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// KvIteratorClose Close kv statement
func (s *WaciInstance) KvIteratorClose() int32 {
	err := wacsi.KvIteratorClose(s.RequestBody, s.ContractId.Name, s.TxSimContext, s.Vm.Memory)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
