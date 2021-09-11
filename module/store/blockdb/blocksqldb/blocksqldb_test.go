/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/store/serialization"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2/test"
	rawsqlprovider "chainmaker.org/chainmaker/store-sqldb/v2"
	"github.com/stretchr/testify/assert"
)

var (
	log = &test.GoLogger{}
	//conf = &rawsqlprovider.SqlDbConfig{
	//	Dsn:        ":memory:",
	//	SqlDbType:  "sqlite",
	//	SqlLogMode: "Info",
	//}
)
var db = rawsqlprovider.NewMemSqlDBHandle(log)

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

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
					ChainId: chainId,
					TxType:  commonPb.TxType_INVOKE_CONTRACT,
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
			ChainId:     chainId,
			BlockHeight: height,
			Proposer:    &acPb.Member{MemberInfo: []byte("User1")},
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

func init5Blocks(db *BlockSqlDB) {
	commitBlock(db, block0)
	commitBlock(db, block1)
	commitBlock(db, block2)
	commitBlock(db, block3)
	commitBlock(db, configBlock4)
	commitBlock(db, block5)
}
func commitBlock(db *BlockSqlDB, block *commonPb.Block) error {
	_, bl, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	return db.CommitBlock(bl)
}
func createBlock(chainId string, height uint64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer:    &acPb.Member{MemberInfo: []byte("User1")},
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId: chainId,
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
	return block
}

func initProvider() *rawsqlprovider.SqlDBHandle {
	p := rawsqlprovider.NewMemSqlDBHandle(log)
	p.CreateTableIfNotExist(&BlockInfo{})
	p.CreateTableIfNotExist(&TxInfo{})
	return p
}
func initSqlDb() *BlockSqlDB {
	db := NewBlockSqlDB(testChainId, initProvider(), log)
	return db
}

//func TestMain(m *testing.M) {
//	fmt.Println("begin")
//	db, err := NewBlockSqlDB(testChainId,initProvider(), log)
//	if err != nil {
//		panic("faild to open mysql")
//	}
//	// clear data
//	//blockMysqlDB := db.(*BlockSqlDB)
//	//blockMysqlDB.db.Migrator().DropTable(&BlockInfo{})
//	//blockMysqlDB.db.Migrator().DropTable(&TxInfo{})
//	m.Run()
//	fmt.Println("end")
//}

func TestBlockMysqlDB_CommitBlock(t *testing.T) {
	defer func() {
		err := recover()
		fmt.Println(err)
	}()
	db := initSqlDb()
	//defer db.Close()
	err := commitBlock(db, block0)
	assert.Nil(t, err)
	err = commitBlock(db, block1)
	assert.Nil(t, err)

	db.Close()
	err = commitBlock(db, block3)
	assert.Equal(t, strings.Contains(err.Error(), "database transaction error"), true)
}

func TestBlockMysqlDB_HasBlock(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	exist, err := db.BlockExists(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, false, exist)
	init5Blocks(db)
	exist, err = db.BlockExists(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, true, exist)

	db.Close()
	exist, err = db.BlockExists(block1.Header.BlockHash)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
	assert.False(t, exist)
}

func TestBlockMysqlDB_GetBlock(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)
	block, err := db.GetBlockByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, block1.Header.BlockHeight, block.Header.BlockHeight)

	db.Close()
	block, err = db.GetBlockByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Nil(t, block)
}

func TestBlockMysqlDB_GetBlockAt(t *testing.T) {
	db := initSqlDb()
	defer db.Close()
	init5Blocks(db)
	block, err := db.GetBlock(block2.Header.BlockHeight)
	assert.Nil(t, err)
	assert.Equal(t, block2.Header.BlockHeight, block.Header.BlockHeight)
}

func TestBlockSqlDB_GetLastBlock(t *testing.T) {
	db := initSqlDb()
	defer db.Close()
	_, genesis, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block0})
	err := db.InitGenesis(genesis)
	assert.Nil(t, err)
	block, err := db.GetLastBlock()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), block.Header.BlockHeight)
	err = commitBlock(db, block1)
	assert.Nil(t, err)
	err = commitBlock(db, block2)
	assert.Nil(t, err)
	block, err = db.GetLastBlock()
	assert.Nil(t, err)
	assert.Equal(t, block2.Header.BlockHeight, block.Header.BlockHeight)

	err = commitBlock(db, block3)
	assert.Nil(t, err)
	block, err = db.GetLastBlock()
	assert.Nil(t, err)
	assert.Equal(t, block3.Header.BlockHeight, block.Header.BlockHeight)
}

func TestBlockMysqlDB_GetLastConfigBlock(t *testing.T) {
	db := initSqlDb()
	defer db.Close()
	commitBlock(db, block0)
	commitBlock(db, block1)
	commitBlock(db, block2)
	commitBlock(db, block3)

	block, err := db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), block.Header.BlockHeight)
	err = commitBlock(db, configBlock4)
	assert.Nil(t, err)
	block, err = db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, configBlock4.String(), block.String())

	err = commitBlock(db, block5)
	assert.Nil(t, err)
	block, err = db.GetLastConfigBlock()
	assert.Nil(t, err)
	assert.Equal(t, configBlock4.String(), block.String())
}

func TestBlockMysqlDB_GetFilteredBlock(t *testing.T) {
	db := initSqlDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetFilteredBlock(block1.Header.BlockHeight)
	assert.Nil(t, err)
	for id, txid := range block.TxIds {
		assert.Equal(t, block1.Txs[id].Payload.TxId, txid)
	}
}

func TestBlockMysqlDB_GetBlockByTx(t *testing.T) {
	db := initSqlDb()
	defer db.Close()
	init5Blocks(db)

	block, err := db.GetBlockByTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, block5.Header.BlockHeight, block.Header.BlockHeight)
}

func TestBlockMysqlDB_GetTx(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	tx, err := db.GetTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, block5.Txs[0].Payload.TxId, tx.Payload.TxId)

	tx, err = db.GetTx("i am test")
	assert.Nil(t, tx)
	assert.Nil(t, err)

	tx, err = db.GetTx("")
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "parameter is null"), true)

	db.Close()
	tx, err = db.GetTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
}

func TestBlockMysqlDB_HasTx(t *testing.T) {
	db := initSqlDb()
	defer db.Close()
	init5Blocks(db)

	exist, err := db.TxExists(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, true, exist)
}

func TestBlockSqlDB_GetHeightByHash(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	blockHeight, err := db.GetHeightByHash(block5.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), blockHeight)

	blockHeight, err = db.GetHeightByHash([]byte("I am testing"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(math.MaxUint64), blockHeight)

	db.Close()
	blockHeight, err = db.GetHeightByHash(block5.Header.BlockHash)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
	assert.Equal(t, uint64(math.MaxUint64), blockHeight)
}

func TestBlockSqlDB_GetBlockHeaderByHeight(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	blockHeader, err := db.GetBlockHeaderByHeight(2)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), blockHeader.BlockHeight)

	blockHeader, err = db.GetBlockHeaderByHeight(7)
	assert.Nil(t, err)
	assert.Nil(t, blockHeader)

	db.Close()

	blockHeader, err = db.GetBlockHeaderByHeight(2)
	assert.Nil(t, err)
	assert.Nil(t, blockHeader)
}

func TestBlockSqlDB_GetTxHeight(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	txHeight, err := db.GetTxHeight(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), txHeight)

	txHeight, err = db.GetTxHeight("I am testing")
	assert.Nil(t, err)
	assert.Equal(t, uint64(math.MaxUint64), txHeight)

	db.Close()
	txHeight, err = db.GetTxHeight(block5.Txs[0].Payload.TxId)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
	assert.Equal(t, uint64(math.MaxUint64), txHeight)
}

func TestBlockSqlDB_TxArchived(t *testing.T) {
	db := &BlockSqlDB{}
	isArchived, err := db.TxArchived("I am testing")
	assert.False(t, isArchived)
	assert.Nil(t, err)
}

func TestBlockSqlDB_GetArchivedPivot(t *testing.T) {
	db := &BlockSqlDB{}
	pivot, err := db.GetArchivedPivot()
	assert.Equal(t, pivot, uint64(0))
	assert.Nil(t, err)
}

func TestBlockSqlDB_ShrinkBlocks(t *testing.T) {
	db := &BlockSqlDB{}
	res, err := db.ShrinkBlocks(0, 1)
	assert.Nil(t, res)
	assert.Equal(t, err.Error(), errNotImplement.Error())
}

func TestBlockSqlDB_RestoreBlocks(t *testing.T) {
	db := &BlockSqlDB{}
	err := db.RestoreBlocks([]*serialization.BlockWithSerializedInfo{})
	assert.Equal(t, err.Error(), errNotImplement.Error())
}

func Test_newBlockSqlDB(t *testing.T) {
	db := NewBlockSqlDB("chain1", db, log)
	db.Close()
}

func TestBlockSqlDB_GetLastSavepoint(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	height, err := db.GetLastSavepoint()
	assert.Equal(t, uint64(5), height)
	assert.Nil(t, err)

	db.Close()
	height, err = db.GetLastSavepoint()
	assert.Equal(t, uint64(0), height)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
}

func TestBlockSqlDB_GetTxWithBlockInfo(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	tx, err := db.GetTxWithBlockInfo(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), tx.BlockHeight)
	assert.Equal(t, uint32(0), tx.TxIndex)

	tx, err = db.GetTxWithBlockInfo("")
	assert.Nil(t, err)
	assert.Nil(t, tx)

	db.Close()
	tx, err = db.GetTxWithBlockInfo(block5.Txs[0].Payload.TxId)
	assert.Nil(t, tx)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
}

func TestBlockSqlDB_initDb(t *testing.T) {
	defer func() {
		err := recover()
		assert.Equal(t, strings.Contains(err.(string), "init state sql db table `block_infos` failtable operation error"), true)
	}()
	db := initSqlDb()

	db.Close()

	db.initDb("chain1")
}

func TestBlockSqlDB_GetTxConfirmedTime(t *testing.T) {
	db := &BlockSqlDB{}
	num, err := db.GetTxConfirmedTime("")
	assert.Equal(t, err.Error(), errNotImplement.Error())
	assert.Equal(t, num, int64(num))
}

func TestBlockSqlDB_getTxsByBlockHeight(t *testing.T) {
	db := initSqlDb()
	//defer db.Close()
	init5Blocks(db)

	txs, err := db.getTxsByBlockHeight(5)
	assert.Equal(t, len(txs), 3)
	assert.Nil(t, err)

	db.Close()
	txs, err = db.getTxsByBlockHeight(5)
	assert.Nil(t, txs)
	assert.Equal(t, strings.Contains(err.Error(), "sql query error"), true)
}
