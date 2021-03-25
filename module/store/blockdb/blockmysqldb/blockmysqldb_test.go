/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockmysqldb

import (
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/store/serialization"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gotest.tools/assert"
	"testing"
)

var log = &logger.GoLogger{}

func generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height int64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createConfigBlock(chainId string, height int64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{
			{
				Header: &commonPb.TxHeader{
					ChainId: chainId,
					TxType:  commonPb.TxType_UPDATE_CHAIN_CONFIG,
					Sender: &acPb.SerializedMember{
						OrgId: "org1",
					},
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
	block.Txs[0].Header.TxId = generateTxId(chainId, height, 0)
	return block
}

func createBlockAndRWSets(chainId string, height int64, txNum int) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
	}

	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Header: &commonPb.TxHeader{
				ChainId: chainId,
				TxId:    generateTxId(chainId, height, i),
				Sender: &acPb.SerializedMember{
					OrgId: "org1",
				},
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
			TxId: block.Txs[i].Header.TxId,
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

func createBlock(chainId string, height int64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{
			{
				Header: &commonPb.TxHeader{
					ChainId: chainId,
					Sender: &acPb.SerializedMember{
						OrgId: "org1",
					},
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
	block.Txs[0].Header.TxId = generateTxId(chainId, height, 0)
	return block
}

func TestMain(m *testing.M) {
	fmt.Println("begin")
	db, err := NewBlockMysqlDB(testChainId, log)
	if err != nil {
		panic("faild to open mysql")
	}
	// clear data
	blockMysqlDB := db.(*BlockMysqlDB)
	blockMysqlDB.db.Migrator().DropTable(&BlockInfo{})
	blockMysqlDB.db.Migrator().DropTable(&TxInfo{})
	m.Run()
	fmt.Println("end")
}

func TestBlockMysqlDB_CommitBlock(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)
	_, blockInfo0, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block0})
	assert.NilError(t, err)
	err = db.CommitBlock(blockInfo0)
	assert.NilError(t, err)
	_, blockInfo1, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block1})
	assert.NilError(t, err)
	err = db.CommitBlock(blockInfo1)
	assert.NilError(t, err)
}

func TestBlockMysqlDB_HasBlock(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)
	exist, err := db.BlockExists(block1.Header.BlockHash)
	assert.NilError(t, err)
	assert.Equal(t, true, exist)
}

func TestBlockMysqlDB_GetBlock(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)
	block, err := db.GetBlockByHash(block1.Header.BlockHash)
	assert.NilError(t, err)
	assert.Equal(t, block1.String(), block.String())
}

func TestBlockMysqlDB_GetBlockAt(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)
	block, err := db.GetBlock(block1.Header.BlockHeight)
	assert.NilError(t, err)
	assert.Equal(t, block1.String(), block.String())
}

func TestBlockMysqlDB_GetLastBlock(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)
	block, err := db.GetLastBlock()
	assert.NilError(t, err)
	assert.Equal(t, block1.String(), block.String())
	_, blockInfo2, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block2})
	err = db.CommitBlock(blockInfo2)
	assert.NilError(t, err)
	block, err = db.GetLastBlock()
	assert.NilError(t, err)
	assert.Equal(t, block2.String(), block.String())

	_, blockInfo3, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block3})
	err = db.CommitBlock(blockInfo3)
	assert.NilError(t, err)
	block, err = db.GetLastBlock()
	assert.NilError(t, err)
	assert.Equal(t, block3.String(), block.String())
}

func TestBlockMysqlDB_GetLastConfigBlock(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)

	block, err := db.GetLastConfigBlock()
	assert.NilError(t, err)
	assert.Equal(t, int64(0), block.Header.BlockHeight)
	_, blockInfo4, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: configBlock4})
	err = db.CommitBlock(blockInfo4)
	assert.NilError(t, err)
	block, err = db.GetLastConfigBlock()
	assert.NilError(t, err)
	assert.Equal(t, configBlock4.String(), block.String())

	block5.Header.PreConfHeight = 4
	_, blockInfo5, err := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block5})
	err = db.CommitBlock(blockInfo5)
	assert.NilError(t, err)
	block, err = db.GetLastConfigBlock()
	assert.NilError(t, err)
	assert.Equal(t, configBlock4.String(), block.String())
}

func TestBlockMysqlDB_GetFilteredBlock(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)

	block, err := db.GetFilteredBlock(block1.Header.BlockHeight)
	assert.NilError(t, err)
	for id, txid := range block.TxIds {
		assert.Equal(t, block1.Txs[id].Header.TxId, txid)
	}
}

func TestBlockMysqlDB_GetLastSavepoint(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)

	height, err := db.GetLastSavepoint()
	assert.NilError(t, err)
	assert.Equal(t, uint64(block5.Header.BlockHeight), height)
}

func TestBlockMysqlDB_GetBlockByTx(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)

	block, err := db.GetBlockByTx(block5.Txs[0].Header.TxId)
	assert.NilError(t, err)
	assert.Equal(t, block5.String(), block.String())
}

func TestBlockMysqlDB_GetTx(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)

	tx, err := db.GetTx(block5.Txs[0].Header.TxId)
	assert.NilError(t, err)
	assert.Equal(t, block5.Txs[0].Header.TxId, tx.Header.TxId)
}

func TestBlockMysqlDB_HasTx(t *testing.T) {
	db, err := NewBlockMysqlDB(testChainId, log)
	assert.NilError(t, err)

	exist, err := db.TxExists(block5.Txs[0].Header.TxId)
	assert.NilError(t, err)
	assert.Equal(t, true, exist)
}
