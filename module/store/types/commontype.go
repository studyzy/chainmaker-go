/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

type SavePoint struct {
	BlockHeight uint64 `gorm:"primarykey"`
}

func (b *SavePoint) GetCreateTableSql(dbType string) string {
	if dbType == "mysql" {
		return "CREATE TABLE `save_points` (`block_height` bigint unsigned AUTO_INCREMENT,PRIMARY KEY (`block_height`))"
	} else if dbType == "sqlite" {
		return "CREATE TABLE `save_points` (`block_height` integer,PRIMARY KEY (`block_height`))"
	}
	panic("Unsupported db type:" + string(dbType))
}
func (b *SavePoint) GetTableName() string {
	return "save_points"
}
func (b *SavePoint) GetInsertSql() (string, []interface{}) {
	return "INSERT INTO save_points values(?)", []interface{}{b.BlockHeight}
}
func (b *SavePoint) GetUpdateSql() (string, []interface{}) {
	return "UPDATE save_points set block_height=?", []interface{}{b.BlockHeight}
}
