/*
 * Copyright (C) BABEC. All rights reserved.
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package blockkvdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	storePb "chainmaker.org/chainmaker/pb-go/v2/store"

	"chainmaker.org/chainmaker-go/store/dbprovider/leveldbprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"github.com/stretchr/testify/assert"
)

var (
	logger  = &test.GoLogger{}
	chainId = "chain1"
)

func createConfigBlock(chainId string, height uint64, preConfHeight uint64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer:    &acPb.Member{MemberInfo: []byte("User1")},
			BlockType:   0,
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId:      chainId,
					TxType:       commonPb.TxType_INVOKE_CONTRACT,
					ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
				},
				Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
					Signature: []byte("sign1"),
				},
				Result: &commonPb.Result{
					Code: commonPb.TxStatusCode_SUCCESS,
					ContractResult: &commonPb.ContractResult{
						Result: []byte("ok"),
					},
				},
			},
		},
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	block.Txs[0].Payload.TxId = generateTxId(chainId, height, 0)
	block.Header.PreConfHeight = preConfHeight
	return block
}

func createBlockAndRWSets(chainId string, height uint64, txNum int, preConfHeight uint64) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        chainId,
			BlockHeight:    height,
			Proposer:       &acPb.Member{MemberInfo: []byte("User1")},
			BlockTimestamp: time.Now().UnixNano() / 1e6,
		},
		Dag:            &commonPb.DAG{},
		AdditionalData: &commonPb.AdditionalData{},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId: chainId,
				TxId:    generateTxId(chainId, height, i),
			},
			Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
				Signature: []byte("sign1"),
			},
			Result: &commonPb.Result{
				Code: commonPb.TxStatusCode_SUCCESS,
				ContractResult: &commonPb.ContractResult{
					Result: []byte("ok"),
				},
			},
		}
		block.Txs = append(block.Txs, tx)
	}

	block.Header.BlockHash = generateBlockHash(chainId, height)
	block.Header.PreConfHeight = preConfHeight
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < txNum; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		txRWset := &commonPb.TxRWSet{
			TxId: block.Txs[i].Payload.TxId,
			TxWrites: []*commonPb.TxWrite{
				{
					Key:          []byte(key),
					Value:        []byte(value),
					ContractName: "contract1",
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return block, txRWSets
}

var testChainId = "testchainid_1"
var block0 = createConfigBlock(testChainId, 0, 0)
var block1, _ = createBlockAndRWSets(testChainId, 1, 10, 0)
var block2, _ = createBlockAndRWSets(testChainId, 2, 2, 0)
var block3, _ = createBlockAndRWSets(testChainId, 3, 2, 0)
var configBlock4 = createConfigBlock(testChainId, 4, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3, 4)

func init5Blocks(db *BlockKvDB) {
	commitBlock(db, block0)
	commitBlock(db, block1)
	commitBlock(db, block2)
	commitBlock(db, block3)
	commitBlock(db, configBlock4)
	commitBlock(db, block5)
}
func commitBlock(db *BlockKvDB, block *commonPb.Block) error {
	_, bl, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	return db.CommitBlock(bl)
}

func initDb() *BlockKvDB {
	blockDB := NewBlockKvDB("test-chain", leveldbprovider.NewMemdbHandle(), logger)
	return blockDB
}

func TestBlockKvDB_GetTxWithBlockInfo(t *testing.T) {
	block := block1
	blockDB := NewBlockKvDB("test-chain", leveldbprovider.NewMemdbHandle(), logger)
	_, sb, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	blockDB.InitGenesis(sb)
	tx, err := blockDB.GetTxWithBlockInfo(block.Txs[1].Payload.TxId)
	assert.Nil(t, err)
	t.Logf("%+v", tx)
	assert.EqualValues(t, 1, tx.BlockHeight)
	assert.EqualValues(t, 1, tx.TxIndex)

	tx, err = blockDB.GetTxWithBlockInfo("i am test")
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}
func TestBlockKvDB_parseTxIdBlockInfo(t *testing.T) {
	value := constructTxIDBlockInfo(1, []byte("hash1"), 2)
	a, b, c, e := parseTxIdBlockInfo(value)
	assert.Nil(t, e)
	t.Log(a, b, c)
	a, b, c, e = parseTxIdBlockInfo([]byte("bad data"))
	assert.NotNil(t, e)
	t.Log(a, b, c, e)
}

func TestBlockKvDB_GetArchivedPivot(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	archivedPivot, err := db.GetArchivedPivot()
	assert.Equal(t, uint64(0), archivedPivot)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)

	err = db.dbHandle.Put([]byte(archivedPivotKey), constructBlockNumKey(10))
	assert.Nil(t, err)
	archivedPivot, err = db.GetArchivedPivot()
	assert.Equal(t, uint64(10), archivedPivot)
	assert.Nil(t, err)
}

func TestBlockKvDB_ShrinkBlocks(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	exist, err := db.BlockExists(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.True(t, exist)

	txIdsMap, err := db.ShrinkBlocks(1, 5)
	assert.Nil(t, err)
	assert.Equal(t, len(txIdsMap[1]), 10)

	txIdsMap, err = db.ShrinkBlocks(1, 10)
	assert.Nil(t, txIdsMap)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)

	_, bl, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block1})
	var arr []*serialization.BlockWithSerializedInfo
	arr = append(arr, bl)
	err = db.RestoreBlocks(arr)
	assert.Nil(t, err)
}

func TestBlockKvDB_GetBlockByHash(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlockByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, block1.String(), block.String())

	block, err = db.GetBlockByHash([]byte("i am test"))
	assert.Nil(t, block)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockKvDB_GetHeightByHash(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	height, err := db.GetHeightByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, height, uint64(1))

	height, err = db.GetHeightByHash([]byte("i am test"))
	assert.Equal(t, height, uint64(0))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockKvDB_GetBlockHeaderByHeight(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	header, err := db.GetBlockHeaderByHeight(3)
	assert.Nil(t, err)
	assert.Equal(t, header.String(), block3.Header.String())

	header, err = db.GetBlockHeaderByHeight(10)
	assert.Nil(t, header)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockKvDB_GetBlock(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlock(3)
	assert.Nil(t, err)
	assert.Equal(t, block.String(), block3.String())
}

func TestBlockKvDB_GetLastBlock(t *testing.T) {
	db := initDb()
	//defer db.Close()
	init5Blocks(db)

	block, err := db.GetLastBlock()
	assert.Nil(t, err)
	assert.Equal(t, block.String(), block5.String())

	db.Close()
	block, err = db.GetLastBlock()
	assert.Nil(t, block)
	assert.Equal(t, strings.Contains(err.Error(), "closed"), true)
}

func TestBlockKvDB_GetLastConfigBlock(t *testing.T) {
	db := initDb()
	defer db.Close()
	//init5Blocks(db)

	commitBlock(db, block0)
	commitBlock(db, block1)
	commitBlock(db, block2)
	commitBlock(db, block3)

	block, err := db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, block.String(), block0.String())

	commitBlock(db, configBlock4)
	commitBlock(db, block5)

	block, err = db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, block.String(), configBlock4.String())
}

func TestBlockKvDB_GetBlockByTx(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlockByTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, block5.String(), block.String())

	block, err = db.GetBlockByTx("i am test")
	assert.Nil(t, block)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockKvDB_GetTxHeight(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	txHeight, err := db.GetTxHeight(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, txHeight, uint64(5))

	txHeight, err = db.GetTxHeight("i am test")
	assert.Equal(t, txHeight, uint64(0))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockKvDB_GetTx(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	tx, err := db.GetTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, tx.String(), block5.Txs[0].String())

	tx, err = db.GetTx("i am test")
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}

func TestBlockKvDB_TxExists(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	exist, err := db.TxExists(block5.Txs[0].Payload.TxId)
	assert.True(t, exist)
	assert.Nil(t, err)

	exist, err = db.TxExists("i am test")
	assert.False(t, exist)
	assert.Nil(t, err)
}

func TestBlockKvDB_TxArchived(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	exist, err := db.TxExists(block1.Txs[0].Payload.TxId)
	assert.True(t, exist)
	assert.Nil(t, err)

	_, err = db.TxArchived(block1.Txs[0].Payload.TxId)
	// todo houfa
	//assert.False(t, archived)
	//assert.Nil(t, err)

	txIdsMap, err := db.ShrinkBlocks(1, 5)
	assert.Nil(t, err)
	assert.Equal(t, len(txIdsMap[1]), 10)

	_, err = db.TxArchived(block1.Txs[0].Payload.TxId)
	//assert.True(t, archived)
	//assert.Nil(t, err)
}

func TestBlockKvDB_GetTxConfirmedTime(t *testing.T) {
	db := initDb()
	defer db.Close()
	init5Blocks(db)

	time, err := db.GetTxConfirmedTime(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, time, block5.Header.BlockTimestamp)

	time, err = db.GetTxConfirmedTime("i am test")
	assert.Equal(t, time, int64(-1))
	assert.Equal(t, strings.Contains(err.Error(), "leveldb: not found"), true)
}
