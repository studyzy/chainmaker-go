/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mysqldbprovider

import (
	"chainmaker.org/chainmaker-go/localconf"

	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var defaultMaxIdleConns = 10
var defaultMaxOpenConns = 10
var defaultConnMaxLifeTime = 60

// Porvider encapsulate the gorm.DB that providers mysql handles
type Provider struct {
	sync.Mutex
	db *gorm.DB
}

// NewProvider construct a new Provider
func NewProvider() *Provider {
	return &Provider{}
}

// GetDB returns a new gorm.DB for given chainid and conf.
func (p *Provider) GetDB(chainId string, conf *localconf.CMConfig) *gorm.DB {
	p.Lock()
	defer p.Unlock()
	if p.db == nil {
		mysqlConf := conf.StorageConfig.MysqlConfig
		err := p.tryCreateDB(chainId, mysqlConf.Dsn)
		if err != nil {
			panic(fmt.Sprintf("failed to create mysql:%s", err))
		}
		//dsn := "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
		dsn := mysqlConf.Dsn + chainId + "?charset=utf8mb4&parseTime=True&loc=Local"
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(fmt.Sprintf("failed to open mysql:%s", err))
		}
		sqlDB, err := db.DB()
		if err != nil {
			panic(fmt.Sprintf("failed to open mysql:%s", err))
		}
		maxIdleConns := mysqlConf.MaxIdleConns
		if maxIdleConns <= 0 {
			maxIdleConns = defaultMaxIdleConns
		}
		sqlDB.SetMaxIdleConns(mysqlConf.MaxIdleConns)

		maxOpenConns := mysqlConf.MaxOpenConns
		if maxOpenConns <= 0 {
			maxOpenConns = defaultMaxOpenConns
		}
		sqlDB.SetMaxOpenConns(mysqlConf.MaxOpenConns)

		connMaxLifeTime := mysqlConf.ConnMaxLifeTime
		if connMaxLifeTime <= 0 {
			connMaxLifeTime = defaultConnMaxLifeTime
		}
		sqlDB.SetConnMaxLifetime(time.Duration(mysqlConf.ConnMaxLifeTime) * time.Second)
		p.db = db
	}
	return p.db
}

// tryCreateDB try create mysql database if not exist
func (p *Provider) tryCreateDB(dbName string, dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + dbName)
	if err != nil {
		return err
	}
	return nil
}

// CreateDB create mysql database for the given chainid and dsn
func (p *Provider) CreateDB(chainId string, dsn string) {
	p.Lock()
	defer p.Unlock()
	if err := p.tryCreateDB(chainId, dsn); err != nil {
		panic(fmt.Sprintf("failed to create mysql, err:%s", err))
	}
}
