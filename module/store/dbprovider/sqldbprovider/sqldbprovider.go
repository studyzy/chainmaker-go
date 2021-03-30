/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
	"errors"
	"gorm.io/driver/sqlite"

	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

var defaultMaxIdleConns = 10
var defaultMaxOpenConns = 10
var defaultConnMaxLifeTime = 60

type SqlDBProvider struct {
	dbs map[string]*SqlDBHandle
}

func NewSqlDBProvider() *SqlDBProvider {
	return &SqlDBProvider{dbs: make(map[string]*SqlDBHandle, 1)}
}
func (p *SqlDBProvider) GetDBHandle(chainId string, conf *localconf.SqlDbConfig) protocol.SqlDBHandle {
	h, exist := p.dbs[chainId]
	if exist {
		return h
	}
	h = NewSqlDBHandle(chainId, conf)
	p.dbs[chainId] = h
	return h
}

// Close closes database
func (p *SqlDBProvider) Close() error {
	for _, h := range p.dbs {
		h.Close()
	}
	return nil
}

// Porvider encapsulate the gorm.DB that providers mysql handles
type SqlDBHandle struct {
	sync.Mutex
	contextDbName string
	db            *gorm.DB
	dbType        types.EngineType
	dbTxCache     map[string]*SqlDBTx
}

// GetDBHandle returns a DBHandle for given dbname
func (p *SqlDBHandle) GetDBHandle(dbName string) protocol.DBHandle {
	p.Lock()
	defer p.Unlock()

	return p
}
func parseSqlDbType(str string) (types.EngineType, error) {
	switch str {
	case "mysql":
		return types.MySQL, nil
	case "sqlite":
		return types.Sqlite, nil
	default:
		return types.UnknowDb, errors.New("uknow sql db type:" + str)
	}
}

// NewSqlDBProvider construct a new SqlDBHandle
func NewSqlDBHandle(dbName string, conf *localconf.SqlDbConfig) *SqlDBHandle {
	provider := &SqlDBHandle{dbTxCache: make(map[string]*SqlDBTx)}
	sqlType, err := parseSqlDbType(conf.SqlDbType)
	if err != nil {
		panic(err.Error())
	}
	provider.dbType = sqlType
	if sqlType == types.MySQL {
		dsn := conf.Dsn + "mysql?charset=utf8mb4&parseTime=True&loc=Local"
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(fmt.Sprintf("failed to open mysql:%s", err))
		}
		provider.db = db
		provider.contextDbName = "mysql" //默认连接mysql数据库
	} else if sqlType == types.Sqlite {
		db, err := gorm.Open(sqlite.Open(conf.Dsn), &gorm.Config{})
		if err != nil {
			panic(fmt.Sprintf("failed to open mysql:%s", err))
		}
		provider.db = db
	} else {
		panic(fmt.Sprintf("unsupport db:%v", sqlType))
	}
	return provider
}

// GetDB returns a new gorm.DB for given chainid and conf.
func (p *SqlDBHandle) GetDB() *gorm.DB {
	return p.db
	//p.Lock()
	//defer p.Unlock()
	//if p.db == nil {
	//	mysqlConf := conf.StorageConfig.MysqlConfig
	//	err := p.tryCreateDB(chainId, mysqlConf.Dsn)
	//	if err != nil {
	//		panic(fmt.Sprintf("failed to create mysql:%s", err))
	//	}
	//	//dsn := "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	//	dsn := mysqlConf.Dsn + chainId + "?charset=utf8mb4&parseTime=True&loc=Local"
	//	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	//	if err != nil {
	//		panic(fmt.Sprintf("failed to open mysql:%s", err))
	//	}
	//	sqlDB, err := db.DB()
	//	if err != nil {
	//		panic(fmt.Sprintf("failed to open mysql:%s", err))
	//	}
	//	maxIdleConns := mysqlConf.MaxIdleConns
	//	if maxIdleConns <= 0 {
	//		maxIdleConns = defaultMaxIdleConns
	//	}
	//	sqlDB.SetMaxIdleConns(mysqlConf.MaxIdleConns)
	//
	//	maxOpenConns := mysqlConf.MaxOpenConns
	//	if maxOpenConns <= 0 {
	//		maxOpenConns = defaultMaxOpenConns
	//	}
	//	sqlDB.SetMaxOpenConns(mysqlConf.MaxOpenConns)
	//
	//	connMaxLifeTime := mysqlConf.ConnMaxLifeTime
	//	if connMaxLifeTime <= 0 {
	//		connMaxLifeTime = defaultConnMaxLifeTime
	//	}
	//	sqlDB.SetConnMaxLifetime(time.Duration(mysqlConf.ConnMaxLifeTime) * time.Second)
	//	p.db = db
	//}
	//return p.db
}

// tryCreateDB try create mysql database if not exist
//func (p *SqlDBHandle) tryCreateDB(dbName string, dsn string) error {
//	db, err := sql.Open("mysql", dsn)
//	if err != nil {
//		return err
//	}
//	defer db.Close()
//	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + dbName)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//// CreateDB create mysql database for the given chainid and dsn
//func (p *SqlDBHandle) CreateDB(chainId string, dsn string) {
//	p.Lock()
//	defer p.Unlock()
//	if err := p.tryCreateDB(chainId, dsn); err != nil {
//		panic(fmt.Sprintf("failed to create mysql, err:%s", err))
//	}
//}
func (p *SqlDBHandle) ChangeContextDb(dbName string) error {
	if dbName == "" {
		return nil
	}
	if p.dbType == types.Sqlite || p.dbType == types.LevelDb { //不支持切换数据库
		return nil
	}
	res := p.db.Exec("use " + dbName)
	if res.Error != nil {
		return res.Error
	}
	p.contextDbName = dbName
	return nil
}
func (p *SqlDBHandle) CreateDatabaseIfNotExist(dbName string) error {
	p.Lock()
	defer p.Unlock()
	if p.dbType == types.Sqlite {
		return nil
	}
	//尝试切换数据库
	res := p.db.Exec("use " + dbName)
	if res.Error != nil { //切换失败，没有这个数据库，则创建
		tx := p.db.Exec("create database " + dbName)
		if tx.Error != nil {
			return tx.Error //创建失败
		}
		//创建成功，再次切换数据库
		res = p.db.Exec("use " + dbName)
		return res.Error
	}
	p.contextDbName = dbName
	return nil
}

func (p *SqlDBHandle) CreateTableIfNotExist(obj interface{}) error {
	p.Lock()
	defer p.Unlock()
	m := p.db.Migrator()
	if !m.HasTable(obj) {
		return m.CreateTable(obj)
	}
	return nil
}

//ExecSql 执行SQL语句
func (p *SqlDBHandle) ExecSql(sql string, values ...interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx := p.db.Exec(sql, values)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}

func (p *SqlDBHandle) Save(value interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	tx := p.db.Save(value)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return tx.RowsAffected, nil
}
func (p *SqlDBHandle) QuerySql(sql string, values ...interface{}) (protocol.SqlRow, error) {
	p.Lock()
	defer p.Unlock()
	db := p.db
	row := db.Raw(sql, values...)
	if row.Error != nil {
		return nil, row.Error
	}
	return NewSqlDBRow(row), nil
}

func (p *SqlDBHandle) QueryTableSql(sql string, values ...interface{}) (protocol.SqlRows, error) {
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
func (p *SqlDBHandle) BeginDbTransaction(txName string) protocol.SqlDBTransaction {
	p.Lock()
	defer p.Unlock()
	if tx, has := p.dbTxCache[txName]; has {
		return tx
	}
	tx := p.db.Begin()
	sqltx := &SqlDBTx{db: tx, dbType: p.dbType}
	p.dbTxCache[txName] = sqltx
	return sqltx
}
func (p *SqlDBHandle) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	p.Lock()
	defer p.Unlock()
	return p.getDbTransaction(txName)
}
func (p *SqlDBHandle) getDbTransaction(txName string) (*SqlDBTx, error) {
	tx, has := p.dbTxCache[txName]
	if !has {
		return nil, errors.New("transaction not found or closed")
	}
	return tx, nil
}
func (p *SqlDBHandle) CommitDbTransaction(txName string) error {
	p.Lock()
	defer p.Unlock()
	tx, err := p.getDbTransaction(txName)
	if err != nil {
		return err
	}
	tx.Commit()
	delete(p.dbTxCache, txName)
	return nil
}
func (p *SqlDBHandle) RollbackDbTransaction(txName string) error {
	p.Lock()
	defer p.Unlock()
	tx, err := p.getDbTransaction(txName)
	if err != nil {
		return err
	}
	tx.Rollback()
	delete(p.dbTxCache, txName)
	return nil
}
func (p *SqlDBHandle) Close() error {
	p.Lock()
	defer p.Unlock()
	db, _ := p.db.DB()
	return db.Close()
}
