/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"chainmaker.org/chainmaker-go/protocol"
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
	data, err := wacsi.GetState(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

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
	err := wacsi.DeleteState(s.RequestBody, s.Sc.ContractId.ContractName, s.Sc.TxSimContext)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}
