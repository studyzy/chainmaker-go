/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package contractmgr

import "errors"

var (
	errContractExist          = errors.New("contract exist")
	errContractNotExist       = errors.New("contract not exist")
	errContractVersionExist   = errors.New("contract version exist")
	errContractStatusInvalid  = errors.New("contract status invalid")
	errInvalidContractName    = errors.New("invalid contract name")
	errInvalidEvmContractName = errors.New("invalid EVM contract name")
)
