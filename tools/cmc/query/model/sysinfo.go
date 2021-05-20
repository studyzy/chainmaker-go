package model

import (
	"strconv"

	"gorm.io/gorm"
)

const (
	KArchivedblockheight = "archived_block_height"
)

type Sysinfo struct {
	BaseModel
	K string `gorm:"unique;type:varchar(64) NOT NULL"`
	V string `gorm:"type:varchar(64) NOT NULL"`
}

func GetArchivedBlockHeight(db *gorm.DB) (int64, error) {
	var sysinfo Sysinfo
	err := db.Find(&sysinfo, "k = ?", KArchivedblockheight).Error
	if err != nil {
		return 0, err
	}

	// no KArchivedblockheight in sysinfos table, init create
	if sysinfo.V == "" {
		sysinfo.K = KArchivedblockheight
		sysinfo.V = "0"
		err = db.Create(&sysinfo).Error
		if err != nil {
			return 0, err
		}
		return 0, nil
	}

	height, err := strconv.ParseInt(sysinfo.V, 10, 64)
	if err != nil {
		return 0, err
	}

	return height, nil
}

func UpdateArchivedBlockHeight(db *gorm.DB, archivedBlockHeight int64) error {
	return db.Model(&Sysinfo{}).Where("k = ?", KArchivedblockheight).
		Update("v", archivedBlockHeight).Error
}
