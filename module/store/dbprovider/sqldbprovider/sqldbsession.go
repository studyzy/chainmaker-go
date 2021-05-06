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

type SqlDBSession struct {
	sync.Mutex
	//name   string
	dbType types.EngineType
	db     *gorm.DB
	logger protocol.Logger
}

func (p *SqlDBSession) CreateTableIfNotExist(obj interface{}) error {
	p.Lock()
	defer p.Unlock()
	m := p.db.Migrator()
	if !m.HasTable(obj) {
		return m.CreateTable(obj)
	}
	return nil
}
func (p *SqlDBSession) ChangeContextDb(dbName string) error {
	if dbName == "" {
		return nil
	}
	p.Lock()
	defer p.Unlock()
	if p.dbType == types.Sqlite || p.dbType == types.LevelDb { //不支持切换数据库
		return nil
	}
	res := p.db.Exec("use " + dbName)
	if res.Error != nil {
		return res.Error
	}
	p.db = res
	return nil
}
func (p *SqlDBSession) Save(value interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx := p.db.Save(value)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
func (p *SqlDBSession) ExecSql(sql string, values ...interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx := p.db.Exec(sql, values)
	p.logger.Debugf("db session exec sql[%s],result:%v", sql, tx.Error)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
func (p *SqlDBSession) QuerySingle(sql string, values ...interface{}) (protocol.SqlRow, error) {
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
func (p *SqlDBSession) QueryMulti(sql string, values ...interface{}) (protocol.SqlRows, error) {
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
