package eventmysqldb

import (
	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/contracteventdb"
	"chainmaker.org/chainmaker-go/store/dbprovider/sqldbprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"gorm.io/gorm"
)

// BlockMysqlDB provider a implementation of `contracteventdb.ContractEventDB`
// This implementation provides a mysql based data model
type ContractEventMysqlDB struct {
	db     protocol.SqlDBHandle
	Logger protocol.Logger
}

// NewContractEventMysqlDB construct a new `ContractEventDB` for given chainId
func NewContractEventMysqlDB(chainId string, sqlDbConfig *localconf.SqlDbConfig,  logger protocol.Logger) (contracteventdb.ContractEventDB, error) {
	db := sqldbprovider.NewSqlDBHandle(chainId, sqlDbConfig, logger)
	return newResultSqlDB(chainId, db, logger)
}
func newResultSqlDB(chainId string, db protocol.SqlDBHandle, logger protocol.Logger) (*ContractEventMysqlDB, error) {
	rdb := &ContractEventMysqlDB{
		db:     db,
		Logger: logger,
	}
	return rdb, nil
}
{
	var contractEventDb *ContractEventMysqlDB
	if !localconf.ChainMakerConfig.StorageConfig.EnableContractEventDB {
		contractEventDb = &ContractEventMysqlDB{
			db:     nil,
			Logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
		}
	} else {
		db := sqldbprovider.NewSqlDBHandle(chainId, sqlDbConfig, logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId))
		contractEventDb = &ContractEventMysqlDB{
			db:     db,
			Logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
		}
		err := contractEventDb.CreateTable(CreateBlockHeightWithTopicTableDdl)
		if err != nil {
			panic(fmt.Sprintf("failed to create %s db:%s", BlockHeightWithTopicTableName, err))
		}
		err = contractEventDb.CreateTable(CreateBlockHeightIndexTableDDL)
		if err != nil {
			panic(fmt.Sprintf("failed to create %s db:%s", BlockHeightIndexTableName, err))
		}
		err = contractEventDb.initBlockHeightIndexTable()
		if err != nil {
			panic(fmt.Sprintf("failed to init %s db:%s", BlockHeightIndexTableName, err))
		}

	}
	return contractEventDb, nil
}

// CommitBlock commits the event in an atomic operation
func (c *ContractEventMysqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	//if not enable contract event db ,return nil
	if c.db == nil {
		return nil
	}
	block := blockInfo.Block
	chanId := block.Header.ChainId
	blockHeight := block.Header.BlockHeight
	contractEventInfo := blockInfo.ContractEvents
	blockIndexDdl := utils.GenerateUpdateBlockHeightIndexDdl(block.Header.BlockHeight)
	return c.db.Transaction(func(tx *gorm.DB) error {
		var res *gorm.DB
		for _, event := range contractEventInfo {

			saveDdl := utils.GenerateSaveContractEventDdl(event, chanId, blockHeight)
			createDdl := utils.GenerateCreateTopicTableDdl(event, chanId)
			heightWithTopicDdl := utils.GenerateSaveBlockHeightWithTopicDdl(event, chanId, blockHeight)
			topicTableName := chanId + "_" + event.ContractName + "_" + event.Topic

			if createDdl != "" {
				res = tx.Debug().Exec(createDdl)
			}
			if res.Error != nil {
				c.Logger.Errorf("failed to create contract event topic table, contract:%s, topic:%s, err:%s", event.ContractName, event.Topic, res.Error)
				return res.Error
			}
			if saveDdl != "" {
				res = tx.Debug().Exec(saveDdl)
			}

			if res.Error != nil {
				c.Logger.Errorf("failed to save contract event, contract:%s, topic:%s, err:%s", event.ContractName, event.Topic, res.Error)
				return res.Error
			}
			if heightWithTopicDdl != "" {
				res = tx.Debug().Exec(heightWithTopicDdl)
			}
			if res.Error != nil {
				c.Logger.Errorf("failed to save block height with topic table, height:%s, topicTableName:%s, err:%s", block.Header.BlockHeight, topicTableName, res.Error)
				return res.Error
			}
		}

		res = tx.Debug().Exec(blockIndexDdl)
		if res.Error != nil {
			c.Logger.Errorf("failed to update block height index, height:%s err:%s", block.Header.BlockHeight, res.Error)
			return res.Error
		}
		c.Logger.Debugf("chain[%s]: commit contract event block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return nil
	})
}

// GetLastSavepoint returns the last block height
func (c *ContractEventMysqlDB) GetLastSavepoint() (uint64, error) {
	var blockHeight int64
	err := c.CreateTable(CreateBlockHeightIndexTableDDL)
	if err != nil {
		c.Logger.Errorf("GetLastSavepoint: try to create " + BlockHeightWithTopicTableName + " table fail")
		return 0, err
	}
	err = c.initBlockHeightIndexTable()
	if err != nil {
		c.Logger.Errorf("GetLastSavepoint: init " + BlockHeightWithTopicTableName + " table fail")
		return 0, err
	}
	err = c.CreateTable(CreateBlockHeightWithTopicTableDdl)
	if err != nil {
		c.Logger.Errorf("GetLastSavepoint: try to create " + BlockHeightIndexTableName + " table fail")
		return 0, err
	}

	row := c.db.Raw("select block_height from " + BlockHeightIndexTableName + "  order by id desc limit 1").Row()
	row.Scan(&blockHeight)
	if row.Err() != nil && row.Err() != gorm.ErrRecordNotFound {
		c.Logger.Errorf("failed to get last savepoint")
		return 0, row.Err()
	}
	return uint64(blockHeight), row.Err()
}

// insert a record to init block height index table
func (c *ContractEventMysqlDB) initBlockHeightIndexTable() error {
	exec := c.db.Debug().Exec(InitBlockHeightIndexTableDDL)
	return exec.Error
}

// Close is used to close database, there is no need for gorm to close db
func (c *ContractEventMysqlDB) Close() {
	sqlDB, err := c.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}

// CreateTable create a contract event topic table
func (c *ContractEventMysqlDB) CreateTable(ddl string) error {
	exec := c.db.Debug().Exec(ddl)
	return exec.Error
}
