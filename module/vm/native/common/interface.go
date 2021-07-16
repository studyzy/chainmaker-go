/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package common

import "chainmaker.org/chainmaker/protocol"

type ContractFunc func(context protocol.TxSimContext, params map[string][]byte) ([]byte, error)

// Contract define native Contract interface
type Contract interface {
	GetMethod(methodName string) ContractFunc
}
