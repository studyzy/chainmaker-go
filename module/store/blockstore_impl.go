/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/dbprovider"
	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/statedb"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/tidwall/wal"
	"golang.org/x/sync/semaphore"
)

const (
	logPath             = "wal"
	logDBBlockKeyPrefix = 'n'
)

// BlockStoreImpl provides an implementation of `protocal.BlockchainStore`.
type BlockStoreImpl struct {
	blockDB          blockdb.BlockDB
	stateDB          statedb.StateDB
	historyDB        historydb.HistoryDB
	wal              *wal.Log
	commonDB         dbprovider.Provider
	workersSemaphore *semaphore.Weighted
	logger           *logImpl.CMLogger
}

// NewBlockStoreImpl constructs new `BlockStoreImpl`
func NewBlockStoreImpl(chainId string,
	blockDB blockdb.BlockDB,
	stateDB statedb.StateDB,
	historyDB historydb.HistoryDB,
	commonDB dbprovider.Provider) (protocol.BlockchainStore, error) {

	walPath := filepath.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, chainId, logPath)
	writeAsync := localconf.ChainMakerConfig.StorageConfig.LogDBWriteAsync
	walOpt := &wal.Options{
		NoSync: writeAsync,
	}
	writeLog, err := wal.Open(walPath, walOpt)
	if err != nil {
		panic(fmt.Sprintf("open wal failed, path:%s, error:%s", walPath, err))
	}
	nWorkers := runtime.NumCPU()
	blockStore := &BlockStoreImpl{
		blockDB:          blockDB,
		stateDB:          stateDB,
		historyDB:        historyDB,
		wal:              writeLog,
		commonDB:         commonDB,
		workersSemaphore: semaphore.NewWeighted(int64(nWorkers)),

		logger: logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
	}
	//check savepoint and recover
	err = blockStore.recover()
	if err != nil {
		return nil, err
	}
	return blockStore, nil
}

// PutBlock commits the block and the corresponding rwsets in an atomic operation
func (bs *BlockStoreImpl) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	startPutBlock := utils.CurrentTimeMillisSeconds()
	//1. commit log
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}
	//try to add consensusArgs
	consensusArgs, err := utils.GetConsensusArgsFromBlock(block)
	if err == nil && consensusArgs.ConsensusData != nil {
		bs.logger.Debugf("add consensusArgs ConsensusData!")
		blockWithRWSet.TxRWSets = append(blockWithRWSet.TxRWSets, consensusArgs.ConsensusData)
	}
	blockBytes, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
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
	numBatches := 3
	var batchWG sync.WaitGroup
	batchWG.Add(numBatches)
	errsChan := make(chan error, numBatches)
	// 2.commit blockDB
	go func() {
		defer batchWG.Done()
		err := bs.blockDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write blockDB, block[%d]",
				block.Header.ChainId, block.Header.BlockHeight)
			errsChan <- err
		}
	}()

	// 3.commit stateDB
	go func() {
		defer batchWG.Done()
		err := bs.stateDB.CommitBlock(blockWithRWSet)
		if err != nil {
			bs.logger.Errorf("chain[%s] failed to write stateDB, block[%d]",
				block.Header.ChainId, block.Header.BlockHeight)
			errsChan <- err
		}
	}()

	// 4.commit historyDB
	if !localconf.ChainMakerConfig.StorageConfig.DisableHistoryDB {
		go func() {
			defer batchWG.Done()
			err := bs.historyDB.CommitBlock(blockWithSerializedInfo)
			if err != nil {
				bs.logger.Errorf("chain[%s] failed to write historyDB, block[%d]",
					block.Header.ChainId, block.Header.BlockHeight)
				errsChan <- err
			}
		}()
	} else {
		batchWG.Done()
	}

	batchWG.Wait()
	if len(errsChan) > 0 {
		return <-errsChan
	}
	elapsedCommitBlock := utils.CurrentTimeMillisSeconds() - startCommitBlock

	// 5. clean wal, delete block and rwset after commit
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

// BlockExists returns true if the black hash exist, or returns false if none exists.
func (bs *BlockStoreImpl) BlockExists(blockHash []byte) (bool, error) {
	return bs.blockDB.BlockExists(blockHash)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
func (bs *BlockStoreImpl) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	return bs.blockDB.GetBlockByHash(blockHash)
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
func (bs *BlockStoreImpl) GetBlock(height int64) (*commonPb.Block, error) {
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

// GetBlockByTx returns a block which contains a tx.
func (bs *BlockStoreImpl) GetBlockByTx(txId string) (*commonPb.Block, error) {
	return bs.blockDB.GetBlockByTx(txId)
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
func (bs *BlockStoreImpl) GetTx(txId string) (*commonPb.Transaction, error) {
	return bs.blockDB.GetTx(txId)
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
func (bs *BlockStoreImpl) SelectObject(contractName string, startKey []byte, limit []byte) protocol.Iterator {
	return bs.stateDB.SelectObject(contractName, startKey, limit)
}

// GetTxRWSet returns an txRWSet for given txId, or returns nil if none exists.
func (bs *BlockStoreImpl) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	return bs.historyDB.GetTxRWSet(txId)
}

// GetTxRWSetsByHeight returns all the rwsets corresponding to the block,
// or returns nil if zhe block does not exist
func (bs *BlockStoreImpl) GetTxRWSetsByHeight(height int64) ([]*commonPb.TxRWSet, error) {
	blockStoreInfo, err := bs.blockDB.GetFilteredBlock(height)
	if err != nil {
		return nil, err
	}
	var txRWSets []*commonPb.TxRWSet
	var batchWG sync.WaitGroup
	batchWG.Add(len(blockStoreInfo.TxIds))
	errsChan := make(chan error, len(blockStoreInfo.TxIds))
	txRWSets = make([]*commonPb.TxRWSet, len(blockStoreInfo.TxIds))
	for index, txId := range blockStoreInfo.TxIds {
		bs.workersSemaphore.Acquire(context.Background(), 1)
		go func(i int, txId string) {
			defer bs.workersSemaphore.Release(1)
			defer batchWG.Done()
			txRWSet, err := bs.GetTxRWSet(txId)
			if err != nil {
				errsChan <- err
			}
			txRWSets[i] = txRWSet
			bs.logger.Debugf("getTxRWSetsByHeight, txid:%s", txId)
		}(index, txId)
	}
	batchWG.Wait()
	if len(errsChan) > 0 {
		return nil, <-errsChan
	}
	return txRWSets, nil
}

// GetBlockWithRWSets returns the block and all the rwsets corresponding to the block,
// or returns nil if zhe block does not exist
func (bs *BlockStoreImpl) GetBlockWithRWSets(height int64) (*storePb.BlockWithRWSet, error) {
	block, err := bs.GetBlock(height)
	if err != nil {
		return nil, err
	} else if block == nil {
		return nil, nil
	}
	var blockWithRWSets storePb.BlockWithRWSet
	blockWithRWSets.Block = block

	var batchWG sync.WaitGroup
	batchWG.Add(len(block.Txs))
	errsChan := make(chan error, len(block.Txs))
	blockWithRWSets.TxRWSets = make([]*commonPb.TxRWSet, len(block.Txs))
	for index, tx := range block.Txs {
		//used to limit the num of concurrency goroutine
		bs.workersSemaphore.Acquire(context.Background(), 1)
		go func(i int, tx *commonPb.Transaction) {
			defer bs.workersSemaphore.Release(1)
			defer batchWG.Done()
			txRWSet, err := bs.GetTxRWSet(tx.Header.TxId)
			if err != nil {
				errsChan <- err
			}
			blockWithRWSets.TxRWSets[i] = txRWSet
		}(index, tx)
	}
	batchWG.Wait()
	if len(errsChan) > 0 {
		return nil, <-errsChan
	}
	return &blockWithRWSets, nil
}

// GetDBHandle returns the database handle for  given dbName(chainId)
func (bs *BlockStoreImpl) GetDBHandle(dbName string) protocol.DBHandle {
	return bs.commonDB.GetDBHandle(dbName)
}

// Close is used to close database
func (bs *BlockStoreImpl) Close() error {
	bs.blockDB.Close()
	bs.stateDB.Close()
	bs.historyDB.Close()
	bs.wal.Close()
	bs.commonDB.Close()
	return nil
}

// recover checks savepoint and recommit lost block
func (bs *BlockStoreImpl) recover() error {
	var logSavepoint, blockSavepoint, stateSavepoint, historySavepoint uint64
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
	if historySavepoint, err = bs.historyDB.GetLastSavepoint(); err != nil {
		return err
	}

	bs.logger.Debugf("recover checking, savepoint: wal[%d] blockDB[%d] stateDB[%d] historyDB[%d]",
		logSavepoint, blockSavepoint, stateSavepoint, historySavepoint)
	//recommit blockdb
	if err := bs.recoverBlockDB(blockSavepoint, logSavepoint); err != nil {
		return nil
	}

	//recommit statedb
	if err := bs.recoverStateDB(stateSavepoint, logSavepoint); err != nil {
		return nil
	}

	if !localconf.ChainMakerConfig.StorageConfig.DisableHistoryDB {
		//recommit historydb
		if err := bs.recoverHistoryDB(stateSavepoint, logSavepoint); err != nil {
			return nil
		}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverBlockDB(currentHeight uint64, savePoint uint64) error {
	for height := currentHeight + 1; height <= savePoint; height++ {
		bs.logger.Infof("[BlockDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithRWSet, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		_, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
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
	for height := currentHeight + 1; height <= savePoint; height++ {
		bs.logger.Infof("[StateDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithRWSet, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		err = bs.stateDB.CommitBlock(blockWithRWSet)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bs *BlockStoreImpl) recoverHistoryDB(currentHeight uint64, savePoint uint64) error {
	for height := currentHeight + 1; height <= savePoint; height++ {
		bs.logger.Infof("[HistoryDB] recommitting lost blocks, blockNum=%d, lastBlockNum=%d", height, savePoint)
		blockWithRWSet, err := bs.getBlockFromLog(height)
		if err != nil {
			return err
		}
		_, blockWithSerializedInfo, err := serialization.SerializeBlock(blockWithRWSet)
		if err != nil {
			return err
		}
		err = bs.historyDB.CommitBlock(blockWithSerializedInfo)
		if err != nil {
			return err
		}
		// delete block from wal after recover
		err = bs.deleteBlockFromLog(height)
		if err != nil {
			bs.logger.Warnf("recover, failed to clean wal, block[%d]", height)
		}
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

func (bs *BlockStoreImpl) getBlockFromLog(num uint64) (*storePb.BlockWithRWSet, error) {
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

func (bs *BlockStoreImpl) construcBlockNumKey(blockNum uint64) []byte {
	blkNumBytes := bs.encodeBlockNum(blockNum)
	return append([]byte{logDBBlockKeyPrefix}, blkNumBytes...)
}

func (bs *BlockStoreImpl) encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}
