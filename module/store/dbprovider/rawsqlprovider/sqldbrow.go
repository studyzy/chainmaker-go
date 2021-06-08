/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

import (
	"database/sql"
)

type SqlDBRow struct {
	rows *sql.Rows
}

func NewSqlDBRow(row *sql.Rows) *SqlDBRow {
	return &SqlDBRow{
		rows: row,
	}
}
func (r *SqlDBRow) ScanColumns(dest ...interface{}) error {
	defer r.rows.Close()
	err := r.rows.Scan(dest...)
	if err != nil {
		return errRow
	}
	return nil
}

func (row *SqlDBRow) Data() (map[string]string, error) {
	defer row.rows.Close()
	return convertRows2Map(row.rows)
}
func (row *SqlDBRow) IsEmpty() bool {
	return false
}

type emptyRow struct {
}

func (r *emptyRow) ScanColumns(dest ...interface{}) error {
	return nil
}

//func (row *emptyRow) ScanObject(dest interface{}) error {
//	return nil
//}
func (row *emptyRow) Data() (map[string]string, error) {
	return make(map[string]string), nil
}
func (row *emptyRow) IsEmpty() bool {
	return true
}

type SqlDBRows struct {
	rows  *sql.Rows
	close func() error
}

func NewSqlDBRows(rows *sql.Rows, close func() error) *SqlDBRows {
	return &SqlDBRows{
		rows:  rows,
		close: close,
	}
}
func (r *SqlDBRows) Next() bool {
	return r.rows.Next()
}
func (r *SqlDBRows) Close() error {
	rClose := r.rows.Close()
	if rClose != nil {
		return rClose
	}
	if r.close != nil {
		return r.close()
	}
	return nil
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
		return nil, errRow
	}
	values := make([]sql.RawBytes, len(cols))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	err = rows.Scan(scanArgs...)
	if err != nil {
		return nil, errRow
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

//func (r *SqlDBRows) ScanObject(dest interface{}) error {
//	return r.rows.Scan(dest)
//}
