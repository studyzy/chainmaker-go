/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package statemysqldb

import (
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gotest.tools/assert"

	//"gotest.tools/assert"
	"testing"
)

var log = &logger.GoLogger{}

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

func TestMain(m *testing.M) {
	fmt.Println("begin")
	db, err := NewStateMysqlDB(testChainId, log)
	if err != nil {
		panic("faild to open mysql")
	}
	// clear data
	stateMysqlDB := db.(*StateMysqlDB)
	stateMysqlDB.db.Migrator().DropTable(&StateInfo{})
	m.Run()
	fmt.Println("end")
}

func TestStateMysqlDB_CommitBlock(t *testing.T) {
	db, err := NewStateMysqlDB(testChainId, log)
	assert.NilError(t, err)
	block1.TxRWSets[0].TxWrites[0].Value = nil
	err = db.CommitBlock(block1)
	assert.NilError(t, err)
}

func TestStateMysqlDB_ReadObject(t *testing.T) {
	db, err := NewStateMysqlDB(testChainId, log)
	assert.NilError(t, err)
	value, err := db.ReadObject(block1.TxRWSets[0].TxWrites[0].ContractName, block1.TxRWSets[0].TxWrites[0].Key)
	assert.NilError(t, err)
	//assert.Equal(t, block1.TxRWSets[0].TxWrites[0].Value, value)
	fmt.Println(string(value))
}

func TestStateMysqlDB_GetLastSavepoint(t *testing.T) {
	db, err := NewStateMysqlDB(testChainId, log)
	assert.NilError(t, err)
	height, err := db.GetLastSavepoint()
	assert.NilError(t, err)
	assert.Equal(t, uint64(block1.Block.Header.BlockHeight), height)

	err = db.CommitBlock(block2)
	assert.NilError(t, err)
	height, err = db.GetLastSavepoint()
	assert.NilError(t, err)
	assert.Equal(t, uint64(block2.Block.Header.BlockHeight), height)

}
