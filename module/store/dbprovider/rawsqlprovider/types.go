/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

type TableDDLGenerator interface {
	GetCreateTableSql(dbType string) string
	GetTableName() string
}
type TableDMLGenerator interface {
	GetInsertSql() (string, []interface{})
	GetUpdateSql() (string, []interface{})
}
