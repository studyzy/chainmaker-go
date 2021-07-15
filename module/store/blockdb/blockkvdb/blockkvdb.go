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

	"chainmaker.org/chainmaker-go/store/archive"
	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"chainmaker.org/chainmaker-go/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	storePb "chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/protocol"
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
	batch.Put(heightKey, blockInfo.SerializedMeta)

	// 3. hash-> height
	hashKey := constructBlockHashKey(block.Header.BlockHash)
	batch.Put(hashKey, heightKey)

	// 4. txid -> tx,  txid -> blockHeight
	txConfirmedTime := make([]byte, 8)
	binary.BigEndian.PutUint64(txConfirmedTime, uint64(block.Header.BlockTimestamp))
	startPrepareTxs := utils.CurrentTimeMillisSeconds()
	for index, txBytes := range blockInfo.SerializedTxs {
		tx := blockInfo.Block.Txs[index]
		txIdKey := constructTxIDKey(tx.Payload.TxId)
		batch.Put(txIdKey, txBytes)

		blockTxIdKey := constructBlockTxIDKey(tx.Payload.TxId)
		batch.Put(blockTxIdKey, heightKey)
		b.Logger.Debugf("chain[%s]: blockInfo[%d] batch transaction index[%d] txid[%s]",
			block.Header.ChainId, block.Header.BlockHeight, index, tx.Payload.TxId)
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


// ShrinkBlocks remove ranged txid--SerializedTx from kvdb
func (b *BlockKvDB) ShrinkBlocks(startHeight uint64, endHeight uint64) (map[uint64][]string, error) {
	var (
		block *commonPb.Block
		err   error
	)

	if block, err = b.getBlockByHeightBytes(constructBlockNumKey(endHeight)); err != nil {
		return nil, err
	}

	if utils.IsConfBlock(block) {
		return nil, archive.ConfigBlockArchiveError
	}

	txIdsMap := make(map[uint64][]string)
	startTime := utils.CurrentTimeMillisSeconds()
	for height := startHeight; height <= endHeight; height++ {
		heightKey := constructBlockNumKey(height)
		blk, err1 := b.getBlockByHeightBytes(heightKey)
		if err1 != nil {
			return nil, err1
		}

		if utils.IsConfBlock(blk) {
			b.Logger.Infof("skip shrink conf block: [%d]", block.Header.BlockHeight)
			continue
		}

		batch := types.NewUpdateBatch()
		txIds := make([]string, 0, len(blk.Txs))
		for _, tx := range blk.Txs {
			// delete tx data
			batch.Delete(constructTxIDKey(tx.Payload.TxId))
			txIds = append(txIds, tx.Payload.TxId)
		}
		txIdsMap[height] = txIds
		//set archivedPivotKey to db
		batch.Put([]byte(archivedPivotKey), constructBlockNumKey(height))
		if err = b.DbHandle.WriteBatch(batch, false); err != nil {
			return nil, err
		}

		b.archivedPivot = height
	}

	go b.compactRange()

	usedTime := utils.CurrentTimeMillisSeconds() - startTime
	b.Logger.Infof("shrink block from [%d] to [%d] time used: %d",
		startHeight, endHeight, usedTime)
	return txIdsMap, nil
}

// RestoreBlocks restore block data from outside to kvdb: txid--SerializedTx
func (b *BlockKvDB) RestoreBlocks(blockInfos []*serialization.BlockWithSerializedInfo) error {
	startTime := utils.CurrentTimeMillisSeconds()
	archivePivot := uint64(0)
	for i := len(blockInfos) - 1; i >= 0; i-- {
		blockInfo := blockInfos[i]

		//check whether block can be archived
		if utils.IsConfBlock(blockInfo.Block) {
			b.Logger.Infof("skip store conf block: [%d]", blockInfo.Block.Header.BlockHeight)
			continue
		}

		//check block hash
		sBlock, err := b.GetFilteredBlock(blockInfo.Block.Header.BlockHeight)
		if err != nil {
			return err
		}

		if !bytes.Equal(blockInfo.Block.Header.BlockHash, sBlock.Header.BlockHash) {
			return archive.InvalidateRestoreBlocksError
		}

		batch := types.NewUpdateBatch()
		//verify imported block txs
		for index, stx := range blockInfo.SerializedTxs {
			// put tx data
			batch.Put(constructTxIDKey(blockInfo.Block.Txs[index].Payload.TxId), stx)
		}

		archivePivot, err = b.getNextArchivePivot(blockInfo.Block)
		if err != nil {
			return err
		}

		batch.Put([]byte(archivedPivotKey), constructBlockNumKey(archivePivot))
		err = b.DbHandle.WriteBatch(batch, false)
		if err != nil {
			return err
		}
		b.archivedPivot = archivePivot
	}

	go b.compactRange()

	usedTime := utils.CurrentTimeMillisSeconds() - startTime
	b.Logger.Infof("shrink block from [%d] to [%d] time used: %d",
		blockInfos[len(blockInfos)-1].Block.Header.BlockHeight, blockInfos[0].Block.Header.BlockHeight, usedTime)
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

// GetBlockHeaderByHeight returns a block header by given it's height, or returns nil if none exists.
func (b *BlockKvDB) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	vBytes, err := b.get(constructBlockNumKey(uint64(height)))
	if err != nil {
		return nil, err
	}

	if vBytes == nil {
		return nil, ValueNotFoundError
	}

	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(vBytes, &blockStoreInfo)
	if err != nil {
		return nil, err
	}

	return blockStoreInfo.Header, nil
}

// GetBlock returns a block given it's block height, or returns nil if none exists.
func (b *BlockKvDB) GetBlock(height uint64) (*commonPb.Block, error) {
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
func (b *BlockKvDB) GetFilteredBlock(height uint64) (*storePb.SerializedBlock, error) {
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
	vBytes, err := b.get(txIdKey)
	if err != nil {
		return nil, err
	} else if len(vBytes) == 0 {
		isArchived, erra := b.TxArchived(txId)
		if erra == nil && isArchived {
			return nil, archive.ArchivedTxError
		}
		return nil, nil
	}

	var tx commonPb.Transaction
	err = proto.Unmarshal(vBytes, &tx)
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

	if decodeBlockNumKey(heightBytes) <= archivedPivot {
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

	vBytes, err := b.get(height)
	if err != nil || vBytes == nil {
		return nil, err
	}

	var blockStoreInfo storePb.SerializedBlock
	err = proto.Unmarshal(vBytes, &blockStoreInfo)
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
		tx, err1 := b.GetTx(txid)
		if err1 != nil {
			if err1 == archive.ArchivedTxError {
				return nil, archive.ArchivedBlockError
			}
			//errsChan <- err
			return nil, err1
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

func (b *BlockKvDB) writeBatch(blockHeight uint64, batch protocol.StoreBatcher) error {
	//update cache
	b.Cache.AddBlock(blockHeight, batch)

	startWriteBatchTime := utils.CurrentTimeMillisSeconds()
	err := b.DbHandle.WriteBatch(batch, false)
	endWriteBatchTime := utils.CurrentTimeMillisSeconds()
	b.Logger.Infof("write block db, block[%d], time used:%d",
		blockHeight, endWriteBatchTime-startWriteBatchTime)

	if err != nil {
		panic(fmt.Sprintf("Error writing leveldb: %s", err))
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

func (b *BlockKvDB) getNextArchivePivot(pivotBlock *commonPb.Block) (uint64, error) {
	curIsConf := true
	archivedPivot := uint64(pivotBlock.Header.BlockHeight)
	for curIsConf {
		//consider restore height 1 and height 0 block
		//1. height 1: this is a config block, archivedPivot should be 0
		//2. height 1: this is not a config block, archivedPivot should be 0
		//3. height 0: archivedPivot should be 0
		if archivedPivot < 2 {
			archivedPivot = 0
			break
		}

		//we should not get block data only if it is config block
		archivedPivot = archivedPivot - 1
		_, errb := b.GetBlock(uint64(archivedPivot))
		if errb == archive.ArchivedBlockError {
			curIsConf = false
			break
		} else if errb != nil {
			return 0, errb
		}
	}
	return archivedPivot, nil
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
	for i := 1; i <= 1; i++ {
		b.Logger.Infof("Do %dst time CompactRange", i)
		if err := b.DbHandle.CompactRange(nil, nil); err != nil {
			b.Logger.Warnf("blockdb level compact failed: %v", err)
		}
		//time.Sleep(2 * time.Second)
	}
}
