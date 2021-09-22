/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"runtime"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
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
		var payload = block.Txs[0].Payload
		//if err = proto.Unmarshal(block.Txs[0].RequestPayload, &payload); err != nil {
		//	return err
		//}
		contractName := string(payload.GetParameter(syscontract.InitContract_CONTRACT_NAME.String()))
		if len(contractName) != 0 {
			//dbName := statesqldb.GetContractDbName(sql.chainId, contractName)
			//if err = blockchainStore.ExecDdlSql(contractName, "drop database "+dbName, "1"); err != nil {
			//	return err
			//}
			if err = blockchainStore.DropDatabase(contractName); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sql *SQLStoreHelper) BeginDbTransaction(blockchainStore protocol.BlockchainStore, txKey string) {
	// TODO: handle error
	blockchainStore.BeginDbTransaction(txKey) //nolint: errcheck
}

func (sql *SQLStoreHelper) GetPoolCapacity() int {
	return 1
}
