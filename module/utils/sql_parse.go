/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package utils

import (
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/studyzy/sqlparse"
)

type colX struct {
	tableNames []string
}

func (v *colX) Enter(in ast.Node) (ast.Node, bool) {
	if name, ok := in.(*ast.TableName); ok {
		if name.Schema.String() == "" {
			v.tableNames = append(v.tableNames, name.Name.String())
		} else {
			v.tableNames = append(v.tableNames, name.Schema.String()+"."+name.Name.String())
		}
	}
	return in, false
}

func (v *colX) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
func extract(rootNode *ast.StmtNode) []string {
	v := &colX{}
	(*rootNode).Accept(v)
	return v.tableNames
}

//GetSqlTableName 获得SQL中使用到的表名，如果带有dbName.tableName，那么返回完整的dbName.tableName
func GetSqlTableName(sql string) []string {
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil
	}
	return extract(&stmtNodes[0])
}

//GetSqlStatementCount 判断一个sql字符串是由多少条独立的SQL语句组成
func GetSqlStatementCount(sql string) int {
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return 0
	}
	return len(stmtNodes)
}
