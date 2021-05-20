package model

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	rowsPerBlockInfoTable = 100000

	prefixDbName         = "cm_archived_chain"
	prefixBlockInfoTable = "t_block_info"
)

type BlockInfo struct {
	BaseModel
	ChainID        string `gorm:"column:Fchain_id;type:varchar(64) NOT NULL"`
	BlockHeight    int64  `gorm:"column:Fblock_height;type:int unsigned NOT NULL;uniqueIndex:idx_blockheight"`
	BlockWithRWSet []byte `gorm:"column:Fblock_with_rwset;type:longblob NOT NULL"`
	Hmac           string `gorm:"column:Fhmac;type:varchar(64) NOT NULL"`
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
func BlockInfoTableNameByBlockHeight(blkHeight int64) string {
	tableNum := blkHeight/rowsPerBlockInfoTable + 1
	return fmt.Sprintf("%s_%d", prefixBlockInfoTable, tableNum)
}

// DbName DbName returns database name by chainId.
func DbName(chainId string) string {
	return fmt.Sprintf("%s_%s", prefixDbName, chainId)
}

func createBlockInfoTable(db *gorm.DB, tableName string) error {
	err := db.Set("gorm:table_options", "ENGINE=InnoDB").Migrator().CreateTable(&blockInfoNew{})
	if err != nil {
		return err
	}
	return db.Migrator().RenameTable(&blockInfoNew{}, tableName)
}

func InsertBlockInfo(db *gorm.DB, chainId string, blkHeight int64, blkWithRWSet []byte, hmac string) (int64, error) {
	var bInfo = BlockInfo{
		ChainID:        chainId,
		BlockHeight:    blkHeight,
		BlockWithRWSet: blkWithRWSet,
		Hmac:           hmac,
	}
	result := db.Scopes(BlockInfoTableScopes(bInfo)).Create(&bInfo)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "Error 1146") {
			err := createBlockInfoTable(db, BlockInfoTableNameByBlockHeight(bInfo.BlockHeight))
			if err != nil {
				return 0, err
			}
			result = db.Scopes(BlockInfoTableScopes(bInfo)).Create(&bInfo)
		}
	}

	return result.RowsAffected, result.Error
}
