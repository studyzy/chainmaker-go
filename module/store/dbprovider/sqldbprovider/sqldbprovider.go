/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqldbprovider

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm/logger"

	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//
//type SqlDBProvider struct {
//	dbs map[string]*SqlDBHandle
//	log protocol.Logger
//}
//
//func NewSqlDBProvider(log protocol.Logger) *SqlDBProvider {
//	return &SqlDBProvider{dbs: make(map[string]*SqlDBHandle, 1), log: log}
//}
//func (p *SqlDBProvider) GetDBHandle(chainId string, conf *localconf.SqlDbConfig) protocol.SqlDBHandle {
//	h, exist := p.dbs[chainId]
//	if exist {
//		return h
//	}
//	h = NewSqlDBHandle(chainId, conf, p.log)
//	p.dbs[chainId] = h
//	return h
//}
//
//// Close closes database
//func (p *SqlDBProvider) Close() error {
//	for _, h := range p.dbs {
//		h.Close()
//	}
//	return nil
//}

// Porvider encapsulate the gorm.DB that providers mysql handles
type SqlDBHandle struct {
	sync.Mutex
	contextDbName string
	db            *gorm.DB
	dbType        types.EngineType
	dbTxCache     map[string]*SqlDBTx
	log           protocol.Logger
}

// GetDBHandle returns a DBHandle for given dbname
func (p *SqlDBHandle) GetDBHandle(dbName string) protocol.DBHandle {
	p.Lock()
	defer p.Unlock()

	return p
}
func ParseSqlDbType(str string) (types.EngineType, error) {
	switch str {
	case "mysql":
		return types.MySQL, nil
	case "sqlite":
		return types.Sqlite, nil
	default:
		return types.UnknownDb, errors.New("uknow sql db type:" + str)
	}
}
func replaceMySqlDsn(dsn string, dbName string) string {
	dsnPattern := regexp.MustCompile(
		`^(?:(?P<user>.*?)(?::(?P<passwd>.*))?@)?` + // [user[:password]@]
			`(?:(?P<net>[^\(]*)(?:\((?P<addr>[^\)]*)\))?)?` + // [net[(addr)]]
			`\/(?P<dbname>.*?)` + // /dbname
			`(?:\?(?P<params>[^\?]*))?$`) // [?param1=value1&paramN=valueN]
	matches := dsnPattern.FindStringSubmatchIndex(dsn)
	if len(matches) < 12 {
		return dsn
	}
	start, end := matches[10], matches[11]
	newDsn := dsn[:start] + dbName + dsn[end:]
	return newDsn
}

// NewSqlDBProvider construct a new SqlDBHandle
func NewSqlDBHandle(dbName string, conf *localconf.SqlDbConfig, log protocol.Logger) *SqlDBHandle {
	provider := &SqlDBHandle{dbTxCache: make(map[string]*SqlDBTx), log: log}
	sqlType, err := ParseSqlDbType(conf.SqlDbType)
	if err != nil {
		panic(err.Error())
	}
	provider.dbType = sqlType
	if sqlType == types.MySQL {
		dsn := replaceMySqlDsn(conf.Dsn, dbName)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Error),
		})
		if err != nil {
			if strings.Contains(err.Error(), "Unknown database") {
				log.Infof("first time connect to a new database,create database %s", dbName)
				err = createDatabase(conf.Dsn, dbName)
				if err != nil {
					panic(fmt.Sprintf("failed to open mysql[%s] and create database %s, %s", dsn, dbName, err))
				}
				db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
					Logger: logger.Default.LogMode(logger.Error),
				})
				if err != nil {
					panic(fmt.Sprintf("failed to open mysql:%s , %s", dsn, err))
				}
			} else {
				panic(fmt.Sprintf("failed to open mysql:%s , %s", dsn, err))
			}
		}
		log.Debug("open new gorm db connection for " + conf.SqlDbType + " dsn:" + dsn)
		provider.db = db
		provider.contextDbName = dbName //默认连接mysql数据库
	} else if sqlType == types.Sqlite {
		dbPath := conf.Dsn
		if !strings.Contains(dbPath, ":memory:") { //不是内存数据库模式，则需要在路径中包含chainId
			dbPath = filepath.Join(dbPath, dbName)
			createDirIfNotExist(dbPath)
			dbPath = filepath.Join(dbPath, "sqlite.db")
		}
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			panic(fmt.Sprintf("failed to open sqlite path:%s,get error:%s", dbPath, err))
		}
		provider.db = db
	} else {
		panic(fmt.Sprintf("unsupport db:%v", sqlType))
	}
	logLevel := logger.Error
	if conf.SqlLogMode != "" {
		switch strings.ToLower(conf.SqlLogMode) {
		case "error":
			logLevel = logger.Error
		case "info":
			logLevel = logger.Info
		case "warn":
			logLevel = logger.Warn
		default:
			logLevel = logger.Silent
		}
	}
	log.Debug("inject ChainMaker logger into gorm db logger.")
	provider.db.Logger = NewSqlLogger(log, logger.Config{
		LogLevel: logLevel,
	})
	return provider
}
func createDatabase(dsn string, dbName string) error {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return err
	}
	result := db.Exec("create database " + dbName)
	defer func() {
		d, _ := db.DB()
		d.Close()
	}()
	return result.Error
}

func createDirIfNotExist(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		// 创建文件夹
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *SqlDBHandle) CreateDatabaseIfNotExist(dbName string) error {
	p.Lock()
	defer p.Unlock()
	if p.contextDbName == dbName {
		return nil
	}
	if p.dbType == types.Sqlite {
		return nil
	}
	//尝试切换数据库
	res := p.db.Exec("use " + dbName)
	if res.Error != nil { //切换失败，没有这个数据库，则创建
		p.log.Debugf("try to run 'use %s' get an error, it means database not exist, create it!", dbName)
		tx := p.db.Exec("create database " + dbName)
		if tx.Error != nil {
			return tx.Error //创建失败
		}
		p.log.Debugf("create database %s", dbName)
		//创建成功，再次切换数据库
		res = p.db.Exec("use " + dbName)
		return res.Error
	}
	p.log.Debugf("use database %s", dbName)
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
	tx := p.db.Exec(sql, values...)
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
func (p *SqlDBHandle) QuerySingle(sql string, values ...interface{}) (protocol.SqlRow, error) {
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
	return NewSqlDBRow(db, rows, nil), nil
}

func (p *SqlDBHandle) QueryMulti(sql string, values ...interface{}) (protocol.SqlRows, error) {
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
	return NewSqlDBRows(db, rows, nil), nil
}

func (p *SqlDBHandle) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	p.Lock()
	defer p.Unlock()

	if _, has := p.dbTxCache[txName]; has {
		return nil, errors.New("transaction already exist, please use GetDbTransaction to get it or commit/rollback it")
	}
	tx := p.db.Begin()
	sqltx := &SqlDBTx{db: tx, dbType: p.dbType, name: txName, logger: p.log}
	p.dbTxCache[txName] = sqltx
	p.contextDbName = ""

	p.log.Debugf("start new db transaction[%s]", txName)
	return sqltx, nil
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
	p.log.Debugf("commit db transaction[%s]", txName)
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
	p.log.Debugf("rollback db transaction[%s]", txName)
	return nil
}
func (p *SqlDBHandle) Close() error {
	p.Lock()
	defer p.Unlock()
	if len(p.dbTxCache) > 0 {
		txNames := ""
		for name := range p.dbTxCache {
			txNames += name + ";"
		}
		p.log.Warnf("these db tx[%s] don't commit or rollback, close them.", txNames)
	}
	db, _ := p.db.DB()
	return db.Close()
}
