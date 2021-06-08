package common

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/statedb/statesqldb"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"runtime"
)

type StoreHelper interface {
	RollBack(*commonpb.Block, protocol.BlockchainStore) error
	BeginDbTransaction(protocol.BlockchainStore, string)
	GetPoolCapacity() int
}

type KVStoreHelper struct {
	chainId string
}

func NewKVStoreHelper(chainId string) *KVStoreHelper {
	return &KVStoreHelper{chainId: chainId}
}

func (kv *KVStoreHelper) RollBack(block *commonpb.Block, blockchainStore protocol.BlockchainStore) error {
	return nil
}

func (kv *KVStoreHelper) BeginDbTransaction(blockchainStore protocol.BlockchainStore, txKey string) {
}

func (kv *KVStoreHelper) GetPoolCapacity() int {
	return runtime.NumCPU() * 4
}

type SQLStoreHelper struct {
	chainId string
}

func NewSQLStoreHelper(chainId string) *SQLStoreHelper {
	return &SQLStoreHelper{chainId: chainId}
}

func (sql *SQLStoreHelper) RollBack(block *commonpb.Block, blockchainStore protocol.BlockchainStore) error {
	txKey := block.GetTxKey()
	err := blockchainStore.RollbackDbTransaction(txKey)
	if err != nil {
		return err
	}
	// drop database if create contract fail
	if len(block.Txs) == 0 && utils.IsManageContractAsConfigTx(block.Txs[0], true) {
		var payload commonpb.ContractMgmtPayload
		err = proto.Unmarshal(block.Txs[0].RequestPayload, &payload)
		if err == nil {
			if payload.ContractId != nil {
				dbName := statesqldb.GetContractDbName(sql.chainId, payload.ContractId.ContractName)
				blockchainStore.ExecDdlSql(payload.ContractId.ContractName, "drop database "+dbName)
			}
		}
	}
	return err
}

func (sql *SQLStoreHelper) BeginDbTransaction(blockchainStore protocol.BlockchainStore, txKey string) {
	blockchainStore.BeginDbTransaction(txKey)
}

func (sql *SQLStoreHelper) GetPoolCapacity() int {
	return 1
}