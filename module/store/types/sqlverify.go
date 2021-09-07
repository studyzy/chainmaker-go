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

	"chainmaker.org/chainmaker/utils/v2"
)

var errorNullSql = errors.New("null sql")
var errorInvalidSql = errors.New("invalid sql")
var errorForbiddenSql = errors.New("forbidden sql")
var errorForbiddenSqlKeyword = errors.New("forbidden sql keyword")
var errorForbiddenMultiSql = errors.New("forbidden multi sql statement in one function call")
var errorForbiddenDotInTable = errors.New("forbidden dot in table name")
var errorStateInfos = errors.New("you can't change table state_infos")

//StandardSqlVerify 如果状态数据库是标准SQL语句，对标准SQL的SQL语句进行语法检查，不关心具体的SQL DB类型的语法差异
type StandardSqlVerify struct {
}

func (s *StandardSqlVerify) VerifyDDLSql(sql string) error {
	newSql, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	if err := s.checkForbiddenSql(newSql); err != nil {
		return err
	}
	reg := regexp.MustCompile(`^(CREATE|ALTER|DROP)\s+(TABLE|VIEW|INDEX)`)
	match := reg.MatchString(newSql)
	if match {
		return nil
	}
	if strings.HasPrefix(newSql, "TRUNCATE TABLE") {
		return nil
	}
	return errorInvalidSql

}
func (s *StandardSqlVerify) VerifyDMLSql(sql string) error {
	newSql, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	if err := s.checkForbiddenSql(newSql); err != nil {
		return err
	}
	reg := regexp.MustCompile(`^(INSERT|UPDATE|DELETE)\s+`)
	match := reg.MatchString(newSql)
	if match {
		return nil
	}
	return errorInvalidSql
}
func (s *StandardSqlVerify) VerifyDQLSql(sql string) error {
	newSql, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	if err := s.checkForbiddenSql(newSql); err != nil {
		return err
	}
	reg := regexp.MustCompile(`^SELECT\s+`)
	match := reg.MatchString(newSql)
	if match {
		return nil
	}
	return errorInvalidSql
}

//禁用use database,禁用 select * from anotherdb.table形式
func (s *StandardSqlVerify) checkForbiddenSql(sql string) error {
	newSql, err := s.getFmtSql(sql)
	if err != nil {
		return err
	}
	reg := regexp.MustCompile(`^(USE|GRANT|CONN|REVOKE|DENY)\s+`)
	match := reg.MatchString(newSql)
	if match {
		return errorForbiddenSql
	}
	tableNames := utils.GetSqlTableName(newSql)
	for _, tableName := range tableNames {
		if strings.Contains(tableName, ".") {
			return errorForbiddenDotInTable
		}
		if strings.Contains(tableName, "STATE_INFOS") {
			return errorStateInfos
		}
	}
	count := utils.GetSqlStatementCount(newSql)
	if count > 1 {
		return errorForbiddenMultiSql
	}
	return s.checkHasForbiddenKeyword(newSql)
}
func (s *StandardSqlVerify) checkHasForbiddenKeyword(sql string) error {
	stringRanges := findStringRange(sql)
	reg := regexp.MustCompile(`(NOW|SYSDATE|RAND|NEWID|UUID)\s*\(`)
	result := reg.FindAllIndex([]byte(sql), -1)
	reg2 := regexp.MustCompile(`\s+(AUTO_INCREMENT|IDENTITY)[^\w]+`)
	result2 := reg2.FindAllIndex([]byte(sql), -1)
	result = append(result, result2...)
	for _, match := range result {
		if !isInString(match, stringRanges) {
			return errorForbiddenSqlKeyword
		}
	}
	return nil
}
func isInString(match []int, strRange [][2]int) bool {
	for _, strR := range strRange {
		if match[0] > strR[0] && match[0] < strR[1] {
			return true
		}
	}
	return false
}
func findStringRange(sql string) [][2]int {
	inString := false
	stringRange := [][2]int{}
	var range1 [2]int
	skipNext := false
	splitChar := int32(0)
	for i, c := range sql {
		if skipNext {
			skipNext = false
			continue
		}
		if (c == '\'' || c == '"') && (splitChar == 0 || c == splitChar) {
			if i != len(sql)-1 && int32(sql[i+1]) == c {
				skipNext = true
				continue
			}
			inString = !inString
			if inString {
				range1[0] = i
				splitChar = c
			} else {
				range1[1] = i
				stringRange = append(stringRange, range1)
				range1 = [2]int{}
				splitChar = 0
			}
		}
	}
	return stringRange
}

func (s *StandardSqlVerify) getFmtSql(sql string) (string, error) {
	newSql := strings.TrimSpace(sql)
	if len(newSql) == 0 {
		return "", errorNullSql
	}

	newSql = strings.ToUpper(newSql)

	return newSql, nil
}

//SqlVerifyPass 用于测试场景，不对SQL语句进行检查，任意SQL检查都通过
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
