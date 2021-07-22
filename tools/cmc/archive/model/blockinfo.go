// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	rowsPerBlockInfoTable = uint64(100000)

	prefixDbName         = "cm_archived_chain"
	prefixBlockInfoTable = "t_block_info"
)

type BlockInfo struct {
	BaseModel
	ChainID        string `gorm:"column:Fchain_id;type:varchar(64) NOT NULL"`
	BlockHeight    uint64 `gorm:"column:Fblock_height;type:int unsigned NOT NULL;uniqueIndex:idx_blockheight"`
	BlockWithRWSet []byte `gorm:"column:Fblock_with_rwset;type:longblob NOT NULL"`
	Hmac           string `gorm:"column:Fhmac;type:varchar(64) NOT NULL"`
	IsArchived     bool   `gorm:"column:Fis_archived;type:tinyint(1) NOT NULL DEFAULT '0'"`
}

type blockInfoNew struct {
	BlockInfo
}

// TableName The BlockInfo table name will be overwritten as 't_block_info_1' the first sharding table
func (BlockInfo) TableName() string {
	return "t_block_info_1"
}

func (blockInfoNew) TableName() string {
	return "t_block_info_new"
}

// BlockInfoTableScopes BlockInfoTable implement Scopes for sharding.
func BlockInfoTableScopes(bInfo BlockInfo) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tableName := BlockInfoTableNameByBlockHeight(bInfo.BlockHeight)
		return tx.Table(tableName)
	}
}

// BlockInfoTableNameByBlockHeight Get BlockInfo table name by block height
func BlockInfoTableNameByBlockHeight(blkHeight uint64) string {
	tableNum := blkHeight/rowsPerBlockInfoTable + 1
	return fmt.Sprintf("%s_%d", prefixBlockInfoTable, tableNum)
}

// DbName DbName returns database name by chainId.
func DbName(chainId string) string {
	return fmt.Sprintf("%s_%s", prefixDbName, chainId)
}

func CreateBlockInfoTableIfNotExists(db *gorm.DB, tableName string) error {
	if !db.Migrator().HasTable(tableName) {
		if !db.Migrator().HasTable(&blockInfoNew{}) {
			err := db.Set("gorm:table_options", "ENGINE=InnoDB").Migrator().CreateTable(&blockInfoNew{})
			if err != nil {
				return err
			}
		}
		return db.Migrator().RenameTable(&blockInfoNew{}, tableName)
	}
	return nil
}

func InsertBlockInfo(db *gorm.DB, chainId string, blkHeight uint64, blkWithRWSet []byte, hmac string) error {
	return db.Table(BlockInfoTableNameByBlockHeight(blkHeight)).Create(&BlockInfo{
		ChainID:        chainId,
		BlockHeight:    blkHeight,
		BlockWithRWSet: blkWithRWSet,
		Hmac:           hmac,
		IsArchived:     true,
	}).Error
}

func RowsPerBlockInfoTable() uint64 {
	return rowsPerBlockInfoTable
}
