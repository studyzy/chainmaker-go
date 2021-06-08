/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

import "errors"

var (
	errSql         = errors.New("sql error")
	errSqlQuery    = errors.New("sql query error")
	errTransaction = errors.New("database transaction error")
	errConnection  = errors.New("database connect error")
	errDatabase    = errors.New("database operation error")
	errTable       = errors.New("table operation error")
	errRow         = errors.New("table row query error")
	errIO          = errors.New("database I/O error")
	errTxNotFound  = errors.New("transaction not found or closed")
	errTypeConvert = errors.New("type convert error")
)
