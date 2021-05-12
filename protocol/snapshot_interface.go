/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
)

// Snapshot management container to manage chained snapshots
type SnapshotManager interface {
	// Create ContractStore at the current block height
	NewSnapshot(prevBlock *common.Block, block *common.Block) Snapshot

	//Once the block is submitted, notify the snapshot to clean up
	NotifyBlockCommitted(block *common.Block) error
}

//Snapshot is a chain structure that saves the read and write cache information of the blocks that are not in the library
type Snapshot interface {

	// Get database for virtual machine access
	GetBlockchainStore() BlockchainStore

	//Read the key from the current snapshot and the previous snapshot
	GetKey(txExecSeq int, contractName string, key []byte) ([]byte, error)

	// After the scheduling is completed, get the read and write set from the current snapshot
	GetTxRWSetTable() []*common.TxRWSet

	// After the scheduling is completed, get the result from the current snapshot
	GetTxResultMap() map[string]*common.Result

	// Get exec seq for snapshot
	GetSnapshotSize() int

	// After the scheduling is completed, obtain the transaction sequence table from the current snapshot
	GetTxTable() []*common.Transaction

	// Get previous snapshot
	GetPreSnapshot() Snapshot

	// Set previous snapshot
	SetPreSnapshot(Snapshot)

	// Get Block Height for current snapshot
	GetBlockHeight() int64

	// Get Block Proposer for current snapshot
	GetBlockProposer() []byte

	// If the transaction can be added to the snapshot after the conflict dependency is established
	// Even if an exception occurs when the transaction is handed over to the virtual machine module,
	// the transaction is still packed into a block, but the read-write set of the transaction is left empty.
	// This situation includes:
	// 1 wrong txtype is used,
	// 2 parameter error occurs when parsing querypayload and transactpayload,
	// 3 virtual machine runtime throws panic,
	// 4 smart contract byte code actively throws panic
	// The second bool parameter here indicates whether the above exception has occurred
	ApplyTxSimContext(TxSimContext, bool) (bool, int)

	// Build a dag for all transactions that have resolved the read-write conflict dependencies
	BuildDAG() *common.DAG

	// If snapshot is sealed, no more transaction will be added into snapshot
	IsSealed() bool
	Seal()
}
