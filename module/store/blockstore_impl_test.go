/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package store

import (
	"bytes"
	"path/filepath"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	storePb "chainmaker.org/chainmaker-go/pb/protogo/store"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/archive"
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/serialization"
	"github.com/tidwall/wal"

	"os"
	"time"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var chainId = "testchain1"

//var dbType = types.MySQL
//var dbType = types.LevelDb

var defaultContractName = "contract1"
var block0 = createConfigBlock(chainId, 0)
var block5 = createBlock(chainId, 5, 1)
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
var config1 = getSqlConfig()

func getSqlConfig() *localconf.StorageConfig {
	conf := &localconf.StorageConfig{}
	conf.StorePath = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	var sqlconfig = &localconf.SqlDbConfig{
		SqlDbType: "sqlite",
		Dsn:       ":memory:",
	}

	dbConfig := &localconf.DbConfig{
		Provider:    "sql",
		SqlDbConfig: sqlconfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig
	conf.ContractEventDbConfig = dbConfig
	conf.DisableContractEventDB = true
	return conf
}
func getMysqlConfig() *localconf.StorageConfig {
	conf := &localconf.StorageConfig{}
	conf.StorePath = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	var sqlconfig = &localconf.SqlDbConfig{
		SqlDbType: "mysql",
		Dsn:       "root:123456@tcp(127.0.0.1)/",
	}

	dbConfig := &localconf.DbConfig{
		Provider:    "sql",
		SqlDbConfig: sqlconfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig
	conf.DisableContractEventDB = true

	return conf
}
func getlvldbConfig(path string) *localconf.StorageConfig {
	conf := &localconf.StorageConfig{}
	if path == "" {
		path = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	}
	conf.StorePath = path

	lvlConfig := &localconf.LevelDbConfig{
		StorePath: path,
	}
	dbConfig := &localconf.DbConfig{
		Provider:      "leveldb",
		LevelDbConfig: lvlConfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig
	conf.DisableContractEventDB = true
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

var txRequestData = generateData()

func generateData() []byte {
	size := 10240
	data := make([]byte, 0, size)
	for i := 0; i < size; i++ {
		data = append(data, 'a')
	}
	return data
}

func createBlock(chainId string, height int64, txNum int) *commonPb.Block {
	block := &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:     chainId,
			BlockHeight: height,
		},
		Txs: []*commonPb.Transaction{},
	}
	for i := 0; i < txNum; i++ {
		tx := &commonPb.Transaction{
			Header: &commonPb.TxHeader{
				ChainId: chainId,
				TxType:  commonPb.TxType_INVOKE_USER_CONTRACT,
				TxId:    generateTxId(chainId, height, i),
				Sender: &acPb.SerializedMember{
					OrgId: "org1",
				},
			},
			//RequestPayload: txRequestData,
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
	block.Txs[0].Header.TxId = generateTxId(chainId, height, 0)

	return block
}
func createContractMgrPayload() []byte {
	p := commonPb.ContractMgmtPayload{
		ChainId: chainId,
		ContractId: &commonPb.ContractId{
			ContractName:    defaultContractName,
			ContractVersion: "1.0",
			RuntimeType:     commonPb.RuntimeType_EVM,
		},
		Method:      "create",
		Parameters:  nil,
		ByteCode:    nil,
		Endorsement: nil,
	}
	d, _ := p.Marshal()
	return d
}
func createInitContractBlockAndRWSets(chainId string, height int64) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := createBlock(chainId, height, 1)
	block.Txs[0].Header.TxType = commonPb.TxType_MANAGE_USER_CONTRACT
	block.Txs[0].RequestPayload = createContractMgrPayload()
	var txRWSets []*commonPb.TxRWSet
	//建表脚本在写集
	txRWset := &commonPb.TxRWSet{
		TxId: block.Txs[0].Header.TxId,
		TxWrites: []*commonPb.TxWrite{
			{
				Key:          nil,
				Value:        []byte("create table t1(name varchar(50) primary key,amount int)"),
				ContractName: defaultContractName,
			},
		},
	}
	txRWSets = append(txRWSets, txRWset)
	return block, txRWSets
}

func createBlockAndRWSets(chainId string, height int64, txNum int) (*commonPb.Block, []*commonPb.TxRWSet) {
	block := createBlock(chainId, height, txNum)
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
func Test_blockchainStoreImpl_GetBlockSqlDb(t *testing.T) {
	testBlockchainStoreImpl_GetBlock(t, config1)
}
func Test_blockchainStoreImpl_GetBlockLevledb(t *testing.T) {
	testBlockchainStoreImpl_GetBlock(t, getlvldbConfig(""))
}
func testBlockchainStoreImpl_GetBlock(t *testing.T, config *localconf.StorageConfig) {
	var funcName = "get block"
	tests := []struct {
		name  string
		block *commonPb.Block
	}{
		{funcName, createBlock(chainId, 1, 1)},
		{funcName, createBlock(chainId, 2, 1)},
		{funcName, createBlock(chainId, 3, 1)},
		{funcName, createBlock(chainId, 4, 1)},
	}
	var factory Factory
	s, err := factory.newStore(chainId, config, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	initGenesis(s)
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
func initGenesis(s protocol.BlockchainStore) {
	genesis := block0
	g := &storePb.BlockWithRWSet{Block: genesis, TxRWSets: txRWSets}
	s.InitGenesis(g)
}

func Test_blockchainStoreImpl_PutBlock(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	initGenesis(s)

	if err != nil {
		panic(err)
	}
	defer s.Close()
	txRWSets[0].TxId = block5.Txs[0].Header.TxId
	err = s.PutBlock(block5, txRWSets)
	assert.NotNil(t, err)
}

func Test_blockchainStoreImpl_HasBlock(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	initGenesis(s)
	exist, _ := s.BlockExists(block0.Header.BlockHash)
	assert.True(t, exist)

	exist, err = s.BlockExists([]byte("not exist"))
	assert.Equal(t, nil, err)
	assert.False(t, exist)
}
func init5Blocks(s protocol.BlockchainStore) {
	genesis := &storePb.BlockWithRWSet{Block: block0}
	s.InitGenesis(genesis)
	b, rw := createBlockAndRWSets(chainId, 1, 1)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 2, 2)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 3, 3)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 4, 10)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 5, 1)
	s.PutBlock(b, txRWSets)
}
func init5ContractBlocks(s protocol.BlockchainStore) {
	genesis := &storePb.BlockWithRWSet{Block: block0}
	s.InitGenesis(genesis)
	b, rw := createInitContractBlockAndRWSets(chainId, 1)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 2, 2)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 3, 3)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 4, 10)
	s.PutBlock(b, rw)
	b, rw = createBlockAndRWSets(chainId, 5, 1)
	s.PutBlock(b, rw)
}
func Test_blockchainStoreImpl_GetBlockAt(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	init5Blocks(s)
	got, err := s.GetBlock(block5.Header.BlockHeight)
	assert.Equal(t, nil, err)
	assert.Equal(t, block5.String(), got.String())
}

func Test_blockchainStoreImpl_GetLastBlock(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	init5Blocks(s)
	assert.Equal(t, nil, err)
	lastBlock, err := s.GetLastBlock()
	assert.Equal(t, nil, err)
	assert.Equal(t, block5.Header.BlockHeight, lastBlock.Header.BlockHeight)
}

func Test_blockchainStoreImpl_GetBlockByTx(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	init5Blocks(s)
	block, err := s.GetBlockByTx(generateTxId(chainId, 3, 0))
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(3), block.Header.BlockHeight)

	blockNotExist, err := s.GetBlockByTx("not_exist_txid")
	assert.Equal(t, nil, err)
	assert.Equal(t, true, blockNotExist == nil)

}

func Test_blockchainStoreImpl_GetTx(t *testing.T) {
	funcName := "has tx"
	tests := []struct {
		name  string
		block *commonPb.Block
	}{
		{funcName, createBlock(chainId, 1, 1)},
		{funcName, createBlock(chainId, 2, 1)},
		{funcName, createBlock(chainId, 3, 1)},
		{funcName, createBlock(chainId, 4, 1)},
		{funcName, createBlock(chainId, 999999, 2)},
	}

	var factory Factory
	s, err := factory.newStore(chainId, getSqlConfig(), binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	//assert.DeepEqual(t, s.GetTx(tests[0].block.Txs[0].TxId, )
	init5ContractBlocks(s)
	tx, err := s.GetTx(tests[0].block.Txs[0].Header.TxId)
	assert.Equal(t, nil, err)
	if tx == nil {
		t.Error("Error, GetTx")
	}
	//assert.Equal(t, tx.Header.TxId, generateTxId(chainId, 1, 0))
	//
	////chain not exist
	//tx, err = s.GetTx(generateTxId("not exist chain", 1, 0))
	//t.Log(tx)
	//assert.NotNil(t,  err)

}

func Test_blockchainStoreImpl_HasTx(t *testing.T) {
	funcName := "has tx"
	tests := []struct {
		name  string
		block *commonPb.Block
	}{
		{funcName, createBlock(chainId, 1, 1)},
		{funcName, createBlock(chainId, 2, 1)},
		{funcName, createBlock(chainId, 3, 1)},
		{funcName, createBlock(chainId, 4, 1)},
		{funcName, createBlock(chainId, 999999, 1)},
	}
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	init5Blocks(s)
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
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	defer s.Close()
	initGenesis(s)
	assert.Equal(t, nil, err)
	value, err := s.ReadObject(defaultContractName, []byte("key1"))
	assert.Equal(t, nil, err)
	assert.Equal(t, value, []byte("value1"))

	value, err = s.ReadObject(defaultContractName, []byte("key2"))
	assert.Equal(t, nil, err)
	assert.Equal(t, value, []byte("value2"))
}

func Test_blockchainStoreImpl_SelectObject(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, getSqlConfig(), binlog.NewMemBinlog(), log)
	defer s.Close()
	init5Blocks(s)
	assert.Equal(t, nil, err)

	iter, err := s.SelectObject(defaultContractName, []byte("key_2"), []byte("key_4"))
	assert.Nil(t, err)
	defer iter.Release()
	var count int = 0
	for iter.Next() {
		count++
		kv, _ := iter.Value()
		t.Logf("key:%s, value:%s\n", string(kv.Key), string(kv.Value))
	}
	assert.Equal(t, 3, count)
}

func Test_blockchainStoreImpl_TxRWSet(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	defer s.Close()
	init5Blocks(s)
	assert.Equal(t, nil, err)
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	txid := block5.Txs[0].Header.TxId
	txRWSetFromDB, err := impl.GetTxRWSet(txid)
	assert.Equal(t, nil, err)
	t.Log(txRWSetFromDB)
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
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	defer s.Close()
	init5Blocks(s)
	assert.Equal(t, nil, err)
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	height, err := impl.getLastSavepoint()
	assert.Equal(t, uint64(5), height)
	height, err = impl.blockDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), height)
	height, err = impl.stateDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), height)
	height, err = impl.resultDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), height)
	height, err = impl.historyDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), height)
}

func TestBlockStoreImpl_GetTxRWSetsByHeight(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	defer s.Close()
	init5Blocks(s)
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

//func TestBlockStoreImpl_GetDBHandle(t *testing.T) {
//	var factory Factory
//	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
//	defer s.Close()
//	assert.Equal(t, nil, err)
//	dbHandle := s.GetDBHandle("test")
//	dbHandle.Put([]byte("a"), []byte("A"))
//	value, err := dbHandle.Get([]byte("a"))
//	assert.Equal(t, nil, err)
//	assert.Equal(t, []byte("A"), value)
//}

func Test_blockchainStoreImpl_GetBlockWith100Tx(t *testing.T) {
	var factory Factory
	s, err := factory.newStore(chainId, config1, binlog.NewMemBinlog(), log)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	init5Blocks(s)
	block, txRWSets := createBlockAndRWSets(chainId, 6, 1)
	err = s.PutBlock(block, txRWSets)
	block, txRWSets = createBlockAndRWSets(chainId, 7, 100)
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
	blog := binlog.NewMemBinlog()
	ldbConfig := getlvldbConfig("")
	s, err := factory.newStore(chainId, ldbConfig, blog, log)
	//defer s.Close()
	assert.Equal(t, nil, err)
	bs, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	init5Blocks(s)
	block6, txRWSets6 := createBlockAndRWSets(chainId, 6, 100)

	//1. commit wal
	blockWithRWSet := &storePb.BlockWithRWSet{
		Block:    block6,
		TxRWSets: txRWSets6,
	}
	blockWithRWSetBytes, _, err := serialization.SerializeBlock(blockWithRWSet)
	assert.Equal(t, nil, err)
	err = bs.writeLog(uint64(block6.Header.BlockHeight), blockWithRWSetBytes)
	if err != nil {
		fmt.Errorf("chain[%s] Failed to write wal, block[%d]",
			block6.Header.ChainId, block6.Header.BlockHeight)
		t.Error(err)
	}
	binlogSavepoint, _ := bs.getLastSavepoint()
	assert.EqualValues(t, 6, binlogSavepoint)
	blockDBSavepoint, _ := bs.blockDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), blockDBSavepoint)

	stateDBSavepoint, _ := bs.stateDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), stateDBSavepoint)

	historyDBSavepoint, _ := bs.historyDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), historyDBSavepoint)
	resultDBSavepoint, _ := bs.resultDB.GetLastSavepoint()
	assert.Equal(t, uint64(5), resultDBSavepoint)

	s.Close()
	t.Log("start recovery db from bin log")
	//recovory
	s, err = factory.newStore(chainId, ldbConfig, blog, log)
	assert.Equal(t, nil, err)
	t.Log("db recovered")
	impl, ok := s.(*BlockStoreImpl)
	assert.Equal(t, true, ok)
	binlogSavepoint, _ = impl.getLastSavepoint()
	assert.EqualValues(t, 6, binlogSavepoint)
	blockDBSavepoint, _ = impl.blockDB.GetLastSavepoint()
	assert.EqualValues(t, 6, blockDBSavepoint)

	stateDBSavepoint, _ = impl.stateDB.GetLastSavepoint()
	assert.EqualValues(t, 6, stateDBSavepoint)

	historyDBSavepoint, _ = impl.historyDB.GetLastSavepoint()
	assert.EqualValues(t, 6, historyDBSavepoint)
	resultDBSavepoint, _ = impl.resultDB.GetLastSavepoint()
	assert.EqualValues(t, 6, resultDBSavepoint)
	s.Close()

	//check recover result
	s, err = factory.newStore(chainId, ldbConfig, blog, log)
	blockWithRWSets, err := s.GetBlockWithRWSets(6)
	assert.Equal(t, nil, err)
	assert.Equal(t, block6.String(), blockWithRWSets.Block.String())
	for i := 0; i < len(blockWithRWSets.TxRWSets); i++ {
		assert.Equal(t, txRWSets6[i].String(), blockWithRWSets.TxRWSets[i].String())
	}
	s.Close()
}

func TestWriteBinlog(t *testing.T) {
	walPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Unix()), logPath)
	writeAsync := true
	walOpt := &wal.Options{
		NoSync: writeAsync,
	}
	writeLog, err := wal.Open(walPath, walOpt)
	assert.Nil(t, err)

	err = writeLog.Write(1, []byte("100"))
	assert.Nil(t, err)
}

//
//func TestLeveldbRange(t *testing.T) {
//	db, err := leveldb.OpenFile("gossip.db", nil)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	wo := &opt.WriteOptions{Sync: true}
//	db.Put([]byte("key-1a"), []byte("value-1"), wo)
//	db.Put([]byte("key-3c"), []byte("value-3"), wo)
//	db.Put([]byte("key-4d"), []byte("value-4"), wo)
//	db.Put([]byte("key-5eff"), []byte("value-5"), wo)
//	db.Put([]byte("key-2b"), []byte("value-2"), wo)
//	iter := db.NewIterator(&util.Range{Start: []byte("key-1a"), Limit: []byte("key-3d")}, nil)
//	for iter.Next() {
//		fmt.Println(string(iter.Key()), string(iter.Value()))
//	}
//	iter.Release()
//	err = iter.Error()
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	defer db.Close()
//}

func Test_blockchainStoreImpl_Archive(t *testing.T) {
	var factory Factory
	s, err := factory.NewStore(chainId, getlvldbConfig("./"))
	assert.Equal(t, nil, err)
	defer s.Close()

	totalHeight := 301000
	archiveHeight1 := 95
	archiveHeight2 := 20
	archiveHeight3 := 26

	//Prepare block data
	blocks := make([]*commonPb.Block, 0, totalHeight)
	txRWSetMp := make(map[int64][]*commonPb.TxRWSet)
	for i := 0; i < totalHeight; i++ {
		block, txRWSet := createBlockAndRWSets(chainId, int64(i), 10)
		err = s.PutBlock(block, txRWSet)
		assert.Equal(t, nil, err)
		blocks = append(blocks, block)
		txRWSetMp[block.Header.BlockHeight] = txRWSet
	}

	verifyArchive(t, 0, blocks, txRWSetMp, s)

	//archive block height1
	err = s.ArchiveBlock(uint64(archiveHeight1))
	assert.Equal(t, nil, err)
	assert.Equal(t, uint64(archiveHeight1), s.GetArchivedPivot())

	verifyArchive(t, 10, blocks, txRWSetMp, s)

	//archive block height2 which is a config block
	err1 := s.ArchiveBlock(uint64(archiveHeight2))
	assert.True(t, err1 == nil)
	assert.Equal(t, uint64(archiveHeight2), s.GetArchivedPivot())

	verifyArchive(t, 15, blocks, txRWSetMp, s)

	//archive block height3
	err = s.ArchiveBlock(uint64(archiveHeight3))
	assert.Equal(t, nil, err)
	assert.Equal(t, uint64(archiveHeight3), s.GetArchivedPivot())

	verifyArchive(t, 25, blocks, txRWSetMp, s)

	//Prepare restore data
	blocksBytes := make([][]byte, 0, archiveHeight3-archiveHeight2+1)
	for i := archiveHeight2; i <= archiveHeight3; i++ {
		blockBytes, _, err5 := serialization.SerializeBlock(&storePb.BlockWithRWSet{
			Block:          blocks[i],
			TxRWSets:       txRWSetMp[blocks[i].Header.BlockHeight],
			ContractEvents: nil,
		})

		assert.Equal(t, nil, err5)
		blocksBytes = append(blocksBytes, blockBytes)
	}

	//restore block
	err = s.RestoreBlocks(blocksBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, uint64(archiveHeight2-1), s.GetArchivedPivot())

	verifyArchive(t, 10, blocks, txRWSetMp, s)
}

func verifyArchive(t *testing.T, confHeight uint64, blocks []*commonPb.Block,
	txRWSetMp map[int64][]*commonPb.TxRWSet, s protocol.BlockchainStore) {
	archivedPivot := s.GetArchivedPivot()

	if archivedPivot == 0 {
		verifyUnarchivedHeight(t, archivedPivot, blocks, txRWSetMp, s)
		verifyUnarchivedHeight(t, archivedPivot+1, blocks, txRWSetMp, s)
		return
	}

	//verify store apis: archived height
	verifyArchivedHeight(t, archivedPivot-1, blocks, txRWSetMp, s)

	//verify store apis: archivedPivot height
	verifyArchivedHeight(t, archivedPivot, blocks, txRWSetMp, s)

	//verify store apis: conf block height
	verifyUnarchivedHeight(t, confHeight, blocks, txRWSetMp, s)

	//verify store apis: unarchived height
	verifyUnarchivedHeight(t, archivedPivot+1, blocks, txRWSetMp, s)
}

func verifyUnarchivedHeight(t *testing.T, avBlkHeight uint64, blocks []*commonPb.Block,
	txRWSetMp map[int64][]*commonPb.TxRWSet, s protocol.BlockchainStore) {
	avBlk := blocks[avBlkHeight]
	vbHeight, err1 := s.GetHeightByHash(avBlk.Header.BlockHash)
	assert.Equal(t, nil, err1)
	assert.Equal(t, vbHeight, avBlkHeight)

	vbm, err2 := s.GetBlockMateByHash(avBlk.Header.BlockHash)
	assert.Equal(t, nil, err2)

	_, bwsInfo, err3 := serialization.SerializeBlock(&storePb.BlockWithRWSet{
		Block:          avBlk,
		TxRWSets:       txRWSetMp[avBlk.Header.BlockHeight],
		ContractEvents: nil,
	})
	assert.Equal(t, nil, err3)
	assert.True(t, bytes.Equal(vbm, bwsInfo.GetSerializedMeta()))

	vtHeight, err4 := s.GetTxHeight(avBlk.Txs[0].Header.TxId)
	assert.Equal(t, nil, err4)
	assert.Equal(t, vtHeight, avBlkHeight)

	vtBlk, err5 := s.GetBlockByTx(avBlk.Txs[0].Header.TxId)
	assert.Equal(t, nil, err5)
	assert.Equal(t, avBlk.Header.ChainId, vtBlk.Header.ChainId)

	vttx, err6 := s.GetTx(avBlk.Txs[0].Header.TxId)
	assert.Equal(t, nil, err6)
	assert.Equal(t, avBlk.Header.ChainId, vttx.Header.ChainId)

	vtBlk2, err7 := s.GetBlockByHash(avBlk.Hash())
	assert.Equal(t, nil, err7)
	assert.Equal(t, avBlk.Header.ChainId, vtBlk2.Header.ChainId)

	vtBlkRW, err8 := s.GetBlockWithRWSets(avBlk.Header.BlockHeight)
	assert.Equal(t, nil, err8)
	assert.Equal(t, avBlk.Header.ChainId, vtBlkRW.Block.Header.ChainId)
}

func verifyArchivedHeight(t *testing.T, avBlkHeight uint64, blocks []*commonPb.Block,
	txRWSetMp map[int64][]*commonPb.TxRWSet, s protocol.BlockchainStore) {
	avBlk := blocks[avBlkHeight]
	vbHeight, err1 := s.GetHeightByHash(avBlk.Header.BlockHash)
	assert.Equal(t, nil, err1)
	assert.Equal(t, vbHeight, avBlkHeight)

	vbm, err2 := s.GetBlockMateByHash(avBlk.Header.BlockHash)
	assert.Equal(t, nil, err2)

	_, bwsInfo, err3 := serialization.SerializeBlock(&storePb.BlockWithRWSet{
		Block:          avBlk,
		TxRWSets:       txRWSetMp[avBlk.Header.BlockHeight],
		ContractEvents: nil,
	})
	assert.Equal(t, nil, err3)
	assert.True(t, bytes.Equal(vbm, bwsInfo.GetSerializedMeta()))

	vtHeight, err4 := s.GetTxHeight(avBlk.Txs[0].Header.TxId)
	assert.Equal(t, nil, err4)
	assert.Equal(t, vtHeight, avBlkHeight)

	vtBlk, err5 := s.GetBlockByTx(avBlk.Txs[0].Header.TxId)
	assert.True(t, true, archive.ArchivedBlockError == err5)
	assert.True(t, vtBlk == nil)

	vttx, err6 := s.GetTx(avBlk.Txs[0].Header.TxId)
	assert.True(t, archive.ArchivedTxError == err6)
	assert.True(t, vttx == nil)

	vtBlk2, err7 := s.GetBlockByHash(avBlk.Hash())
	assert.True(t, archive.ArchivedBlockError == err7)
	assert.True(t, vtBlk2 == nil)

	vtBlkRW, err8 := s.GetBlockWithRWSets(avBlk.Header.BlockHeight)
	assert.True(t, archive.ArchivedBlockError == err8)
	assert.True(t, vtBlkRW == nil)
}
