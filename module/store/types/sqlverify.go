/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package types

import (
	"errors"
	"regexp"
	"strings"
)

var ERROR_NULL_SQL = errors.New("null sql")
var ERROR_INVALID_SQL = errors.New("invalid sql")
var ERROR_FORBIDDEN_SQL = errors.New("forbidden sql")
var ERROR_FORBIDDEN_MULTI_SQL = errors.New("forbidden multi sql statement in one function call")
var ERROR_FORBIDDEN_DOT_IN_TABLE = errors.New("forbidden dot in table name")
var ERROR_STATE_INFOS = errors.New("you can't change table state_infos")

//如果状态数据库是标准SQL语句，对标准SQL的SQL语句进行语法检查，不关心具体的SQL DB类型的语法差异
type StandardSqlVerify struct {
}

func (s *StandardSqlVerify) VerifyDDLSql(sql string) error {
	SQL, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	if err := s.checkForbiddenSql(SQL); err != nil {
		return err
	}
	reg := regexp.MustCompile(`^(CREATE|ALTER|DROP)\s+(TABLE|VIEW|INDEX)`)
	match := reg.MatchString(SQL)
	if match {
		return nil
	}
	if strings.HasPrefix(SQL, "TRUNCATE TABLE") {
		return nil
	}
	return ERROR_INVALID_SQL

}
func (s *StandardSqlVerify) VerifyDMLSql(sql string) error {
	SQL, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	if err := s.checkForbiddenSql(SQL); err != nil {
		return err
	}
	reg := regexp.MustCompile(`^(INSERT|UPDATE|DELETE)\s+`)
	match := reg.MatchString(SQL)
	if match {
		return nil
	}
	return ERROR_INVALID_SQL
}
func (s *StandardSqlVerify) VerifyDQLSql(sql string) error {
	SQL, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	if err := s.checkForbiddenSql(SQL); err != nil {
		return err
	}
	reg := regexp.MustCompile(`^SELECT\s+`)
	match := reg.MatchString(SQL)
	if match {
		return nil
	}
	return ERROR_INVALID_SQL
}

//禁用use database,禁用 select * from anotherdb.table形式
func (s *StandardSqlVerify) checkForbiddenSql(sql string) error {
	SQL, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	reg := regexp.MustCompile(`^(USE|GRANT|CONN|REVOKE|DENY)\s+`)
	match := reg.MatchString(SQL)
	if match {
		return ERROR_FORBIDDEN_SQL
	}
	tableNames := s.getSqlTableName(sql)
	for _, tableName := range tableNames {
		if strings.Contains(tableName, ".") {
			return ERROR_FORBIDDEN_DOT_IN_TABLE
		}
		if strings.Contains(tableName, "STATE_INFOS") {
			return ERROR_STATE_INFOS
		}
	}
	count := s.getSqlStatementCount(sql)
	if count > 1 {
		return ERROR_FORBIDDEN_MULTI_SQL
	}
	return nil
}

func (s *StandardSqlVerify) getFmtSql(sql string) (string, error) {
	SQL := strings.TrimSpace(sql)
	if SQL == "" {
		return "", ERROR_NULL_SQL
	}

	SQL = strings.ToUpper(SQL)

	if SQL[len(SQL)-1] == ';' {
		SQL = SQL[0 : len(SQL)-2]
	}
	return SQL, nil
}

//用于测试场景，不对SQL语句进行检查，任意SQL检查都通过
type SqlVerifyPass struct {
}

func (s *SqlVerifyPass) VerifyDDLSql(sql string) error {
	return nil
}
func (s *SqlVerifyPass) VerifyDMLSql(sql string) error {
	return nil
}
func (s *SqlVerifyPass) VerifyDQLSql(sql string) error {
	return nil
}
