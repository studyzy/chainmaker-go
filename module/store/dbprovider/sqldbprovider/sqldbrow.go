/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package sqldbprovider

import (
	"database/sql"
	"gorm.io/gorm"
)

type SqlDBRow struct {
	db *gorm.DB
}

func NewSqlDBRow(db *gorm.DB) *SqlDBRow {
	return &SqlDBRow{
		db: db,
	}
}
func (r *SqlDBRow) ScanColumns(dest ...interface{}) error {
	row := r.db.Row()
	return row.Scan(dest...)
}
func (row *SqlDBRow) ScanObject(dest interface{}) error {
	row.db.Scan(dest)
	if row.db.Error != nil {
		return row.db.Error
	}
	return nil
}

type SqlDBRows struct {
	db   *gorm.DB
	rows *sql.Rows
}

func NewSqlDBRows(db *gorm.DB, rows *sql.Rows) *SqlDBRows {
	return &SqlDBRows{
		db:   db,
		rows: rows,
	}
}
func (r *SqlDBRows) Next() bool {
	return r.rows.Next()
}
func (r *SqlDBRows) Close() error {
	return r.rows.Close()
}
func (r *SqlDBRows) ScanColumns(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}
func (r *SqlDBRows) ScanObject(dest interface{}) error {
	return r.db.ScanRows(r.rows, dest)
}
