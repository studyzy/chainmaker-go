/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

// RequestHeader receive sdk common data: json byte array
type RequestHeader struct {
	CtxPtr  int32  `json:"ctx_ptr"`
	Method  string `json:"method"`
	Version string `json:"version"`
}

// GetStateRequest receive sdk get state request data: json byte array
type GetStateRequest struct {
	ValuePtr int32  `json:"value_ptr"` // GetStateLen mean valueLenPtr, GetState mean valuePtr
	Key      string `json:"key"`
	Field    string `json:"field"`
}

// PutStateRequest receive sdk get state request data: json byte array
type PutStateRequest struct {
	Key   string `json:"key"`
	Field string `json:"field"`
	Value []byte `json:"value"`
}

// PutStateRequest receive sdk delete state request data: json byte array
type DeleteStateRequest struct {
	Key   string `json:"key"`
	Field string `json:"field"`
}

// PutStateRequest receive sdk call contract request data: json byte array
type CallContractRequest struct {
	ValuePtr     int32             `json:"value_ptr"` // CallContractLen mean valueLenPtr, CallContract mean valuePtr
	ContractName string            `json:"contract_name"`
	Method       string            `json:"method"`
	Param        map[string]string `json:"param"`
}
