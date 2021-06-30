/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package resultsqldb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"chainmaker.org/chainmaker-go/localconf"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	storePb "chainmaker.org/chainmaker/pb-go/store"
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker/protocol/test"
	"chainmaker.org/chainmaker-go/store/dbprovider/rawsqlprovider"
	"chainmaker.org/chainmaker-go/store/serialization"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}

func generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height int64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:32])
}

func createConfigBlock(chainId string, height int64) *storePb.BlockWithRWSet {
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
	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: []*commonPb.TxRWSet{},
	}
}

func createBlockAndRWSets(chainId string, height int64, txNum int) *storePb.BlockWithRWSet {
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

	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: txRWSets,
	}
}

var testChainId = "testchainid_1"
var block0 = createConfigBlock(testChainId, 0)
var block1 = createBlockAndRWSets(testChainId, 1, 10)
var block2 = createBlockAndRWSets(testChainId, 2, 2)

/*var block3, _ = createBlockAndRWSets(testChainId, 3, 2)
var configBlock4 = createConfigBlock(testChainId, 4)
var block5, _ = createBlockAndRWSets(testChainId, 5, 3)*/

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

//
//func TestMain(m *testing.M) {
//	fmt.Println("begin")
//	db, err := NewHistoryMysqlDB(testChainId, log)
//	if err != nil {
//		panic("faild to open mysql")
//	}
//	// clear data
//	historyMysqlDB := db.(*ResultSqlDB)
//	historyMysqlDB.db.Migrator().DropTable(&HistoryInfo{})
//	m.Run()
//	fmt.Println("end")
//}

func initProvider() protocol.SqlDBHandle {
	conf := &localconf.SqlDbConfig{}
	conf.Dsn = ":memory:"
	conf.SqlDbType = "sqlite"
	conf.SqlLogMode = "Info"
	p := rawsqlprovider.NewSqlDBHandle("chain1", conf, log)
	return p
}

//初始化DB并同时初始化创世区块
func initSqlDb() *ResultSqlDB {
	db, _ := newResultSqlDB(testChainId, initProvider(), log)
	_, blockInfo, _ := serialization.SerializeBlock(block0)
	db.InitGenesis(blockInfo)
	return db
}

func TestResultSqlDB_CommitBlock(t *testing.T) {
	db := initSqlDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
}

func TestHistorySqlDB_GetLastSavepoint(t *testing.T) {
	db := initSqlDb()
	_, block1, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(block1)
	assert.Nil(t, err)
	height, err := db.GetLastSavepoint()
	assert.Nil(t, err)
	assert.Equal(t, uint64(block1.Block.Header.BlockHeight), height)

	_, block2, err := serialization.SerializeBlock(block2)
	assert.Nil(t, err)
	err = db.CommitBlock(block2)
	assert.Nil(t, err)
	height, err = db.GetLastSavepoint()
	assert.Nil(t, err)
	assert.Equal(t, uint64(block2.Block.Header.BlockHeight), height)
}
