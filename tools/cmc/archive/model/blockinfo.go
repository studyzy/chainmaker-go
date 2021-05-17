package model

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	rowsPerBlockInfoTable = 100000

	prefixDbName         = "cmc_archived_chain"
	prefixBlockInfoTable = "t_block_info"
)

type BlockInfo struct {
	BaseModel
	ChainID        string `gorm:"column:Fchain_id;type:varchar(64) NOT NULL"`
	BlockHeight    int64  `gorm:"column:Fblock_height;type:int unsigned NOT NULL;uniqueIndex:idx_blockheight"`
	BlockWithRWSet []byte `gorm:"column:Fblock_with_rwset;type:blob NOT NULL"`
	Hmac           string `gorm:"column:Fhmac;type:varchar(64) NOT NULL"`
}

// TableName The BlockInfo table name will be overwritten as 't_block_info_1' the first sharding table
func (BlockInfo) TableName() string {
	return "t_block_info_1"
}

// BlockInfoTable BlockInfoTable implement Scopes for sharding.
func BlockInfoTable(bInfo BlockInfo) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tableNum := bInfo.BlockHeight/rowsPerBlockInfoTable + 1
		tableName := fmt.Sprintf("%s_%d", prefixBlockInfoTable, tableNum)
		return tx.Table(tableName)
	}
}

// DBName DBName returns database name by chainId.
func DBName(chainId string) string {
	return fmt.Sprintf("%s_%s", prefixDbName, chainId)
}

func InsertBlockInfo(db *gorm.DB, chainId string, blkHeight int64, blkWithRWSet []byte, hmac string) (int64, error) {
	var bInfo = BlockInfo{
		ChainID:        chainId,
		BlockHeight:    blkHeight,
		BlockWithRWSet: blkWithRWSet,
		Hmac:           hmac,
	}
	result := db.Create(&bInfo)
	return result.RowsAffected, result.Error
}
