/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	pbac "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
)

// TxScheduler schedules a transaction batch and returns a block (maybe not complete) with DAG
// TxScheduler also can run VM with a given DAG, and return results.
// It can only be called by BlockProposer
// Should have multiple implementations and adaptive mode
type TxScheduler interface {
	// schedule a transaction batch into a block with DAG
	// Return result(and read write set) of each transaction, no matter it is executed OK, or fail, or timeout.
	// For cross-contracts invoke, result(and read write set) include all contract relative.
	Schedule(block *common.Block, txBatch []*common.Transaction, snapshot Snapshot) (map[string]*common.TxRWSet, error)
	// Run VM with a given DAG, and return results.
	SimulateWithDag(block *common.Block, snapshot Snapshot) (map[string]*common.TxRWSet, map[string]*common.Result, error)
	// To halt scheduler and release VM resources.
	Halt()
}

// The simulated execution context of the transaction, providing a cache for the transaction to read and write
type TxSimContext interface {
	// Get key from cache
	Get(name string, key []byte) ([]byte, error)
	// Put key into cache
	Put(name string, key []byte, value []byte) error
	// Delete key from cache
	Del(name string, key []byte) error
	// TODO Query a series of keys from the database, not yet implemented
	Select(name string, startKey []byte, limit []byte) (Iterator, error)
	// Cross contract call, return (contract result, gas used)
	CallContract(contractId *common.ContractId, method string, byteCode []byte,
		parameter map[string]string, gasUsed uint64, refTxType common.TxType) (*common.ContractResult, common.TxStatusCode)
	// Get cross contract call result, cache for len
	GetCurrentResult() []byte
	// Get related transaction
	GetTx() *common.Transaction
	// Get related transaction
	GetBlockHeight() int64
	// Get the tx result
	GetTxResult() *common.Result
	// Set the tx result
	SetTxResult(*common.Result)
	// Get the read and write set completed by the current transaction
	GetTxRWSet() *common.TxRWSet
	// Get the creator of the contract
	GetCreator(namespace string) *pbac.SerializedMember
	// Get the invoker of the transaction
	GetSender() *pbac.SerializedMember
	// Get related blockchain store
	GetBlockchainStore() BlockchainStore
	// Get access control service
	GetAccessControl() (AccessControlProvider, error)
	// Get organization service
	GetChainNodesInfoProvider() (ChainNodesInfoProvider, error)
	// The execution sequence of the transaction, used to construct the dag,
	// indicating the number of completed transactions during transaction scheduling
	GetTxExecSeq() int
	SetTxExecSeq(int)
	// Get cross contract call deep
	GetDepth() int
}
