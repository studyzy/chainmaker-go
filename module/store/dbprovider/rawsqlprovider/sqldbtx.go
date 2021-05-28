/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package rawsqlprovider

import (
	"database/sql"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
)

type SqlDBTx struct {
	sync.Mutex
	name      string
	dbType    types.EngineType
	db        *sql.Tx
	logger    protocol.Logger
	startTime time.Time
}

func NewSqlDBTx(name string, dbType types.EngineType, db *sql.Tx, logger protocol.Logger) *SqlDBTx {
	return &SqlDBTx{
		name:      name,
		dbType:    dbType,
		db:        db,
		logger:    logger,
		startTime: time.Now(),
	}
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
	sqlStr := "use " + dbName
	p.logger.Debug("Exec sql:", sqlStr)
	_, err := p.db.Exec(sqlStr)
	if err != nil {
		p.logger.Warnf("change context db fail, error: %s", err)
		return TRANSACTION_ERROR
	}
	return nil
}
func (p *SqlDBTx) Save(val interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	value, ok := val.(TableDMLGenerator)
	if !ok {
		p.logger.Errorf("%v not a TableDMLGenerator", val)
		return 0, TYPE_CONVERT_ERROR
	}
	update, args := value.GetUpdateSql()
	p.logger.Debug("Exec sql:", update, args)
	effect, err := p.db.Exec(update, args...)
	if err != nil {
		p.logger.Error(err)
		return 0, SQL_ERROR
	}
	rowCount, err := effect.RowsAffected()
	if err != nil {
		p.logger.Error(err)
		return 0, SQL_ERROR
	}
	if rowCount != 0 {
		return rowCount, nil
	}
	insert, args := value.GetInsertSql()
	p.logger.Debug("Exec sql:", insert, args)
	result, err := p.db.Exec(insert, args...)
	if err != nil {
		p.logger.Error(err)
		return 0, SQL_ERROR
	}
	rowCount, err = result.RowsAffected()
	if err != nil {
		p.logger.Error(err)
		return 0, SQL_ERROR
	}
	return rowCount, nil
}
func (p *SqlDBTx) ExecSql(sql string, values ...interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx, err := p.db.Exec(sql, values...)
	p.logger.Debugf("db tx[%s] exec sql[%s],result:%v", p.name, sql, err)
	if err != nil {
		p.logger.Error(err)
		return 0, SQL_ERROR
	}
	rowCount, err := tx.RowsAffected()
	if err != nil {
		p.logger.Error(err)
		return 0, SQL_ERROR
	}
	return rowCount, nil
}
func (p *SqlDBTx) QuerySingle(sql string, values ...interface{}) (protocol.SqlRow, error) {
	p.Lock()
	defer p.Unlock()
	db := p.db
	p.logger.Debug("Query sql:", sql, values)
	rows, err := db.Query(sql, values...)
	if err != nil {
		p.logger.Error(err)
		return nil, SQL_QUERY_ERROR
	}
	if !rows.Next() {
		return &emptyRow{}, nil
	}
	return NewSqlDBRow(rows), nil
}
func (p *SqlDBTx) QueryMulti(sql string, values ...interface{}) (protocol.SqlRows, error) {
	p.Lock()
	defer p.Unlock()
	p.logger.Debug("Query sql:", sql, values)
	rows, err := p.db.Query(sql, values...)
	if err != nil {
		p.logger.Error(err)
		return nil, SQL_QUERY_ERROR
	}
	return NewSqlDBRows(rows, nil), nil
}
func (p *SqlDBTx) Commit() error {
	p.Lock()
	defer p.Unlock()
	err := p.db.Commit()
	p.logger.Debugf("commit tx[%s], tx duration：%s", p.name, time.Since(p.startTime).String())
	if err != nil {
		p.logger.Error(err)
		return TRANSACTION_ERROR
	}
	return nil
}
func (p *SqlDBTx) Rollback() error {
	p.Lock()
	defer p.Unlock()
	err := p.db.Rollback()
	p.logger.Warnf("rollback tx[%s], tx duration：%s", p.name, time.Since(p.startTime).String())
	if err != nil {
		p.logger.Error(err)
		return TRANSACTION_ERROR
	}
	return nil
}

func (p *SqlDBTx) BeginDbSavePoint(spName string) error {
	p.Lock()
	defer p.Unlock()
	savePointName := getSavePointName(spName)
	_, err := p.db.Exec("SAVEPOINT " + savePointName)
	p.logger.Debugf("db tx[%s] new savepoint[%s],result:%s", p.name, savePointName, err)
	if err != nil {
		p.logger.Error(err)
		return TRANSACTION_ERROR
	}
	return nil
}
func (p *SqlDBTx) RollbackDbSavePoint(spName string) error {
	p.Lock()
	defer p.Unlock()
	savePointName := getSavePointName(spName)
	_, err := p.db.Exec("ROLLBACK TO SAVEPOINT " + savePointName)
	p.logger.Infof("db tx[%s] rollback savepoint[%s],result:%s", p.name, savePointName, err)
	if err != nil {
		p.logger.Error(err)
		return TRANSACTION_ERROR
	}
	return nil
}
func getSavePointName(spName string) string {
	return "SP_" + spName
}
