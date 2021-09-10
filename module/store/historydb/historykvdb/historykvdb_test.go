/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package historykvdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/store/cache"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	leveldbprovider "chainmaker.org/chainmaker/store-leveldb/v2"

	"chainmaker.org/chainmaker-go/store/historydb"
	"chainmaker.org/chainmaker-go/store/serialization"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/test"
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

func createConfigBlock(chainId string, height uint64) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer: &acPb.Member{
				OrgId:      "org1",
				MemberInfo: []byte("User1"),
			},
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId:      chainId,
					TxType:       commonPb.TxType_INVOKE_CONTRACT,
					ContractName: syscontract.SystemContract_CHAIN_CONFIG.String(),
				},
				Sender: &commonPb.EndorsementEntry{
					Signer: &acPb.Member{
						OrgId:      "org1",
						MemberInfo: []byte("Admin"),
					},
					Signature: []byte("signature1"),
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
	return &storePb.BlockWithRWSet{
		Block:    block,
		TxRWSets: []*commonPb.TxRWSet{},
	}
}

func createBlockAndRWSets(chainId string, height uint64, txNum int) *storePb.BlockWithRWSet {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer: &acPb.Member{
				OrgId:      "org1",
				MemberInfo: []byte("User1"),
				MemberType: 0,
			},
		},
	}

	for i := 0; i < txNum; i++ {

		tx := &commonPb.Transaction{
			Payload: &commonPb.Payload{
				ChainId:      chainId,
				TxId:         generateTxId(chainId, height, i),
				TxType:       commonPb.TxType_INVOKE_CONTRACT,
				ContractName: "contract1",
				Method:       "Function1",
				Parameters:   nil,
			},
			Sender: &commonPb.EndorsementEntry{
				Signer: &acPb.Member{
					OrgId:      "org1",
					MemberInfo: []byte("User" + strconv.Itoa(i)),
				},
				Signature: []byte("signature1"),
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

func createBlock(chainId string, height uint64) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
			Proposer: &acPb.Member{
				OrgId:      "org1",
				MemberInfo: []byte("User1"),
				MemberType: 0,
			},
		},
		Txs: []*commonPb.Transaction{
			{
				Payload: &commonPb.Payload{
					ChainId: chainId,
				},
				Sender: &commonPb.EndorsementEntry{
					Signer: &acPb.Member{
						OrgId:      "org1",
						MemberInfo: []byte("User1"),
					},
					Signature: []byte("signature1"),
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

func commitBlock(db *HistoryKvDB, block *commonPb.Block) error {
	_, bl, _ := serialization.SerializeBlock(&storePb.BlockWithRWSet{Block: block})
	return db.CommitBlock(bl)
}

func initProvider() protocol.DBHandle {
	//conf := &localconf.StorageConfig{}
	//path := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	//conf.StorePath = path
	//
	//lvlConfig := &localconf.LevelDbConfig{
	//	StorePath: path,
	//}
	//p := leveldbprovider.NewLevelDBHandle(testChainId, "test", lvlConfig, log)
	//return p
	return leveldbprovider.NewMemdbHandle()
}

//初始化DB并同时初始化创世区块
func initKvDb() *HistoryKvDB {
	db := NewHistoryKvDB(initProvider(), cache.NewStoreCacheMgr(testChainId, 10, log), log)
	_, blockInfo, _ := serialization.SerializeBlock(block0)
	db.InitGenesis(blockInfo)
	return db
}

func TestHistoryKVDB_CommitBlock(t *testing.T) {
	defer func() {
		err := recover()
		assert.True(t, strings.Contains(err.(string), "closed"))
	}()
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)

	db.Close()
	// should panic
	err = commitBlock(db, block2.Block)
}

func TestHistoryKvDB_GetLastSavepoint(t *testing.T) {
	db := initKvDb()
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

	db.Close()
	height, err = db.GetLastSavepoint()
	assert.Equal(t, height, uint64(0))
	t.Log(err)
	assert.Equal(t, strings.Contains(err.Error(), "closed"), true)
}
func TestHistoryKvDB_GetHistoryForKey(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
	result, err := db.GetHistoryForKey("contract1", []byte("key_1"))
	assert.Nil(t, err)

	assert.Equal(t, 1, getCount(result))

	result, err = db.GetHistoryForKey("contract1", []byte("key_1"))
	assert.Nil(t, err)
	result.Next()
	value, err := result.Value()
	assert.Equal(t, value.BlockHeight, block1.Block.Header.BlockHeight)

}
func getCount(i historydb.HistoryIterator) int {
	count := 0
	for i.Next() {
		count++
	}
	return count
}
func TestHistoryKvDB_GetAccountTxHistory(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	assert.Nil(t, err)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
	result, err := db.GetAccountTxHistory([]byte("User1"))
	assert.Nil(t, err)
	assert.Equal(t, 1, getCount(result))
	for result.Next() {
		v, _ := result.Value()
		t.Logf("%#v", v)
	}

	result, err = db.GetAccountTxHistory([]byte("User1"))
	assert.Nil(t, err)
	result.Next()
	value, err := result.Value()
	assert.Equal(t, value.BlockHeight, block1.Block.Header.BlockHeight)
}
func TestHistoryKvDB_GetContractTxHistory(t *testing.T) {
	db := initKvDb()
	block1.TxRWSets[0].TxWrites[0].Value = nil
	_, blockInfo, err := serialization.SerializeBlock(block1)
	err = db.CommitBlock(blockInfo)
	assert.Nil(t, err)
	result, err := db.GetContractTxHistory("contract1")
	assert.Nil(t, err)
	assert.Equal(t, 10, getCount(result))
	for result.Next() {
		v, _ := result.Value()
		t.Logf("%#v", v)
	}
	value, err := result.Value()
	assert.Error(t, err, "empty dbKey")
	assert.Nil(t, value)
	result.Release()

	result, err = db.GetContractTxHistory("contract1")
	assert.Nil(t, err)
	result.Next()
	value, err = result.Value()
	assert.Equal(t, value.BlockHeight, block1.Block.Header.BlockHeight)
}
