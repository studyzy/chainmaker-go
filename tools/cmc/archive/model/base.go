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
