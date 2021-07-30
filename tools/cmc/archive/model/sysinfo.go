// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"strconv"

	"gorm.io/gorm"
)

const (
	// KArchivedblockheight archived_block_height
	KArchivedblockheight = "archived_block_height"
)

type Sysinfo struct {
	BaseModel
	K string `gorm:"unique;type:varchar(64) NOT NULL"`
	V string `gorm:"type:varchar(64) NOT NULL"`
}

func GetArchivedBlockHeight(db *gorm.DB) (uint64, error) {
	var sysinfo Sysinfo
	err := db.First(&sysinfo, "k = ?", KArchivedblockheight).Error
	if err != nil {
		// no KArchivedblockheight in sysinfos table, init create
		if err == gorm.ErrRecordNotFound {
			sysinfo.K = KArchivedblockheight
			sysinfo.V = "0"
			return 0, db.Create(&sysinfo).Error
		}
		return 0, err
	}

	return strconv.ParseUint(sysinfo.V, 10, 64)
}

func UpdateArchivedBlockHeight(db *gorm.DB, archivedBlockHeight uint64) error {
	return db.Model(&Sysinfo{}).Where("k = ?", KArchivedblockheight).
		Update("v", archivedBlockHeight).Error
}
