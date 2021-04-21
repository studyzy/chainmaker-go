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
	name   string
	dbType types.EngineType
	db     *gorm.DB
	logger protocol.Logger
}

func (p *SqlDBTx) ChangeContextDb(dbName string) error {
	if dbName == "" {
		return nil
	}
	p.Lock()
	defer p.Unlock()
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
	p.logger.Debugf("db tx[%s] exec sql[%s],result:%v", p.name, sql, tx.Error)
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
	p.Lock()
	defer p.Unlock()
	result := p.db.Commit()
	if result.Error != nil {
		return result.Error
	}
	return nil
}
func (p *SqlDBTx) Rollback() error {
	p.Lock()
	defer p.Unlock()
	result := p.db.Rollback()
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (p *SqlDBTx) BeginDbSavePoint(spName string) error {
	p.Lock()
	defer p.Unlock()
	savePointName := getSavePointName(spName)
	db := p.db.SavePoint(savePointName)
	p.logger.Debugf("db tx[%s] new savepoint[%s],result:%s", p.name, savePointName, db.Error)
	return db.Error
}
func (p *SqlDBTx) RollbackDbSavePoint(spName string) error {
	p.Lock()
	defer p.Unlock()
	savePointName := getSavePointName(spName)
	db := p.db.RollbackTo(savePointName)
	p.logger.Debugf("db tx[%s] rollback savepoint[%s],result:%s", p.name, savePointName, db.Error)
	return db.Error
}
func getSavePointName(spName string) string {
	return "SP_" + spName
}
