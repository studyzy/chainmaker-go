/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/store/serialization"
	"chainmaker.org/chainmaker-go/store/types"
	"os"
	"time"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var ledgerPath = os.TempDir()
var chainId = "testchain1"

//var dbType = types.MySQL
var dbType = types.LevelDb

var defaultContractName = "contract1"
var defaultChainId = "testchainid"
var block5 = createBlock(chainId, 5)
var txRWSets = []*commonPb.TxRWSet{
	{
		//TxId: "abcdefg",
		TxWrites: []*commonPb.TxWrite{
			{
				Key:          []byte("key1"),
				Value:        []byte("value1"),
				ContractName: defaultContractName,
			},
			{
				Key:          []byte("key2"),
				Value:        []byte("value2"),
				ContractName: defaultContractName,
			},
			{
				Key:          []byte("key3"),
				Value:        nil,
				ContractName: defaultContractName,
			},
		},
	},
}
var config = getConfig()

func getConfig() *localconf.StorageConfig {
	conf := &localconf.StorageConfig{}
	leveldbFolder := os.TempDir() + fmt.Sprintf("/lvldb%d", time.Now().Unix())
	conf.StorePath = leveldbFolder
	var sqlConfig = &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn:       ":memory:",
	}
	lvlConfig := &localconf.LevelDbConfig{
		StorePath: leveldbFolder,
	}
	dbConfig := localconf.DbConfig{
		DbType:        "sql",
		LevelDbConfig: lvlConfig,
		SqlDbConfig:   sqlConfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig

	return conf
}

func generateBlockHash(chainId string, height int64) []byte {
	blockHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", chainId, height)))
	return blockHash[:]
}

func generateTxId(chainId string, height int64, index int) string {
	txIdBytes := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", chainId, height, index)))
	return hex.EncodeToString(txIdBytes[:])
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
					TxId:    generateTxId(chainId, height, 0),
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
					ContractName: defaultContractName,
				},
			},
		}
		txRWSets = append(txRWSets, txRWset)
	}

	return block, txRWSets
}

var log = &logger.GoLogger{}

//func TestMain(m *testing.M) {
//	fmt.Println("begin")
//	if dbType == types.MySQL {
//		// drop mysql table
//		conf := &localconf.ChainMakerConfig.StorageConfig
//		conf.Provider = "MySQL"
//		conf.MysqlConfig.Dsn = "root:123456@tcp(127.0.0.1:3306)/"
//		db, err := blocksqldb.NewBlockSqlDB(chainId, log)
//		if err != nil {
//			panic("faild to open mysql")
//		}
//		// clear data
//		gormDB := db.(*blocksqldb.BlockSqlDB).GetDB()
//		gormDB.Migrator().DropTable(&blocksqldb.BlockInfo{})
//		gormDB.Migrator().DropTable(&blocksqldb.TxInfo{})
//		gormDB.Migrator().DropTable(&statesqldb.StateInfo{})
//		gormDB.Migrator().DropTable(&historysqldb.HistoryInfo{})
//	}
//	os.RemoveAll(chainId)
//	m.Run()
//	fmt.Println("end")
//}

func Test_blockchainStoreImpl_GetBlock(t *testing.T) {
	var funcName = "get block"
	tests := []struct {
		name  string
		block *commonPb.Block
	}{
		{funcName, createBlock(defaultChainId, 0)},
		{funcName, createBlock(defaultChainId, 1)},
		{funcName, createBlock(defaultChainId, 2)},
		{funcName, createBlock(defaultChainId, 3)},
		{funcName, createBlock(defaultChainId, 4)},
	}
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.PutBlock(tt.block, nil); err != nil {
				t.Errorf("blockchainStoreImpl.PutBlock(), error %v", err)
			}
			got, err := s.GetBlockByHash(tt.block.Header.BlockHash)
			assert.Nil(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.block.String(), got.String())
		})
	}
}

func Test_blockchainStoreImpl_PutBlock(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	//s, err := NewBlockStoredbTypeImpl(ledgerPath)

	if err != nil {
		panic(err)
	}
	defer s.Close()
	txRWSets[0].TxId = block5.Txs[0].Header.TxId
	err = s.PutBlock(block5, txRWSets)
	assert.Equal(t, nil, err)
}

func Test_blockchainStoreImpl_HasBlock(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	exist, err := s.BlockExists(block5.Header.BlockHash)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, exist)

	exist, err = s.BlockExists([]byte("not exist"))
	assert.Equal(t, nil, err)
	assert.Equal(t, false, exist)
}

func Test_blockchainStoreImpl_GetBlockAt(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	got, err := s.GetBlock(block5.Header.BlockHeight)
	assert.Equal(t, nil, err)
	assert.Equal(t, block5.String(), got.String())
}

func Test_blockchainStoreImpl_GetLastBlock(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	assert.Equal(t, nil, err)
	lastBlock, err := s.GetLastBlock()
	assert.Equal(t, nil, err)
	assert.Equal(t, block5.Header.BlockHeight, lastBlock.Header.BlockHeight)
}

func Test_blockchainStoreImpl_GetLastConfigBlock(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	configBlock := createConfigBlock(chainId, 6)
	err = s.PutBlock(configBlock, txRWSets)
	assert.Equal(t, nil, err)
	lastBlock, err := s.GetLastConfigBlock()
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(6), lastBlock.Header.BlockHeight)
}

func Test_blockchainStoreImpl_GetBlockByTx(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	block, err := s.GetBlockByTx(generateTxId(defaultChainId, 3, 0))
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(3), block.Header.BlockHeight)
}

func Test_blockchainStoreImpl_GetTx(t *testing.T) {
	funcName := "has tx"
	tests := []struct {
		name  string
		block *commonPb.Block
	}{
		{funcName, createBlock(defaultChainId, 1)},
		{funcName, createBlock(defaultChainId, 2)},
		{funcName, createBlock(defaultChainId, 3)},
		{funcName, createBlock(defaultChainId, 4)},
		{funcName, createBlock(defaultChainId, 999999)},
	}

	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	//assert.DeepEqual(t, s.GetTx(tests[0].block.Txs[0].TxId, )

	tx, err := s.GetTx(tests[0].block.Txs[0].Header.TxId)
	assert.Equal(t, nil, err)
	if tx == nil {
		t.Error("Error, GetTx")
	}
	assert.Equal(t, tx.Header.TxId, generateTxId(defaultChainId, 1, 0))

	//chain not exist
	tx, err = s.GetTx(generateTxId("not exist chain", 1, 0))
	assert.Equal(t, nil, err)
	if tx != nil {
		t.Error("Error, GetTx, expect nil")
	}
}

func Test_blockchainStoreImpl_HasTx(t *testing.T) {
	funcName := "has tx"
	tests := []struct {
		name  string
		block *commonPb.Block
	}{
		{funcName, createBlock(defaultChainId, 1)},
		{funcName, createBlock(defaultChainId, 2)},
		{funcName, createBlock(defaultChainId, 3)},
		{funcName, createBlock(defaultChainId, 4)},
		{funcName, createBlock(defaultChainId, 999999)},
	}
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	exist, err := s.TxExists(tests[0].block.Txs[0].Header.TxId)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, exist)
	exist, err = s.TxExists(tests[1].block.Txs[0].Header.TxId)
	assert.Equal(t, nil, err)
	assert.Equal(t, true, exist)
	exist, err = s.TxExists(tests[2].block.Txs[0].Header.TxId)
	assert.Equal(t, true, exist)
	assert.Equal(t, nil, err)
	exist, err = s.TxExists(tests[3].block.Txs[0].Header.TxId)
	assert.Equal(t, true, exist)
	assert.Equal(t, nil, err)
	exist, err = s.TxExists(tests[4].block.Txs[0].Header.TxId)
	assert.Equal(t, nil, err)
	assert.Equal(t, false, exist)
}

func Test_blockchainStoreImpl_ReadObject(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	defer s.Close()
	assert.Equal(t, nil, err)
	value, err := s.ReadObject(defaultContractName, []byte("key1"))
	assert.Equal(t, nil, err)
	assert.Equal(t, value, []byte("value1"))

	value, err = s.ReadObject(defaultContractName, []byte("key2"))
	assert.Equal(t, nil, err)
	assert.Equal(t, value, []byte("value2"))
}

func Test_blockchainStoreImpl_SelectObject(t *testing.T) {
	if dbType == types.MySQL {
		//not supported
		return
	}
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	defer s.Close()
	assert.Equal(t, nil, err)

	iter := s.SelectObject(defaultContractName, []byte("key1"), []byte("key4"))
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		//fmt.Printf("key:%s, value:%s\n", string(iter.Key()), string(iter.Value()))
	}
	assert.Equal(t, 2, count)
}

func Test_blockchainStoreImpl_TxRWSet(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	defer s.Close()
	assert.Equal(t, nil, err)
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	txid := block5.Txs[0].Header.TxId
	txRWSetFromDB, err := impl.GetTxRWSet(txid)
	assert.Equal(t, nil, err)
	assert.Equal(t, txRWSets[0].String(), txRWSetFromDB.String())
}

/*func TestBlockStoreImpl_Recovery(t *testing.T) {
	fmt.Println("test recover, please delete DB file in 20s")
	time.Sleep(20 * time.Second)
	s, err := Factory{}.NewStore(dbType, chainId)
	defer s.Close()
	assert.Equal(t, nil, err)
	fmt.Println("recover commpleted")
}*/

func Test_blockchainStoreImpl_getLastSavepoint(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	defer s.Close()
	assert.Equal(t, nil, err)
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	height, err := impl.getLastSavepoint()
	assert.Equal(t, uint64(6), height)
}

func TestBlockStoreImpl_GetTxRWSetsByHeight(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	defer s.Close()
	assert.Equal(t, nil, err)
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	txRWSetsFromDB, err := impl.GetTxRWSetsByHeight(5)
	assert.Equal(t, nil, err)
	assert.Equal(t, len(txRWSets), len(txRWSetsFromDB))
	for index, txRWSet := range txRWSetsFromDB {
		assert.Equal(t, txRWSets[index].String(), txRWSet.String())
	}
}

func TestBlockStoreImpl_GetDBHandle(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	defer s.Close()
	assert.Equal(t, nil, err)
	dbHandle := s.GetDBHandle("test")
	dbHandle.Put([]byte("a"), []byte("A"))
	value, err := dbHandle.Get([]byte("a"))
	assert.Equal(t, nil, err)
	assert.Equal(t, []byte("A"), value)
}

func Test_blockchainStoreImpl_GetBlockWith100Tx(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	block, txRWSets := createBlockAndRWSets(chainId, 7, 100)
	err = s.PutBlock(block, txRWSets)

	assert.Equal(t, nil, err)
	blockFromDB, err := s.GetBlock(7)
	assert.Equal(t, nil, err)
	assert.Equal(t, block.String(), blockFromDB.String())

	txRWSetsFromDB, err := s.GetTxRWSetsByHeight(7)
	assert.Equal(t, nil, err)
	assert.Equal(t, len(txRWSets), len(txRWSetsFromDB))
	for i := 0; i < len(txRWSets); i++ {
		assert.Equal(t, txRWSets[i].String(), txRWSetsFromDB[i].String())
	}

	blockWithRWSets, err := s.GetBlockWithRWSets(7)
	assert.Equal(t, nil, err)
	assert.Equal(t, block.String(), blockWithRWSets.Block.String())
	for i := 0; i < len(blockWithRWSets.TxRWSets); i++ {
		assert.Equal(t, txRWSets[i].String(), blockWithRWSets.TxRWSets[i].String())
	}
}

func Test_blockchainStoreImpl_recovory(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, config, log)
	//defer s.Close()
	assert.Equal(t, nil, err)
	bs, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)

	block8, txRWSets8 := createBlockAndRWSets(chainId, 8, 100)

	//1. commit wal
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:    block8,
		TxRWSets: txRWSets8,
	}
	blockWithRWSetBytes, _, err := serialization.SerializeBlock(blockWithRWSet)
	assert.Equal(t, nil, err)
	err = bs.writeLog(uint64(block8.Header.BlockHeight), blockWithRWSetBytes)
	if err != nil {
		fmt.Errorf("chain[%s] Failed to write wal, block[%d]",
			block8.Header.ChainId, block8.Header.BlockHeight)
		t.Error(err)
	}
	s.Close()

	//recovory
	s, err = factory.NewStore(chainId, config, log)
	assert.Equal(t, nil, err)
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	blockDBSavepoint, _ := impl.blockDB.GetLastSavepoint()
	assert.Equal(t, uint64(block8.Header.BlockHeight), blockDBSavepoint)

	stateDBSavepoint, _ := impl.stateDB.GetLastSavepoint()
	assert.Equal(t, uint64(block8.Header.BlockHeight), stateDBSavepoint)

	historyDBSavepoint, _ := impl.historyDB.GetLastSavepoint()
	assert.Equal(t, uint64(block8.Header.BlockHeight), historyDBSavepoint)
	s.Close()

	//check recover result
	s, err = factory.NewStore(chainId, config, log)
	blockWithRWSets, err := s.GetBlockWithRWSets(8)
	assert.Equal(t, nil, err)
	assert.Equal(t, block8.String(), blockWithRWSets.Block.String())
	for i := 0; i < len(blockWithRWSets.TxRWSets); i++ {
		assert.Equal(t, txRWSets8[i].String(), blockWithRWSets.TxRWSets[i].String())
	}
	s.Close()
}
