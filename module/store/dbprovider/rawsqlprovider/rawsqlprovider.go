/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rawsqlprovider

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/types"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var defaultMaxIdleConns = 10
var defaultMaxOpenConns = 10
var defaultConnMaxLifeTime = 60

type SqlDBHandle struct {
	sync.Mutex
	contextDbName string
	db            *sql.DB
	dbType        types.EngineType
	dbTxCache     map[string]*SqlDBTx
	log           protocol.Logger
}

func ParseSqlDbType(str string) (types.EngineType, error) {
	switch str {
	case "mysql":
		return types.MySQL, nil
	case "sqlite":
		return types.Sqlite, nil
	default:
		return types.UnknownDb, errors.New("unknown sql db type:" + str)
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
		log.Panic(err.Error())
	}
	provider.dbType = sqlType
	if sqlType == types.MySQL {
		dsn := replaceMySqlDsn(conf.Dsn, dbName)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Panic("connect to mysql error:" + err.Error())
		}
		_, err = db.Query("SELECT DATABASE()")
		if err != nil {
			if strings.Contains(err.Error(), "Unknown database") {
				log.Infof("first time connect to a new database,create database %s", dbName)
				err = provider.createDatabase(conf.Dsn, dbName)
				if err != nil {
					log.Panicf("failed to open mysql[%s] and create database %s, %s", dsn, dbName, err)
				}
				db, err = sql.Open("mysql", dsn)
				if err != nil {
					log.Panicf("failed to open mysql:%s , %s", dsn, err)
				}
			} else {
				log.Panicf("failed to open mysql:%s , %s", dsn, err)
			}
		}
		log.Debug("open new db connection for " + conf.SqlDbType + " dsn:" + dsn)
		db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifeTime))
		db.SetMaxIdleConns(conf.MaxIdleConns)
		db.SetMaxOpenConns(conf.MaxOpenConns)
		provider.db = db
		provider.contextDbName = dbName //默认连接mysql数据库
	} else if sqlType == types.Sqlite {
		dbPath := conf.Dsn
		if !strings.Contains(dbPath, ":memory:") { //不是内存数据库模式，则需要在路径中包含chainId
			dbPath = filepath.Join(dbPath, dbName)
			createDirIfNotExist(dbPath)
			dbPath = filepath.Join(dbPath, "sqlite.db")
		}
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Panicf("failed to open sqlite path:%s,get error:%s", dbPath, err)
		}
		provider.db = db
	} else {
		log.Panicf("unsupported db:%v", sqlType)
	}

	log.Debug("inject ChainMaker logger into db logger.")
	provider.log = log
	return provider
}
func (p *SqlDBHandle) createDatabase(dsn string, dbName string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	sqlStr := "create database " + dbName
	_, err = db.Exec(sqlStr)
	p.log.Debug("Exec sql:", sqlStr)
	return err
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
	_, err := p.db.Exec("use " + dbName)
	if err != nil { //切换失败，没有这个数据库，则创建
		p.log.Debugf("try to run 'use %s' get an error, it means database not exist, create it!", dbName)
		_, err = p.db.Exec("create database " + dbName)
		if err != nil {
			return err //创建失败
		}
		p.log.Debugf("create database %s", dbName)
		//创建成功，再次切换数据库
		_, err = p.db.Exec("use " + dbName)
		return err
	}
	p.log.Debugf("use database %s", dbName)
	p.contextDbName = dbName
	return nil
}

func (p *SqlDBHandle) CreateTableIfNotExist(objI interface{}) error {
	p.Lock()
	defer p.Unlock()
	obj := objI.(TableDDLGenerator)
	if !p.HasTable(obj) {
		return p.CreateTable(obj)
	}
	return nil
}
func (p *SqlDBHandle) HasTable(obj TableDDLGenerator) bool {
	//obj:=objI.(TableDDLGenerator)
	sql := ""
	if p.dbType == types.MySQL {
		sql = fmt.Sprintf(
			"SELECT count(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s' AND table_type = 'BASE TABLE'",
			p.contextDbName, obj.GetTableName())
	}
	if p.dbType == types.Sqlite {
		sql = fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=\"%s\"", obj.GetTableName())
	}
	p.log.Debug("Query sql:", sql)
	row := p.db.QueryRow(sql)
	count := 0
	row.Scan(&count)
	return count > 0
}
func (p *SqlDBHandle) CreateTable(obj TableDDLGenerator) error {
	sql := obj.GetCreateTableSql(p.dbType.LowerString())
	_, err := p.db.Exec(sql)
	return err
}

//ExecSql 执行SQL语句
func (p *SqlDBHandle) ExecSql(sql string, values ...interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	p.log.Debug("Exec sql:", sql, values)
	tx, err := p.db.Exec(sql, values...)
	if err != nil {
		return 0, err
	}
	return tx.RowsAffected()
}

func (p *SqlDBHandle) Save(val interface{}) (int64, error) {
	p.Lock()
	defer p.Unlock()
	value := val.(TableDMLGenerator)
	update, args := value.GetUpdateSql()
	p.log.Debug("Exec sql:", update, args)
	effect, err := p.db.Exec(update, args...)
	if err != nil {
		return 0, err
	}
	rowCount, err := effect.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rowCount != 0 {
		return rowCount, nil
	}
	insert, args := value.GetInsertSql()
	p.log.Debug("Exec sql:", insert, args)
	result, err := p.db.Exec(insert, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
func (p *SqlDBHandle) QuerySingle(sql string, values ...interface{}) (protocol.SqlRow, error) {
	p.Lock()
	defer p.Unlock()
	db := p.db
	p.log.Debug("Query sql:", sql, values)
	rows, err := db.Query(sql, values...)
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return &emptyRow{}, nil
	}
	return NewSqlDBRow(rows, nil), nil
}

func (p *SqlDBHandle) QueryMulti(sql string, values ...interface{}) (protocol.SqlRows, error) {
	p.Lock()
	defer p.Unlock()
	p.log.Debug("Query sql:", sql, values)
	rows, err := p.db.Query(sql, values...)
	if err != nil {
		return nil, err
	}
	return NewSqlDBRows(rows, nil), nil
}

func (p *SqlDBHandle) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	p.Lock()
	defer p.Unlock()

	if _, has := p.dbTxCache[txName]; has {
		return nil, errors.New("transaction already exist, please use GetDbTransaction to get it or commit/rollback it")
	}
	tx, err := p.db.Begin()
	if err != nil {
		return nil, err
	}
	sqltx := NewSqlDBTx(txName, p.dbType, tx, p.log)
	p.dbTxCache[txName] = sqltx
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
	err = tx.Commit()
	if err != nil {
		return err
	}
	delete(p.dbTxCache, txName)
	//p.log.Debugf("commit db transaction[%s]", txName) //devin: already log in tx.Commit()
	return nil
}
func (p *SqlDBHandle) RollbackDbTransaction(txName string) error {
	p.Lock()
	defer p.Unlock()
	tx, err := p.getDbTransaction(txName)
	if err != nil {
		return err
	}
	err = tx.Rollback()
	if err != nil {
		return err
	}
	delete(p.dbTxCache, txName)
	//p.log.Debugf("rollback db transaction[%s]", txName) //devin: already log in tx.Rollback()
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
	return p.db.Close()
}
