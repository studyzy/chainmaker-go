package eventsqldb

import (
	"fmt"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/utils"
)

// BlockMysqlDB provider a implementation of `contracteventdb.ContractEventDB`
// This implementation provides a mysql based data model
type ContractEventSqlDB struct {
	db     protocol.SqlDBHandle
	Logger protocol.Logger
	dbName string
}

// NewContractEventMysqlDB construct a new `ContractEventDB` for given chainId
func NewContractEventMysqlDB(chainId string, sqlDbConfig *localconf.SqlDbConfig, logger protocol.Logger) (*ContractEventSqlDB, error) {
	dbName := getDbName(sqlDbConfig, chainId)
	db := rawsqlprovider.NewSqlDBHandle(dbName, sqlDbConfig, logger)
	return newContractEventDB(dbName, db, logger)
}

func newContractEventDB(dbName string, db protocol.SqlDBHandle, logger protocol.Logger) (*ContractEventSqlDB, error) {
	cdb := &ContractEventSqlDB{
		db:     db,
		Logger: logger,
		dbName: dbName,
	}
	cdb.initDb(dbName)
	return cdb, nil
}

func (c *ContractEventSqlDB) initDb(dbName string) {

	err := c.db.CreateDatabaseIfNotExist(dbName)
	if err != nil {
		panic(fmt.Sprintf("failed to create database %s db:%s", dbName, err))
	}
	err = c.createTable(CreateBlockHeightWithTopicTableDdl)
	if err != nil {
		panic(fmt.Sprintf("failed to create table %s db:%s", BlockHeightWithTopicTableName, err))
	}
	err = c.createTable(CreateBlockHeightIndexTableDdl)
	if err != nil {
		panic(fmt.Sprintf("failed to create table %s db:%s", BlockHeightIndexTableName, err))
	}
	err = c.initBlockHeightIndexTable()
	if err != nil {
		panic(fmt.Sprintf("failed to init %s db:%s", BlockHeightIndexTableName, err))
	}

}

func (c *ContractEventSqlDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	c.initDb(c.dbName)
	return nil
}

func getDbName(sqlDbConfig *localconf.SqlDbConfig, chainId string) string {
	return sqlDbConfig.DbPrefix + "contract_eventdb" + chainId
}

// CommitBlock commits the event in an atomic operation
func (c *ContractEventSqlDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	//if not enable contract event db ,return nil
	if c.db == nil {
		return nil
	}
	block := blockInfo.Block
	chanId := block.Header.ChainId
	blockHeight := block.Header.BlockHeight
	blockIndexDdl := utils.GenerateUpdateBlockHeightIndexDdl(block.Header.BlockHeight)
	blockHashStr := block.GetBlockHashStr()
	topicTableCache := make(map[string]bool)
	dbTx, err := c.db.BeginDbTransaction(blockHashStr)
	if err != nil {
		return err
	}
	var preBlockHeight int64
	single, err := c.db.QuerySingle("select block_height from " + BlockHeightIndexTableName + "  order by id desc limit 1")
	err = single.ScanColumns(&preBlockHeight)
	if err != nil {
		c.Logger.Errorf("failed to get block_height err%s", err)
		c.db.RollbackDbTransaction(blockHashStr)
		return err
	}
	//avoid nodes repeat commit block in same db
	if blockHeight <= preBlockHeight {
		c.db.CommitDbTransaction(blockHashStr)
		c.Logger.Debugf("chain[%s]: commit contract event block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return nil
	}
	for _, tx := range blockInfo.Block.Txs {
		for eventIndex, event := range tx.Result.ContractResult.ContractEvent {
			createDdl := utils.GenerateCreateTopicTableDdl(event, chanId)
			saveDdl := utils.GenerateSaveContractEventDdl(event, chanId, blockHeight, eventIndex)
			heightWithTopicDdl := utils.GenerateSaveBlockHeightWithTopicDdl(event, chanId, blockHeight)
			topicTableName := chanId + "_" + event.ContractName + "_" + event.Topic
			if createDdl != "" {
				_, err := dbTx.ExecSql(createDdl)
				if err != nil {
					c.Logger.Errorf("failed to create contract event topic table, contract:%s, topic:%s, err:%s", event.ContractName, event.Topic, err.Error)
					c.db.RollbackDbTransaction(blockHashStr)
					return err
				}
			}

			if saveDdl != "" {
				_, err := dbTx.ExecSql(saveDdl)
				if err != nil {
					c.Logger.Errorf("failed to save contract event, contract:%s, topic:%s, err:%s", event.ContractName, event.Topic, err.Error)
					c.db.RollbackDbTransaction(blockHashStr)
					return err
				}
			}

			if heightWithTopicDdl != "" {
				if _, ok := topicTableCache[topicTableName]; !ok {
					topicTableCache[topicTableName] = true
					_, err := dbTx.ExecSql(heightWithTopicDdl)
					if err != nil {
						c.Logger.Errorf("failed to save block height with topic table, height:%s, topicTableName:%s, err:%s", block.Header.BlockHeight, topicTableName, err.Error())
						c.db.RollbackDbTransaction(blockHashStr)
						return err
					}
				}
			}
		}
	}
	_, err = dbTx.ExecSql(blockIndexDdl)
	if err != nil {
		c.Logger.Errorf("failed to update block height index, height:%s err:%s", block.Header.BlockHeight, err.Error())
		c.db.RollbackDbTransaction(blockHashStr)
		return err
	}

	c.db.CommitDbTransaction(blockHashStr)
	c.Logger.Debugf("chain[%s]: commit contract event block[%d]",
		block.Header.ChainId, block.Header.BlockHeight)
	return nil
}

// GetLastSavepoint returns the last block height
func (c *ContractEventSqlDB) GetLastSavepoint() (uint64, error) {
	var blockHeight int64
	_, err := c.db.ExecSql(CreateBlockHeightIndexTableDdl)
	if err != nil {
		c.Logger.Errorf("GetLastSavepoint: try to create " + BlockHeightWithTopicTableName + " table fail")
		return 0, err
	}
	err = c.initBlockHeightIndexTable()
	if err != nil {
		c.Logger.Errorf("GetLastSavepoint: init " + BlockHeightWithTopicTableName + " table fail")
		return 0, err
	}
	err = c.createTable(CreateBlockHeightWithTopicTableDdl)
	if err != nil {
		c.Logger.Errorf("GetLastSavepoint: try to create " + BlockHeightIndexTableName + " table fail")
		return 0, err
	}

	single, err := c.db.QuerySingle("select block_height from " + BlockHeightIndexTableName + "  order by id desc limit 1")
	err = single.ScanColumns(&blockHeight)
	if err != nil {
		c.Logger.Errorf("failed to get last savepoint")
		return 0, err
	}
	return uint64(blockHeight), err
}

// insert a record to init block height index table
func (c *ContractEventSqlDB) initBlockHeightIndexTable() error {
	_, err := c.db.ExecSql(InitBlockHeightIndexTableDdl)
	return err
}

// Close is used to close database, there is no need for gorm to close db
func (c *ContractEventSqlDB) Close() {
	c.Logger.Info("close result sql db")
	c.db.Close()

}

// CreateTable create a contract event topic table
func (c *ContractEventSqlDB) createTable(ddl string) error {
	_, err := c.db.ExecSql(ddl)
	return err
}
