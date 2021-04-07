/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package sqldbprovider

import (
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
	"gorm.io/gorm"
	"sync"
)

type SqlDBTx struct {
	sync.Mutex
	dbType types.EngineType
	db     *gorm.DB
}

func (p *SqlDBTx) ChangeContextDb(dbName string) error {
	if dbName == "" {
		return nil
	}
	if p.dbType == types.Sqlite || p.dbType == types.LevelDb { //不支持切换数据库
		return nil
	}
	res := p.db.Exec("use " + dbName)
	return res.Error
}
func (p *SqlDBTx) Save(value interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx := p.db.Save(value)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
func (p *SqlDBTx) ExecSql(sql string, values ...interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx := p.db.Exec(sql, values)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
func (p *SqlDBTx) QuerySingle(sql string, values ...interface{}) (protocol.SqlRow, error) {
	p.Lock()
	defer p.Unlock()
	db := p.db
	row := db.Raw(sql, values...)
	if row.Error != nil {
		return nil, row.Error
	}
	rows, err := row.Rows()
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return &emptyRow{}, nil
	}
	return NewSqlDBRow(db, rows), nil
}
func (p *SqlDBTx) QueryMulti(sql string, values ...interface{}) (protocol.SqlRows, error) {
	p.Lock()
	defer p.Unlock()
	db := p.db
	row := db.Raw(sql, values...)
	if row.Error != nil {
		return nil, row.Error
	}
	rows, err := row.Rows()
	if err != nil {
		return nil, err
	}
	return NewSqlDBRows(db, rows), nil
}
func (p *SqlDBTx) Commit() error {
	result := p.db.Commit()
	if result.Error != nil {
		return result.Error
	}
	return nil
}
func (p *SqlDBTx) Rollback() error {
	result := p.db.Rollback()
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (p *SqlDBTx) BeginDbSavePoint(savePointName string) error {
	p.Lock()
	defer p.Unlock()
	p.db.SavePoint(savePointName)
	return nil
}
func (p *SqlDBTx) RollbackDbSavePoint(savePointName string) error {
	p.Lock()
	defer p.Unlock()
	p.db.RollbackTo(savePointName)
	return nil
}
