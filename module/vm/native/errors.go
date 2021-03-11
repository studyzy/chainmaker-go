/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import "errors"

var (
	ErrContractIdIsNil  = errors.New("the contractId is empty")
	ErrContractNotFound = errors.New("the contractName is not exist")
	ErrTxTypeNotSupport = errors.New("the txType does not support")
	ErrMethodNotFound   = errors.New("the method does not found")
	ErrParamsEmpty      = errors.New("the params is empty")
	ErrContractName     = errors.New("the contractName is error")
	ErrOutOfRange       = errors.New("out of range")
	ErrParams           = errors.New("params is error")
	ErrSequence         = errors.New("sequence is error")
	ErrUnmarshalFailed  = errors.New("unmarshal is error")
)
