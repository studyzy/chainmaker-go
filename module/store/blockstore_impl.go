/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/archive"
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/contracteventdb"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/resultdb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/statedb"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	configPb "chainmaker.org/chainmaker/pb-go/config"
	storePb "chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/protocol"
	"github.com/tidwall/wal"
	"golang.org/x/sync/semaphore"
)

const (
	logPath = "wal"
	//logDBBlockKeyPrefix = 'n'
)

// BlockStoreImpl provides an implementation of `protocol.BlockchainStore`.
type BlockStoreImpl struct {
	blockDB         blockdb.BlockDB
	stateDB         statedb.StateDB
	historyDB       historydb.HistoryDB
	resultDB        resultdb.ResultDB
	contractEventDB contracteventdb.ContractEventDB
	wal             binlog.BinLoger
	//一个本地数据库，用于对外提供一些本节点的数据存储服务
	commonDB         protocol.DBHandle
	ArchiveMgr       *archive.ArchiveMgr
	workersSemaphore *semaphore.Weighted
	logger           protocol.Logger
	storeConfig      *localconf.StorageConfig
}

// NewBlockStoreImpl constructs new `BlockStoreImpl`
func NewBlockStoreImpl(chainId string,
	blockDB blockdb.BlockDB,
	stateDB statedb.StateDB,
	historyDB historydb.HistoryDB,
	contractEventDB contracteventdb.ContractEventDB,
	resultDB resultdb.ResultDB,
	commonDB protocol.DBHandle,
	storeConfig *localconf.StorageConfig,
	binLog binlog.BinLoger,
	logger protocol.Logger) (*BlockStoreImpl, error) {
	walPath := filepath.Join(storeConfig.StorePath, chainId, logPath)
	writeAsync := storeConfig.LogDBWriteAsync
	walOpt := &wal.Options{
		NoSync: writeAsync,
	}
	if binLog == nil {
		writeLog, err := wal.Open(walPath, walOpt)
		if err != nil {
			panic(fmt.Sprintf("open wal failed, path:%s, error:%s", walPath, err))
		}
		binLog = writeLog
	}
	nWorkers := runtime.NumCPU()

	blockStore := &BlockStoreImpl{
		blockDB:          blockDB,
		stateDB:          stateDB,
		historyDB:        historyDB,
		contractEventDB:  contractEventDB,
		resultDB:         resultDB,
		wal:              binLog,
		commonDB:         commonDB,
		workersSemaphore: semaphore.NewWeighted(int64(nWorkers)),
		logger:           logger,
		storeConfig:      storeConfig,
	}

	if err :=  blockStore.InitArchiveMgr(chainId); err != nil {
		return nil, err
	}

	//binlog 有SavePoint，不是空数据库，进行数据恢复
	if i, errbs := blockStore.getLastSavepoint(); errbs == nil && i > 0 {
		//check savepoint and recover
		errbs = blockStore.recover()
		if errbs != nil {
			return nil, errbs
		}
	} else {
		logger.Info("binlog is empty, don't need recover")
	}
	return blockStore, nil
}

//InitGenesis 初始化创世区块到数据库，对应的数据库必须为空数据库，否则报错
func (bs *BlockStoreImpl) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	bs.logger.Debug("start initial genesis block to database...")
	//1.检查创世区块是否有异常
	if err := checkGenesis(genesisBlock); err != nil {
		return err
	}
	//创世区块只执行一次，而且可能涉及到创建创建数据库，所以串行执行，而且无法启用事务
	blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(genesisBlock)
	if err != nil {
		return err
	}
	block := genesisBlock.Block
	err = bs.writeLog(uint64(block.Header.BlockHeight), blockBytes)
	if err != nil {
		return err
	}
	//2.初始化BlockDB
	err = bs.blockDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write blockDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}
	//3. 初始化StateDB
	err = bs.stateDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write stateDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}
	//4. 初始化历史数据库
	err = bs.historyDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write historyDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}
	//5. 初始化Result数据库
	err = bs.resultDB.InitGenesis(blockWithSerializedInfo)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write resultDB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		return err
	}
	//6. init contract event db
	if !bs.storeConfig.DisableContractEventDB {
		if parseEngineType(bs.storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
			bs.storeConfig.ContractEventDbConfig.Provider == localconf.DbConfig_Provider_Sql {
			err = bs.contractEventDB.InitGenesis(blockWithSerializedInfo)
			if err != nil {
				bs.logger.Errorf("chain[%s] failed to write event db, block[%d]",
					block.Header.ChainId, block.Header.BlockHeight)
				return err
			}
		} else {
			return errors.New("contract event db config err")
		}
	}
	bs.logger.Infof("chain[%s]: put block[%d] (txs:%d bytes:%d), ",
		block.Header.ChainId, block.Header.BlockHeight, len(block.Txs), len(blockBytes))

	//7. init archive manager
	err = bs.InitArchiveMgr(block.Header.ChainId)
	if err != nil {
		return err
	}

	return err
}
func checkGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	if genesisBlock.Block.Header.BlockHeight != 0 {
		return errors.New("genesis block height must be 0")
	}
	return nil
}

// PutBlock commits the block and the corresponding rwsets in an atomic operation
func (bs *BlockStoreImpl) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	bs.logger.Infof("chain[%s]: start put block[%d]", block.Header.ChainId, block.Header.BlockHeight)

	startPutBlock := utils.CurrentTimeMillisSeconds()
	//1. commit log
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}

	blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write log, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}
	elapsedMarshalBlockAndRWSet := utils.CurrentTimeMillisSeconds() - startPutBlock

	startCommitLogDB := utils.CurrentTimeMillisSeconds()
	err = bs.writeLog(uint64(block.Header.BlockHeight), blockBytes)
	elapsedCommitlogDB := utils.CurrentTimeMillisSeconds() - startCommitLogDB
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write log, block[%d], err:%s",
			block.Header.ChainId, block.Header.BlockHeight, err)
		return err
	}

	//commit db concurrently
	startCommitBlock := utils.CurrentTimeMillisSeconds()
	//the amount of commit db work
	numBatches := 5
	var batchWG sync.WaitGroup
	batchWG.Add(numBatches)
	errsChan := make(chan error, numBatches)
	// 2.commit blockDB
	go func() {
		defer batchWG.Done()
		bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.blockDB.CommitBlock)
	}()

	// 3.commit stateDB
	go func() {
		defer batchWG.Done()
		bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.stateDB.CommitBlock)
	}()

	// 4.commit historyDB
	if !bs.storeConfig.DisableHistoryDB {
		go func() {
			defer batchWG.Done()
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.historyDB.CommitBlock)
		}()
	} else {
		batchWG.Done()
	}
	//5. result db
	if !bs.storeConfig.DisableResultDB {
		go func() {
			defer batchWG.Done()
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.resultDB.CommitBlock)
		}()
	} else {
		batchWG.Done()
	}
	//6.commit contractEventDB
	if !bs.storeConfig.DisableContractEventDB {
		go func() {
			defer batchWG.Done()
			bs.putBlock2DB(blockWithSerializedInfo, errsChan, bs.contractEventDB.CommitBlock)
		}()
	} else {
		batchWG.Done()
	}

	batchWG.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	elapsedCommitBlock := utils.CurrentTimeMillisSeconds() - startCommitBlock

	//7. clean wal, delete block and rwset after commit
	go func() {
		err := bs.deleteBlockFromLog(uint64(block.Header.BlockHeight))
		if err != nil {
			bs.logger.Warnf("chain[%s]: failed to clean log, block[%d], err:%s",
				block.Header.ChainId, block.Header.BlockHeight, err)
		}
	}()
	bs.logger.Infof("chain[%s]: put block[%d] (txs:%d bytes:%d), "+
		"time used (mashal:%d, log:%d, commit:%d, total:%d)",
		block.Header.ChainId, block.Header.BlockHeight, len(block.Txs), len(blockBytes),
		elapsedMarshalBlockAndRWSet, elapsedCommitlogDB, elapsedCommitBlock,
		utils.CurrentTimeMillisSeconds()-startPutBlock)
	return nil
}

// GetArchivedPivot return archived pivot
func (bs *BlockStoreImpl) GetArchivedPivot() uint64 {
	if !bs.isSupportArchive() {
		return 0
	}
	height, _ := bs.ArchiveMgr.GetArchivedPivot()
	return height
}

// ArchiveBlock the block after backup
func (bs *BlockStoreImpl) ArchiveBlock(archiveHeight uint64) error {
	if !bs.isSupportArchive() {
		return nil
	}
	return bs.ArchiveMgr.ArchiveBlock(archiveHeight)
}

// RestoreBlocks restore blocks from outside serialized block data
func (bs *BlockStoreImpl) RestoreBlocks(serializedBlocks [][]byte) error {
	if !bs.isSupportArchive() {
		return nil
	}
	blockInfos := make([]*serialization.BlockWithSerializedInfo, 0, len(serializedBlocks))
	for _, blockInfo := range serializedBlocks {
		bwsInfo, err := serialization.DeserializeBlock(blockInfo)
		if err != nil {
			return err
		}

		blockInfos = append(blockInfos, bwsInfo)
	}

	return bs.ArchiveMgr.RestoreBlock(blockInfos)
}

type commitBlock func(blockInfo *serialization.BlockWithSerializedInfo) error

func (bs *BlockStoreImpl) putBlock2DB(blockWithSerializedInfo *serialization.BlockWithSerializedInfo,
	errsChan chan error, commit commitBlock) {
	err := commit(blockWithSerializedInfo)
	block := blockWithSerializedInfo.Block
	if err != nil {
		bs.logger.Errorf("chain[%s] failed to write DB, block[%d]",
			block.Header.ChainId, block.Header.BlockHeight)
		errsChan <- err
	}
}

// BlockExists returns true if the black hash exist, or returns false if none exists.
func (bs *BlockStoreImpl) BlockExists(blockHash []byte) (bool, error) {
	return bs.blockDB.BlockExists(blockHash)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
func (bs *BlockStoreImpl) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	return bs.blockDB.GetBlockByHash(blockHash)
}

// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
func (bs *BlockStoreImpl) GetHeightByHash(blockHash []byte) (uint64, error) {
	return bs.blockDB.GetHeightByHash(blockHash)
}

// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
func (bs *BlockStoreImpl) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	return bs.blockDB.GetBlockHeaderByHeight(height)
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
func (bs *BlockStoreImpl) GetBlock(height uint64) (*commonPb.Block, error) {
	return bs.blockDB.GetBlock(height)
}

// GetLastBlock returns the last block.
func (bs *BlockStoreImpl) GetLastBlock() (*commonPb.Block, error) {
	return bs.blockDB.GetLastBlock()
}

// GetLastConfigBlock returns the last config block.
func (bs *BlockStoreImpl) GetLastConfigBlock() (*commonPb.Block, error) {
	return bs.blockDB.GetLastConfigBlock()
}

//GetLastChainConfig returns the last chain config
func (bs *BlockStoreImpl) GetLastChainConfig() (*configPb.ChainConfig, error) {
	return bs.stateDB.GetChainConfig()
}

// GetBlockByTx returns a block which contains a tx.
func (bs *BlockStoreImpl) GetBlockByTx(txId string) (*commonPb.Block, error) {
	return bs.blockDB.GetBlockByTx(txId)
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
func (bs *BlockStoreImpl) GetTx(txId string) (*commonPb.Transaction, error) {
	return bs.blockDB.GetTx(txId)
}

// GetTxHeight retrieves a transaction height by txid, or returns nil if none exists.
func (bs *BlockStoreImpl) GetTxHeight(txId string) (uint64, error) {
	return bs.blockDB.GetTxHeight(txId)
}

func (bs *BlockStoreImpl) GetTxWithBlockInfo(txId string) (*commonPb.TransactionInfo, error) {
	return bs.blockDB.GetTxWithBlockInfo(txId)
}

// GetTxConfirmedTime returns the confirmed time of a given tx
func (bs *BlockStoreImpl) GetTxConfirmedTime(txId string) (int64, error) {
	return bs.blockDB.GetTxConfirmedTime(txId)
}

// TxExists returns true if the tx exist, or returns false if none exists.
func (bs *BlockStoreImpl) TxExists(txId string) (bool, error) {
	return bs.blockDB.TxExists(txId)
}

// ReadObject returns the state value for given contract name and key, or returns nil if none exists.
func (bs *BlockStoreImpl) ReadObject(contractName string, key []byte) ([]byte, error) {
	return bs.stateDB.ReadObject(contractName, key)
}

// SelectObject returns an iterator that contains all the key-values between given key ranges.
// startKey is included in the results and limit is excluded.
func (bs *BlockStoreImpl) SelectObject(contractName string, startKey []byte, limit []byte) (
	protocol.StateIterator, error) {
	return bs.stateDB.SelectObject(contractName, startKey, limit)
}
func (bs *BlockStoreImpl) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	txs, err := bs.historyDB.GetHistoryForKey(contractName, key)
	if err != nil {
		return nil, err
	}
	return types.NewHistoryIterator(contractName, key, txs, bs.resultDB, bs.blockDB), nil
}
func (bs *BlockStoreImpl) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
	txs, err := bs.historyDB.GetAccountTxHistory(accountId)
	if err != nil {
		return nil, err
	}
	return types.NewTxHistoryIterator(txs, bs.blockDB), nil
}
func (bs *BlockStoreImpl) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
	txs, err := bs.historyDB.GetContractTxHistory(contractName)
	if err != nil {
		return nil, err
	}
	return types.NewTxHistoryIterator(txs, bs.blockDB), nil
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
func (bs *BlockStoreImpl) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	var (
		rwSet      *commonPb.TxRWSet
		err        error
		isArchived bool
	)

	if rwSet, err = bs.resultDB.GetTxRWSet(txId); err != nil {
		return nil, err
	}

	if rwSet == nil {
		if isArchived, err = bs.blockDB.TxArchived(txId); err != nil {
			return nil, err
		} else if isArchived {
			return nil, archive.ArchivedRWSetError
		}
	}

	return rwSet, err
}

// GetTxRWSetsByHeight returns all the rwsets corresponding to the block,
// or returns nil if zhe block does not exist
func (bs *BlockStoreImpl) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
	blockStoreInfo, err := bs.blockDB.GetFilteredBlock(height)
	if err != nil || blockStoreInfo == nil {
		return nil, err
	}
	var txRWSets = make([]*commonPb.TxRWSet, len(blockStoreInfo.TxIds))
	for i, txId := range blockStoreInfo.TxIds {

		txRWSet, err := bs.GetTxRWSet(txId)
		if err != nil {
			return nil, err
		}
		if txRWSet == nil { //数据库未找到记录，这不正常，记录日志，初始化空实例
			bs.logger.Errorf("not found rwset data in database by txid=%d, please check database", txId)
			txRWSet = &commonPb.TxRWSet{}
		}
		txRWSets[i] = txRWSet
		bs.logger.Debugf("getTxRWSetsByHeight, txid:%s", txId)

	}

	return txRWSets, nil
}

// GetBlockWithRWSets returns the block and all the rwsets corresponding to the block,
// or returns nil if zhe block does not exist
func (bs *BlockStoreImpl) GetBlockWithRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
	block, err := bs.GetBlock(height)
	if err != nil {
		return nil, err
	} else if block == nil {
		return nil, nil
	}
	var blockWithRWSets storePb.BlockWithRWSet
	blockWithRWSets.Block = block

	//var batchWG sync.WaitGroup
	//batchWG.Add(len(block.Txs))
	//errsChan := make(chan error, len(block.Txs))
	blockWithRWSets.TxRWSets = make([]*commonPb.TxRWSet, len(block.Txs))
	for i, tx := range block.Txs {
		//used to limit the num of concurrency goroutine
		//bs.workersSemaphore.Acquire(context.Background(), 1)
		//go func(i int, tx *commonPb.Transaction) {
		//	defer bs.workersSemaphore.Release(1)
		//	defer batchWG.Done()
		txRWSet, err := bs.GetTxRWSet(tx.Payload.TxId)
		if err != nil {
			return nil, err
		}
		if txRWSet == nil { //数据库未找到记录，这不正常，记录日志，初始化空实例
			bs.logger.Errorf("not found rwset data in database by txid=%d, please check database", tx.Payload.TxId)
			txRWSet = &commonPb.TxRWSet{}
		}
		blockWithRWSets.TxRWSets[i] = txRWSet
		//}
	}
	//batchWG.Wait()
	//if len(errsChan) > 0 {
	//	return nil, <-errsChan
	//}
	return &blockWithRWSets, nil
}

// GetDBHandle returns the database handle for  given dbName(chainId)
func (bs *BlockStoreImpl) GetDBHandle(dbName string) protocol.DBHandle {
	return bs.commonDB
}

// Close is used to close database
func (bs *BlockStoreImpl) Close() error {
	bs.blockDB.Close()
	bs.stateDB.Close()
	if !bs.storeConfig.DisableHistoryDB && bs.historyDB != nil {
		bs.historyDB.Close()
	}
	if !bs.storeConfig.DisableContractEventDB && bs.contractEventDB != nil {
		if parseEngineType(bs.storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
			bs.storeConfig.ContractEventDbConfig.Provider == localconf.DbConfig_Provider_Sql {
			bs.contractEventDB.Close()
		} else {
			return errors.New("contract event db config err")
		}
	}
	if !bs.storeConfig.DisableResultDB && bs.resultDB != nil {
		bs.resultDB.Close()
	}
	bs.wal.Close()
	bs.commonDB.Close()
	bs.logger.Debug("close all database and bin log")
	return nil
}

// recover checks savepoint and recommit lost block
func (bs *BlockStoreImpl) recover() error {
	var logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, resultSavepoint, contractEventSavepoint uint64
	var err error
	if logSavepoint, err = bs.getLastSavepoint(); err != nil {
		return err
	}
	if blockSavepoint, err = bs.blockDB.GetLastSavepoint(); err != nil {
		return err
	}
	if stateSavepoint, err = bs.stateDB.GetLastSavepoint(); err != nil {
		return err
	}
	if !bs.storeConfig.DisableHistoryDB {
		if historySavepoint, err = bs.historyDB.GetLastSavepoint(); err != nil {
			return err
		}
	}
	if !bs.storeConfig.DisableResultDB {
		if resultSavepoint, err = bs.resultDB.GetLastSavepoint(); err != nil {
			return err
		}
	}
	if !bs.storeConfig.DisableContractEventDB {
		if parseEngineType(bs.storeConfig.ContractEventDbConfig.SqlDbConfig.SqlDbType) == types.MySQL &&
			bs.storeConfig.ContractEventDbConfig.Provider == localconf.DbConfig_Provider_Sql {
			if contractEventSavepoint, err = bs.contractEventDB.GetLastSavepoint(); err != nil {
				return err
			}
		} else {
			return errors.New("contract event db config err")
		}
	}

	bs.logger.Debugf("recover checking, savepoint: wal[%d] blockDB[%d] stateDB[%d] historyDB[%d] contractEventDB[%d]",
		logSavepoint, blockSavepoint, stateSavepoint, historySavepoint, contractEventSavepoint)
	//recommit blockdb
	if err := bs.recoverBlockDB(blockSavepoint, logSavepoint); err != nil {
		return err
	}

	//recommit statedb
	if err := bs.recoverStateDB(stateSavepoint, logSavepoint); err != nil {
		return err
	}

	if !bs.storeConfig.DisableHistoryDB {
		//recommit historydb
		if err := bs.recoverHistoryDB(stateSavepoint, logSavepoint); err != nil {
			return err
		}
	}
	if !bs.storeConfig.DisableResultDB {
		//recommit resultdb
		if err := bs.recoverResultDB(resultSavepoint, logSavepoint); err != nil {
			return err
		}
	}
	//recommit contract event db
	if !bs.storeConfig.DisableContractEventDB {
		if err := bs.recoverContractEventDB(contractEventSavepoint, logSavepoint); err != nil {
			return err
		}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverBlockDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[BlockDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		err = bs.blockDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverStateDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[StateDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		err = bs.stateDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverContractEventDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[ContractEventDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}

		err = bs.contractEventDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverHistoryDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[HistoryDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}

		err = bs.historyDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			return err
		}
		// delete block from wal after recover
		//err = bs.deleteBlockFromLog(height)
		//if err != nil {
		//	bs.logger.Warnf("recover, failed to clean wal, block[%d]", height)
		//}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverResultDB(currentHeight uint64, savePoint uint64) error {
	height := bs.calculateRecoverHeight(currentHeight, savePoint)
	for ; height <= savePoint; height++ {
		bs.logger.Infof("[HistoryDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithSerializedInfo, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		err = bs.resultDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			return err
		}
		// delete block from wal after recover
		//err = bs.deleteBlockFromLog(height)
		//if err != nil {
		//	bs.logger.Warnf("recover, failed to clean wal, block[%d]", height)
		//}
	}
	return nil
}

func (bs *BlockStoreImpl) writeLog(blockHeight uint64, bytes []byte) error {
	// wal log, index increase from 1, while blockHeight increase form 0
	return bs.wal.Write(blockHeight+1, bytes)
}

func (bs *BlockStoreImpl) getLastSavepoint() (uint64, error) {
	lastIndex, err := bs.wal.LastIndex()
	if err != nil {
		return 0, err
	}
	if lastIndex == 0 {
		return 0, nil
	}
	return lastIndex - 1, nil
}

func (bs *BlockStoreImpl) getBlockFromLog(num uint64) (*serialization.BlockWithSerializedInfo, error) {
	index := num + 1
	bytes, err := bs.wal.Read(index)
	if err != nil {
		bs.logger.Errorf("read log failed, err:%s", err)
		return nil, err
	}
	return serialization.DeserializeBlock(bytes)
}

func (bs *BlockStoreImpl) deleteBlockFromLog(num uint64) error {
	index := num + 1
	//delete block from log every 100 block
	if (index % 100) != 0 {
		return nil
	}
	lastBlockNum := ((index - 1) / 100) * 100
	if lastBlockNum == 0 {
		return nil
	}
	return bs.wal.TruncateFront(lastBlockNum)
}

//func (bs *BlockStoreImpl) construcBlockNumKey(blockNum uint64) []byte {
//	blkNumBytes := bs.encodeBlockNum(blockNum)
//	return append([]byte{logDBBlockKeyPrefix}, blkNumBytes...)
//}

//func (bs *BlockStoreImpl) encodeBlockNum(blockNum uint64) []byte {
//	return proto.EncodeVarint(blockNum)
//}

//QuerySingle 不在事务中，直接查询状态数据库，返回一行结果
func (bs *BlockStoreImpl) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	return bs.stateDB.QuerySingle(contractName, sql, values...)
}

//QueryMulti 不在事务中，直接查询状态数据库，返回多行结果
func (bs *BlockStoreImpl) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	return bs.stateDB.QueryMulti(contractName, sql, values...)
}

//ExecDdlSql execute DDL SQL in a contract
func (bs *BlockStoreImpl) ExecDdlSql(contractName, sql string) error {
	return bs.stateDB.ExecDdlSql(contractName, sql)
}

//BeginDbTransaction 启用一个事务
func (bs *BlockStoreImpl) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return bs.stateDB.BeginDbTransaction(txName)
}

//GetDbTransaction 根据事务名，获得一个已经启用的事务
func (bs *BlockStoreImpl) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	return bs.stateDB.GetDbTransaction(txName)

}

//CommitDbTransaction 提交一个事务
func (bs *BlockStoreImpl) CommitDbTransaction(txName string) error {
	return bs.stateDB.CommitDbTransaction(txName)

}

//RollbackDbTransaction 回滚一个事务
func (bs *BlockStoreImpl) RollbackDbTransaction(txName string) error {
	return bs.stateDB.RollbackDbTransaction(txName)
}

func (bs *BlockStoreImpl) calculateRecoverHeight(currentHeight uint64, savePoint uint64) uint64 {
	height := currentHeight + 1
	if savePoint == 0 && currentHeight == 0 {
		//check whether has genesis block
		if data, _ := bs.wal.Read(1); len(data) > 0 {
			height = height - 1
		}
	}

	return height
}

func (bs *BlockStoreImpl) InitArchiveMgr(chainId string) error {
	if bs.isSupportArchive() {
		archiveMgr, err := archive.NewArchiveMgr(chainId, bs.blockDB, bs.resultDB, bs.storeConfig, bs.logger)
		if err != nil {
			return err
		}

		bs.ArchiveMgr = archiveMgr
	}

	return nil
}

func (bs *BlockStoreImpl) isSupportArchive() bool {
	return bs.storeConfig.BlockDbConfig.IsKVDB() && bs.storeConfig.ResultDbConfig.IsKVDB()
}