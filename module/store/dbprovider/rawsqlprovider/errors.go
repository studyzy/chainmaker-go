/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

import "errors"

var SQL_ERROR = errors.New("sql error")
var SQL_QUERY_ERROR = errors.New("sql query error")
var TRANSACTION_ERROR = errors.New("database transaction error")
var CONNECTION_ERROR = errors.New("database connect error")
var DATABASE_ERROR = errors.New("database operation error")
var TABLE_ERROR = errors.New("table operation error")
var ROW_ERROR = errors.New("table row query error")
var IO_ERROR = errors.New("database I/O error")
var TX_NOT_FOUND_ERROR = errors.New("transaction not found or closed")
