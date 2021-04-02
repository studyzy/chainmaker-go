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
	db   *gorm.DB
	rows *sql.Rows
}

func NewSqlDBRow(db *gorm.DB, rows *sql.Rows) *SqlDBRow {
	return &SqlDBRow{
		db:   db,
		rows: rows,
	}
}
func (r *SqlDBRow) ScanColumns(dest ...interface{}) error {
	defer r.rows.Close()
	return r.rows.Scan(dest...)

}
func (row *SqlDBRow) ScanObject(dest interface{}) error {
	defer row.rows.Close()
	return row.db.ScanRows(row.rows, dest)
}
func (row *SqlDBRow) Data() (map[string]string, error) {
	defer row.rows.Close()
	return convertRows2Map(row.rows)
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
func (r *SqlDBRows) Data() (map[string]string, error) {
	return convertRows2Map(r.rows)
}

func convertRows2Map(rows *sql.Rows) (map[string]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]sql.RawBytes, len(cols))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	err = rows.Scan(scanArgs...)
	if err != nil {
		return nil, err
	}
	var value string
	resultC := map[string]string{}
	for i, col := range values {
		if col == nil {
			value = ""
		} else {
			value = string(col)
		}
		resultC[cols[i]] = value
	}
	return resultC, nil
}
func (r *SqlDBRows) ScanObject(dest interface{}) error {
	return r.db.ScanRows(r.rows, dest)
}
