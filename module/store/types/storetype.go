/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

type EngineType int32

const (
	LevelDb EngineType = 1
	RocksDb EngineType = 2
	MySQL   EngineType = 3
)

var CommonDBDir = "common" // used to define database dir for other module (for instance consensus) to use kv database
