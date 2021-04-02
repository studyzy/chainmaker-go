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
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/binlog"
	"chainmaker.org/chainmaker-go/store/serialization"
	"path/filepath"

	"os"
	"time"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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
		DbType:      "sql",
		SqlDbConfig: sqlconfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig

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
		DbType:      "sql",
		SqlDbConfig: sqlconfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig

	return conf
}
func getlvldbConfig() *localconf.StorageConfig {
	conf := &localconf.StorageConfig{}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	conf.StorePath = path

	lvlConfig := &localconf.LevelDbConfig{
		StorePath: path,
	}
	dbConfig := &localconf.DbConfig{
		DbType:        "leveldb",
		LevelDbConfig: lvlConfig,
	}
	conf.BlockDbConfig = dbConfig
	conf.StateDbConfig = dbConfig
	conf.HistoryDbConfig = dbConfig
	conf.ResultDbConfig = dbConfig

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
	testBlockchainStoreImpl_GetBlock(t, getlvldbConfig())
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

//func Test_blockchainStoreImpl_GetLastConfigBlock(t *testing.T) {
//	var factory Factory
//	s, err := factory.newStore(chainId,config1,binlog.NewMemBinlog(),log)
//	if err != nil {
//		panic(err)
//	}
//	defer s.Close()
//	configBlock := createConfigBlock(chainId, 6)
//	err = s.PutBlock(configBlock, txRWSets)
//	assert.Equal(t, nil, err)
//	lastBlock, err := s.GetLastConfigBlock()
//	assert.Equal(t, nil, err)
//	assert.Equal(t, int64(6), lastBlock.Header.BlockHeight)
//}

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

	iter := s.SelectObject(defaultContractName, []byte("key_2"), []byte("key_4"))
	defer iter.Release()
	var count int
	for iter.Next() {
		count++
		t.Logf("key:%s, value:%s\n", string(iter.Key()), string(iter.Value()))
	}
	assert.Equal(t, 2, count)
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
	ldbConfig := getlvldbConfig()
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
