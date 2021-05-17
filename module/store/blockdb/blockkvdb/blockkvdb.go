/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockkvdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/store/archive"
	"github.com/syndtr/goleveldb/leveldb/util"

	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/sync/semaphore"
)

const (
	blockNumIdxKeyPrefix     = 'n'
	blockHashIdxKeyPrefix    = 'h'
	txIDIdxKeyPrefix         = 't'
	txConfirmedTimeKeyPrefix = 'c'
	blockTxIDIdxKeyPrefix    = 'b'
	lastBlockNumKeyStr       = "lastBlockNumKey"
	lastConfigBlockNumKey    = "lastConfigBlockNumKey"
	archivedPivotKey         = "archivedPivotKey"
)

const (
	blockDBName = ""
)

var (
	ValueNotFoundError = errors.New("value not found")
)

// BlockKvDB provider a implementation of `blockdb.BlockDB`
// This implementation provides a key-value based data model
type BlockKvDB struct {
	DbHandle         protocol.DBHandle
	WorkersSemaphore *semaphore.Weighted
	Cache            *cache.StoreCacheMgr
	archivedPivot    uint64

	Logger protocol.Logger
}

func (b *BlockKvDB) SaveBlockHeader(header *commonPb.BlockHeader) error {
	heightKey := constructBlockNumKey(uint64(header.BlockHeight))
	data, _ := header.Marshal()
	return b.DbHandle.Put(heightKey, data)
}
func (b *BlockKvDB) InitGenesis(genesisBlock *serialization.BlockWithSerializedInfo) error {
	return b.CommitBlock(genesisBlock)
}

// CommitBlock commits the block and the corresponding rwsets in an atomic operation
func (b *BlockKvDB) CommitBlock(blockInfo *serialization.BlockWithSerializedInfo) error {
	batch := types.NewUpdateBatch()

	// 1. last blockInfo height
	startMarshalBlock := utils.CurrentTimeMillisSeconds()
	block := blockInfo.Block
	lastBlockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lastBlockNumBytes, uint64(block.Header.BlockHeight))
	batch.Put([]byte(lastBlockNumKeyStr), lastBlockNumBytes)

	// 2. height-> blockInfo
	heightKey := constructBlockNumKey(uint64(block.Header.BlockHeight))
	batch.Put(heightKey, blockInfo.GetSerializedMeta())

	// 3. hash-> height
	hashKey := constructBlockHashKey(block.Header.BlockHash)
	batch.Put(hashKey, heightKey)

	// 4. txid -> tx,  txid -> blockHeight
	txConfirmedTime := make([]byte, 8)
	binary.BigEndian.PutUint64(txConfirmedTime, uint64(block.Header.BlockTimestamp))
	startPrepareTxs := utils.CurrentTimeMillisSeconds()
	for index, txBytes := range blockInfo.GetSerializedTxs() {
		tx := blockInfo.Block.Txs[index]
		txIdKey := constructTxIDKey(tx.Header.TxId)
		batch.Put(txIdKey, txBytes)

		blockTxIdKey := constructBlockTxIDKey(tx.Header.TxId)
		batch.Put(blockTxIdKey, heightKey)
		b.Logger.Debugf("chain[%s]: blockInfo[%d] batch transaction index[%d] txid[%s]",
			block.Header.ChainId, block.Header.BlockHeight, index, tx.Header.TxId)
	}
	elapsedPrepareTxs := utils.CurrentTimeMillisSeconds() - startPrepareTxs

	// last configBlock height
	if utils.IsConfBlock(block) {
		batch.Put([]byte(lastConfigBlockNumKey), heightKey)
		b.Logger.Infof("chain[%s]: commit config blockInfo[%d]", block.Header.ChainId, block.Header.BlockHeight)
	}

	startCommitBlock := utils.CurrentTimeMillisSeconds()
	err := b.writeBatch(block.Header.BlockHeight, batch)
	if err != nil {
		return err
	}
	elapsedCommitBlock := utils.CurrentTimeMillisSeconds() - startCommitBlock
	b.Logger.Infof("chain[%s]: commit blockInfo[%d] time used (prepare_txs:%d write_batch:%d, total:%d)",
		block.Header.ChainId, block.Header.BlockHeight, elapsedPrepareTxs, elapsedCommitBlock,
		utils.CurrentTimeMillisSeconds()-startMarshalBlock)
	return nil
}

// GetArchivedPivot return archived pivot
func (b *BlockKvDB) GetArchivedPivot() (uint64, error) {
	heightBytes, err := b.DbHandle.Get([]byte(archivedPivotKey))
	if err != nil {
		return 0, err
	}

	// heightBytes can be nil while db do not has archive pivot, we use pivot 1 as default
	dbHeight := uint64(0)
	if heightBytes != nil {
		dbHeight = decodeBlockNumKey(heightBytes)
	}

	if dbHeight != b.archivedPivot {
		b.Logger.Warnf("DB archivedPivot:[%d] is not match using archivedPivot:[%d], use write DB overwrite it!")
		b.archivedPivot = dbHeight
	}

	return b.archivedPivot, nil
}

// SetArchivedPivot set archived pivot
func (b *BlockKvDB) SetArchivedPivot(archivedPivot uint64) error {
	batch := types.NewUpdateBatch()
	batch.Put([]byte(archivedPivotKey), constructBlockNumKey(archivedPivot))
	if err := b.DbHandle.WriteBatch(batch, false); err != nil {
		return err
	}

	b.archivedPivot = archivedPivot
	return nil
}

// ShrinkBlocks remove ranged heightKey--SerializedMeta and txid--SerializedTx from kvdb
func (b *BlockKvDB) ShrinkBlocks(startHeight uint64, endHeight uint64) error {
	block, err := b.getBlockByHeightBytes(constructBlockNumKey(endHeight))
	if err != nil {
		return err
	}

	if utils.IsConfBlock(block) {
		return archive.ConfigBlockArchiveError
	}

	batch := types.NewUpdateBatch()

	startTime := utils.CurrentTimeMillisSeconds()
	for height := endHeight; height >= startHeight; height-- {
		heightKey := constructBlockNumKey(height)
		blk, err1 := b.getBlockByHeightBytes(heightKey)
		if err1 != nil {
			return err1
		}

		if utils.IsConfBlock(blk) {
			b.Logger.Infof("skip shrink conf block: [%d]", block.Header.BlockHeight)
			continue
		}

		for _, tx := range block.Txs {
			// delete tx data
			batch.Delete(constructTxIDKey(tx.Header.TxId))
		}
	}

	beforeWrite := utils.CurrentTimeMillisSeconds()
	err = b.DbHandle.WriteBatch(batch, false)
	if err != nil {
		return err
	}

	go b.compactRange()

	writeTime := utils.CurrentTimeMillisSeconds() - beforeWrite
	b.Logger.Infof("shrink block from [%d] to [%d] time used (prepare_txs:%d write_batch:%d, total:%d)",
		startHeight, endHeight, beforeWrite-startTime, writeTime,
		utils.CurrentTimeMillisSeconds()-startTime)
	return nil
}

// RestoreBlocks restore block data from outside to kvdb: txid--SerializedTx
func (b *BlockKvDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	batch := types.NewUpdateBatch()

	startTime := utils.CurrentTimeMillisSeconds()
	for i := len(blockInfos) - 1; i >= 0; i-- {
		blockInfo := blockInfos[i]

		//check whether block can be archived
		if utils.IsConfBlock(blockInfo.Block) {
			b.Logger.Warnf("skip store conf block: [%d]", blockInfo.Block.Header.BlockHeight)
			continue
		}

		//check block hash
		sBlock, err2 := b.GetFilteredBlock(blockInfo.Block.Header.BlockHeight)
		if err2 != nil {
			return err2
		}

		if !bytes.Equal(blockInfo.Block.Header.BlockHash, sBlock.Header.BlockHash) {
			return archive.InvalidateRestoreBlocksError
		}

		//verify imported block txs
		for index, stx := range blockInfo.GetSerializedTxs() {
			// put tx data
			batch.Put(constructTxIDKey(blockInfo.Block.Txs[index].Header.TxId), stx)
		}
	}

	beforeWrite := utils.CurrentTimeMillisSeconds()
	if err := b.DbHandle.WriteBatch(batch, false); err != nil {
		return err
	}

	go b.compactRange()

	writeTime := utils.CurrentTimeMillisSeconds() - beforeWrite
	b.Logger.Infof("shrink block from [%d] to [%d] time used (prepare_txs:%d write_batch:%d, total:%d)",
		blockInfos[len(blockInfos)-1].Block.Header.BlockHeight, blockInfos[0].Block.Header.BlockHeight, beforeWrite-startTime, writeTime,
		utils.CurrentTimeMillisSeconds()-startTime)
	return nil
}

// BlockExists returns true if the block hash exist, or returns false if none exists.
func (b *BlockKvDB) BlockExists(blockHash []byte) (bool, error) {
	hashKey := constructBlockHashKey(blockHash)
	return b.has(hashKey)
}

// GetBlockByHash returns a block given it's hash, or returns nil if none exists.
func (b *BlockKvDB) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	hashKey := constructBlockHashKey(blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return nil, err
	}

	if heightBytes == nil {
		return nil, ValueNotFoundError
	}

	archivedPivot, err := b.GetArchivedPivot()
	if err != nil {
		return nil, err
	}

	if decodeBlockNumKey(heightBytes) < archivedPivot {
		return nil, archive.ArchivedBlockError
	}

	return b.getBlockByHeightBytes(heightBytes)
}

// GetHeightByHash returns a block height given it's hash, or returns nil if none exists.
func (b *BlockKvDB) GetHeightByHash(blockHash []byte) (uint64, error) {
	hashKey := constructBlockHashKey(blockHash)
	heightBytes, err := b.get(hashKey)
	if err != nil {
		return 0, err
	}

	if heightBytes == nil {
		return 0, ValueNotFoundError
	}

	return decodeBlockNumKey(heightBytes), nil
}

// GetBlockMateByHash returns a block metadata given it's hash, or returns nil if none exists.
func (b *BlockKvDB) GetBlockMateByHash(blockHash []byte) ([]byte, error) {
	height, err := b.GetHeightByHash(blockHash)
	if err != nil {
		return nil, err
	}

	vBytes, err := b.get(constructBlockNumKey(height))
	if err != nil {
		return nil, err
	}

	if vBytes == nil {
		return nil, ValueNotFoundError
	}

	return vBytes, nil
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
func (b *BlockKvDB) GetBlock(height int64) (*commonPb.Block, error) {
	archivedPivot, err := b.GetArchivedPivot()
	if err != nil {
		return nil, err
	}

	if uint64(height) < archivedPivot {
		return nil, archive.ArchivedBlockError
	}
	heightBytes := constructBlockNumKey(uint64(height))
	return b.getBlockByHeightBytes(heightBytes)
}

// GetLastBlock returns the last block.
func (b *BlockKvDB) GetLastBlock() (*commonPb.Block, error) {
	num, err := b.GetLastSavepoint()
	if err != nil {
		return nil, err
	}

	heightBytes := constructBlockNumKey(num)
	return b.getBlockByHeightBytes(heightBytes)
}

// GetLastConfigBlock returns the last config block.
func (b *BlockKvDB) GetLastConfigBlock() (*commonPb.Block, error) {
	heightKey, err := b.get([]byte(lastConfigBlockNumKey))
	if err != nil {
		return nil, err
	}
	b.Logger.Debugf("configBlock height:%v", heightKey)
	return b.getBlockByHeightBytes(heightKey)
}

// GetFilteredBlock returns a filtered block given it's block height, or return nil if none exists.
func (b *BlockKvDB) GetFilteredBlock(height int64) (*storePb.SerializedBlock, error) {
	heightKey := constructBlockNumKey(uint64(height))
	bytes, err := b.get(heightKey)
	if err != nil {
		return nil, err
	} else if bytes == nil {
		return nil, nil
	}
	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(bytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}
	return &blockStoreInfo, nil
}

// GetLastSavepoint reurns the last block height
func (b *BlockKvDB) GetLastSavepoint() (uint64, error) {
	bytes, err := b.get([]byte(lastBlockNumKeyStr))
	if err != nil {
		return 0, err
	} else if bytes == nil {
		return 0, nil
	}

	num := binary.BigEndian.Uint64(bytes)
	return num, nil
}

// GetBlockByTx returns a block which contains a tx.
func (b *BlockKvDB) GetBlockByTx(txId string) (*commonPb.Block, error) {
	blockTxIdKey := constructBlockTxIDKey(txId)
	heightBytes, err := b.get(blockTxIdKey)
	if err != nil {
		return nil, err
	}
	return b.getBlockByHeightBytes(heightBytes)
}

// GetTxHeight retrieves a transaction height by txid, or returns nil if none exists.
func (b *BlockKvDB) GetTxHeight(txId string) (uint64, error) {
	blockTxIdKey := constructBlockTxIDKey(txId)
	vBytes, err := b.get(blockTxIdKey)
	if err != nil {
		return 0, err
	}

	if vBytes == nil {
		return 0, ValueNotFoundError
	}

	return decodeBlockNumKey(vBytes), nil
}

// GetTx retrieves a transaction by txid, or returns nil if none exists.
func (b *BlockKvDB) GetTx(txId string) (*commonPb.Transaction, error) {
	txIdKey := constructTxIDKey(txId)
	bytes, err := b.get(txIdKey)
	if err != nil {
		return nil, err
	} else if len(bytes) == 0 {
		isArchived, erra := b.TxArchived(txId)
		if erra == nil && isArchived {
			return nil, archive.ArchivedTxError
		}

		return nil, nil
	}

	var tx commonPb.Transaction
	err = proto.Unmarshal(bytes, &tx)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}
func (b *BlockKvDB) GetTxWithBlockInfo(txId string) (*commonPb.TransactionInfo, error) {
	txIdKey := constructTxIDKey(txId)
	bytes, err := b.get(txIdKey)
	if err != nil {
		return nil, err
	} else if len(bytes) == 0 {
		return nil, nil
	}

	var tx commonPb.Transaction
	err = proto.Unmarshal(bytes, &tx)
	if err != nil {
		return nil, err
	}
	//TODO devin add block info
	return &commonPb.TransactionInfo{Transaction: &tx}, nil
}

// TxExists returns true if the tx exist, or returns false if none exists.
func (b *BlockKvDB) TxExists(txId string) (bool, error) {
	txHashKey := constructTxIDKey(txId)
	exist, err := b.has(txHashKey)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// TxArchived returns true if the tx archived, or returns false.
func (b *BlockKvDB) TxArchived(txId string) (bool, error) {
	heightBytes, err := b.DbHandle.Get(constructBlockTxIDKey(txId))
	if err != nil {
		return false, err
	}

	if heightBytes == nil {
		return false, ValueNotFoundError
	}

	archivedPivot, err := b.GetArchivedPivot()
	if err != nil {
		return false, err
	}

	if decodeBlockNumKey(heightBytes) < archivedPivot {
		return true, nil
	}

	return false, nil
}

// GetTxConfirmedTime returns the confirmed time of a given tx
func (b *BlockKvDB) GetTxConfirmedTime(txId string) (int64, error) {
	txConfirmedTimeKey := constructTxConfirmedTimeKey(txId)
	bytes, err := b.get(txConfirmedTimeKey)
	if err != nil {
		return 0, err
	} else if len(bytes) == 0 {
		return -1, nil
	}
	confirmedTime := binary.BigEndian.Uint64(bytes)
	return int64(confirmedTime), nil
}

// Close is used to close database
func (b *BlockKvDB) Close() {
	b.Logger.Info("close block kv db")
	b.DbHandle.Close()
}

func (b *BlockKvDB) getBlockByHeightBytes(height []byte) (*commonPb.Block, error) {
	if height == nil {
		return nil, nil
	}
	bytes, err := b.get(height)
	if err != nil {
		return nil, err
	} else if bytes == nil {
		return nil, nil
	}

	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(bytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}

	var block = commonPb.Block{
		Header:         blockStoreInfo.Header,
		Dag:            blockStoreInfo.Dag,
		AdditionalData: blockStoreInfo.AdditionalData,
	}

	//var batchWG sync.WaitGroup
	//batchWG.Add(len(blockStoreInfo.TxIds))
	//errsChan := make(chan error, len(blockStoreInfo.TxIds))
	block.Txs = make([]*commonPb.Transaction, len(blockStoreInfo.TxIds))
	for index, txid := range blockStoreInfo.TxIds {
		//used to limit the num of concurrency goroutine
		//b.WorkersSemaphore.Acquire(context.Background(), 1)
		//go func(i int, txid string) {
		//	defer b.WorkersSemaphore.Release(1)
		//	defer batchWG.Done()
		tx, err := b.GetTx(txid)
		if err != nil {
			//errsChan <- err
			return nil, err
		}
		block.Txs[index] = tx
		//}(index, txid)
	}
	//batchWG.Wait()
	//if len(errsChan) > 0 {
	//	return nil, <-errsChan
	//}
	b.Logger.Debugf("chain[%s]: get block[%d] with transactions[%d]",
		block.Header.ChainId, block.Header.BlockHeight, len(block.Txs))
	return &block, nil
}

func (b *BlockKvDB) writeBatch(blockHeight int64, batch protocol.StoreBatcher) error {
	//update cache
	b.Cache.AddBlock(blockHeight, batch)

	startWriteBatchTime := utils.CurrentTimeMillisSeconds()
	err := b.DbHandle.WriteBatch(batch, false)
	endWriteBatchTime := utils.CurrentTimeMillisSeconds()
	b.Logger.Infof("write block db, block[%d], time used:%d",
		blockHeight, endWriteBatchTime-startWriteBatchTime)

	if err != nil {
		panic(fmt.Sprintf("Error writting leveldb: %s", err))
	}
	//db committed, clean cache
	b.Cache.DelBlock(blockHeight)

	return nil
}

func (b *BlockKvDB) get(key []byte) ([]byte, error) {
	//get from cache
	value, exist := b.Cache.Get(string(key))
	if exist {
		b.Logger.Debugf("get content: [%x] by [%d] in cache", value, key)
		return value, nil
	}
	//get from database
	val, err := b.DbHandle.Get(key)
	return val, err
}

func (b *BlockKvDB) has(key []byte) (bool, error) {
	//check has from cache
	isDelete, exist := b.Cache.Has(string(key))
	if exist {
		return !isDelete, nil
	}
	return b.DbHandle.Has(key)
}

func constructBlockNumKey(blockNum uint64) []byte {
	blkNumBytes := encodeBlockNum(blockNum)
	return append([]byte{blockNumIdxKeyPrefix}, blkNumBytes...)
}

func constructBlockHashKey(blockHash []byte) []byte {
	return append([]byte{blockHashIdxKeyPrefix}, blockHash...)
}

func constructTxIDKey(txId string) []byte {
	return append([]byte{txIDIdxKeyPrefix}, txId...)
}

func constructTxConfirmedTimeKey(txId string) []byte {
	return append([]byte{txConfirmedTimeKeyPrefix}, txId...)
}

func constructBlockTxIDKey(txID string) []byte {
	return append([]byte{blockTxIDIdxKeyPrefix}, []byte(txID)...)
}

func encodeBlockNum(blockNum uint64) []byte {
	return proto.EncodeVarint(blockNum)
}

func decodeBlockNumKey(blkNumBytes []byte) uint64 {
	blkNumBytes = blkNumBytes[len([]byte{blockNumIdxKeyPrefix}):]
	return decodeBlockNum(blkNumBytes)
}

func decodeBlockNum(blockNumBytes []byte) uint64 {
	blockNum, _ := proto.DecodeVarint(blockNumBytes)
	return blockNum
}

func (b *BlockKvDB) compactRange() {
	//trigger level compact
	if err := b.DbHandle.CompactRange(util.Range{
		Start: []byte(""),
		Limit: []byte(""),
	}); err != nil {
		b.Logger.Warnf("kvdb level compact failed: %v", err)
	}
}

