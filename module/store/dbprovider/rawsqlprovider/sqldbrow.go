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
	rows  *sql.Rows
	close func() error
}

func NewSqlDBRow(row *sql.Rows, close func() error) *SqlDBRow {
	return &SqlDBRow{
		rows:  row,
		close: close,
	}
}
func (r *SqlDBRow) ScanColumns(dest ...interface{}) error {
	if r.close != nil {
		defer r.close()
	}
	defer r.rows.Close()
	return r.rows.Scan(dest...)

}

//func (row *SqlDBRow) ScanObject(dest interface{}) error {
//	if row.close != nil {
//		defer row.close()
//	}
//	return row.db.ScanRows(row.rows, dest)
//}
func (row *SqlDBRow) Data() (map[string]string, error) {
	if row.close != nil {
		defer row.close()
	}
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
	db    *sql.DB
	rows  *sql.Rows
	close func() error
}

func NewSqlDBRows(rows *sql.Rows, close func() error) *SqlDBRows {
	return &SqlDBRows{
		//db:    db,
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

//func (r *SqlDBRows) ScanObject(dest interface{}) error {
//	return r.rows.Scan(dest)
//}
