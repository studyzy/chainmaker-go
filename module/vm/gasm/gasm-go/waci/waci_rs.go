/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package waci

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/wasi"
)

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQuery() int32 {
	err := wasi.ExecuteQuery(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory)
	if err == nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOneLen() int32 {
	data, err := wasi.ExecuteQueryOne(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory, s.GetStateCache)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// ExecuteQuery execute query sql, return result set index
func (s *WaciInstance) ExecuteQueryOne() int32 {
	data, err := wasi.ExecuteQueryOne(s.RequestBody, s.ContractId.ContractName, s.TxSimContext, s.Vm.Memory, s.GetStateCache)
	s.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}