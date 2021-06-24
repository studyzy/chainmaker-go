package common

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/store/statedb/statesqldb"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"runtime"
)

type KVStoreHelper struct {
	chainId string
}

func NewKVStoreHelper(chainId string) *KVStoreHelper {
	return &KVStoreHelper{chainId: chainId}
}

// KVDB do nothing
func (kv *KVStoreHelper) RollBack(block *commonpb.Block, blockchainStore protocol.BlockchainStore) error {
	return nil
}

// KVDB do nothing
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
	if len(block.Txs) == 1 && utils.IsManageContractAsConfigTx(block.Txs[0], true) {
		var payload commonpb.ContractMgmtPayload
		if err = proto.Unmarshal(block.Txs[0].RequestPayload, &payload); err != nil {
			return err
		}
		if payload.ContractId != nil {
			dbName := statesqldb.GetContractDbName(sql.chainId, payload.ContractId.ContractName)
			blockchainStore.ExecDdlSql(payload.ContractId.ContractName, "drop database "+dbName,"")
		}
	}
	return nil
}

func (sql *SQLStoreHelper) BeginDbTransaction(blockchainStore protocol.BlockchainStore, txKey string) {
	blockchainStore.BeginDbTransaction(txKey)
}

func (sql *SQLStoreHelper) GetPoolCapacity() int {
	return 1
}