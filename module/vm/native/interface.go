/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package native

import "chainmaker.org/chainmaker/protocol"

type ContractFunc func(context protocol.TxSimContext, params map[string]string) ([]byte, error)

// Contract define native Contract interface
type Contract interface {
	getMethod(methodName string) ContractFunc
}
