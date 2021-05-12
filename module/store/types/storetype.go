/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

import "strings"

type EngineType int32

const (
	UnknownDb EngineType = 0
	LevelDb   EngineType = 1
	RocksDb   EngineType = 2
	MySQL     EngineType = 3
	Sqlite    EngineType = 4
)

func (t EngineType) String() string {
	switch t {
	case UnknownDb:
		return "UnknownDb"
	case LevelDb:
		return "LevelDb"
	case RocksDb:
		return "RocksDb"
	case MySQL:
		return "MySQL"
	case Sqlite:
		return "Sqlite"
	}
	return ""
}
func (t EngineType) LowerString() string {
	return strings.ToLower(t.String())
}

var CommonDBDir = "common" // used to define database dir for other module (for instance consensus) to use kv database
