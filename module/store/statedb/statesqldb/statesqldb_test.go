/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statesqldb

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
	"chainmaker.org/chainmaker/protocol"
	"chainmaker.org/chainmaker/protocol/test"
	"github.com/stretchr/testify/assert"
)

var log = &test.GoLogger{}

//生成测试用的blockHash
func generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

//生成测试用的txid
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
	var txRWSets []*commonPb.TxRWSet
	for i := 0; i < 1; i++ {
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

//var block2 = createBlockAndRWSets(testChainId, 2, 2)

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

var conf = &localconf.SqlDbConfig{
	Dsn:        ":memory:",
	SqlDbType:  "sqlite",
	SqlLogMode: "Info",
}

func initProvider() protocol.SqlDBHandle {
	p := rawsqlprovider.NewSqlDBHandle("chain1", conf, log)
	return p
}
func initStateSqlDB() *StateSqlDB {
	db, _ := newStateSqlDB("dbName", "chain1", initProvider(), conf, log)
	_, blockInfo0, _ := serialization.SerializeBlock(block0)
	db.InitGenesis(blockInfo0)
	return db
}

//func TestMain(m *testing.M) {
//	fmt.Println("begin")
//	db, err := NewStateMysqlDB(testChainId, log)
//	if err != nil {
//		panic("faild to open mysql")
//	}
//	// clear data
//	stateMysqlDB := db.(*StateSqlDB)
//	stateMysqlDB.db.Migrator().DropTable(&StateInfo{})
//	m.Run()
//	fmt.Println("end")
//}

func TestStateSqlDB_CommitBlock(t *testing.T) {
	db := initStateSqlDB()
	block1.TxRWSets[0].TxWrites[0].ContractName = ""
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo1, _ := serialization.SerializeBlock(block1)
	err := db.CommitBlock(blockInfo1)
	assert.Nil(t, err)
}

func TestStateSqlDB_ReadObject(t *testing.T) {
	db := initStateSqlDB()
	block1.TxRWSets[0].TxWrites[0].ContractName = ""
	_, blockInfo1, _ := serialization.SerializeBlock(block1)
	db.CommitBlock(blockInfo1)
	value, err := db.ReadObject(block1.TxRWSets[0].TxWrites[0].ContractName, block1.TxRWSets[0].TxWrites[0].Key)
	assert.Nil(t, err)
	assert.Equal(t, block1.TxRWSets[0].TxWrites[0].Value, value)
	//t.Logf("%s", string(value))
	value, err = db.ReadObject(block1.TxRWSets[0].TxWrites[0].ContractName, []byte("another key"))
	assert.Nil(t, err)
	assert.Nil(t, value)
}

//func TestStateSqlDB_GetLastSavepoint(t *testing.T) {
//	db:=initStateSqlDB()
//	height, err := db.GetLastSavepoint()
//	assert.Nil(t, err)
//	assert.Equal(t, uint64(block1.Block.Header.BlockHeight), height)
//
//	err = db.CommitBlock(block2)
//	assert.Nil(t, err)
//	height, err = db.GetLastSavepoint()
//	assert.Nil(t, err)
//	assert.Equal(t, uint64(block2.Block.Header.BlockHeight), height)
//
//}
func TestGetContractDbName(t *testing.T) {
	config := &localconf.SqlDbConfig{DbPrefix: "org1_"}
	contractName := "0x61e778670e7c6e2b65f0f0491963afd10d9bfd90308388361ce7ea5916437571"
	t.Log(contractName)
	dbName := getContractDbName(config, "chain1", contractName)
	t.Log(dbName, len(dbName))
}
