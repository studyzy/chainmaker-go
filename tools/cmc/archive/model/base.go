// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"database/sql"
)

type BaseModel struct {
	ID        int64        `gorm:"primaryKey;column:Fid;type:int unsigned NOT NULL AUTO_INCREMENT"`
	CreatedAt sql.NullTime `gorm:"index;column:Fcreate_time;type:timestamp"`
	UpdatedAt sql.NullTime `gorm:"column:Fmodify_time;type:timestamp"`
	DeletedAt sql.NullTime `gorm:"column:Fdelete_time;type:timestamp"`
}
