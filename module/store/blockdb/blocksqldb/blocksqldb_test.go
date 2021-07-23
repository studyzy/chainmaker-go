/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blocksqldb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	storePb "chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/protocol/test"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}

func generateBlockHash(chainId string, height uint64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height uint64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createConfigBlock(chainId string, height uint64) *commonPb.Block {
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
	return block
}

func createBlockAndRWSets(chainId string, height uint64, txNum int) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer:    &acPb.Member{MemberInfo: []byte("User1")},
		},
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
var block0 = createConfigBlock(testChainId, 0)
var block1, _ = createBlockAndRWSets(testChainId, 1, 10)
var block2, _ = createBlockAndRWSets(testChainId, 2, 2)
var block3, _ = createBlockAndRWSets(testChainId, 3, 2)
var configBlock4 = createConfigBlock(testChainId, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3)

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
	conf := &localconf.SqlDbConfig{}
	conf.Dsn = ":memory:"
	conf.SqlDbType = "sqlite"
	conf.SqlLogMode = "Info"
	p := rawsqlprovider.NewSqlDBHandle("chain1", conf, log)
	p.CreateTableIfNotExist(&BlockInfo{})
	p.CreateTableIfNotExist(&TxInfo{})
	return p
}
func initSqlDb() *BlockSqlDB {
	db, _ := newBlockSqlDB(testChainId, initProvider(), log)
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
	db := initSqlDb()
	err := commitBlock(db, block0)
	assert.Nil(t, err)
	err = commitBlock(db, block1)
	assert.Nil(t, err)
}

func TestBlockMysqlDB_HasBlock(t *testing.T) {
	db := initSqlDb()
	exist, err := db.BlockExists(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, false, exist)
	init5Blocks(db)
	exist, err = db.BlockExists(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, true, exist)
}

func TestBlockMysqlDB_GetBlock(t *testing.T) {
	db := initSqlDb()
	init5Blocks(db)
	block, err := db.GetBlockByHash(block1.Header.BlockHash)
	assert.Nil(t, err)
	assert.Equal(t, block1.Header.BlockHeight, block.Header.BlockHeight)
}

func TestBlockMysqlDB_GetBlockAt(t *testing.T) {
	db := initSqlDb()
	init5Blocks(db)
	block, err := db.GetBlock(block2.Header.BlockHeight)
	assert.Nil(t, err)
	assert.Equal(t, block2.Header.BlockHeight, block.Header.BlockHeight)
}

func TestBlockSqlDB_GetLastBlock(t *testing.T) {
	db := initSqlDb()
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

//func TestBlockMysqlDB_GetLastConfigBlock(t *testing.T) {
//	db:=initSqlDb()
//	init5Blocks(db)
//
//	block, err := db.GetLastConfigBlock()
//	assert.Nil(t, err)
//	assert.Equal(t, int64(0), block.Header.BlockHeight)
//	err = db.CommitBlock(configBlock4)
//	assert.Nil(t, err)
//	block, err = db.GetLastConfigBlock()
//	assert.Nil(t, err)
//	assert.Equal(t, configBlock4.String(), block.String())
//
//	block5.Header.PreConfHeight = 4
//	err = db.CommitBlock(block5)
//	assert.Nil(t, err)
//	block, err = db.GetLastConfigBlock()
//	assert.Nil(t, err)
//	assert.Equal(t, configBlock4.String(), block.String())
//}

func TestBlockMysqlDB_GetFilteredBlock(t *testing.T) {
	db := initSqlDb()
	init5Blocks(db)

	block, err := db.GetFilteredBlock(block1.Header.BlockHeight)
	assert.Nil(t, err)
	for id, txid := range block.TxIds {
		assert.Equal(t, block1.Txs[id].Payload.TxId, txid)
	}
}

func TestBlockMysqlDB_GetBlockByTx(t *testing.T) {
	db := initSqlDb()
	init5Blocks(db)

	block, err := db.GetBlockByTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, block5.Header.BlockHeight, block.Header.BlockHeight)
}

func TestBlockMysqlDB_GetTx(t *testing.T) {
	db := initSqlDb()
	init5Blocks(db)

	tx, err := db.GetTx(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, block5.Txs[0].Payload.TxId, tx.Payload.TxId)
}

func TestBlockMysqlDB_HasTx(t *testing.T) {
	db := initSqlDb()
	init5Blocks(db)

	exist, err := db.TxExists(block5.Txs[0].Payload.TxId)
	assert.Nil(t, err)
	assert.Equal(t, true, exist)
}
