/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker-go/pb/protogo/common"
)

// Cache proposed blocks that are not committed yet
type ProposalCache interface {
	// Clear proposed blocks with height.
	ClearProposedBlockAt(height int64)
	// Get all proposed blocks at a specific height
	GetProposedBlocksAt(height int64) []*common.Block
	// Get proposed block with specific block hash in current consensus height.
	GetProposedBlock(b *common.Block) (*common.Block, map[string]*common.TxRWSet, map[string][]*common.ContractEvent)
	// Set porposed block in current consensus height, after it's generated or verified.
	SetProposedBlock(b *common.Block, rwSetMap map[string]*common.TxRWSet, contractEventMap map[string][]*common.ContractEvent, selfProposed bool) error
	// Get proposed block that is proposed by node itself.
	GetSelfProposedBlockAt(height int64) *common.Block
	// Get proposed block by block hash and block height
	GetProposedBlockByHashAndHeight(hash []byte, height int64) (*common.Block, map[string]*common.TxRWSet)
	// Return if a proposed block has cached in current consensus height.
	HasProposedBlockAt(height int64) bool
	// Return if this node has proposed a block as proposer.
	IsProposedAt(height int64) bool
	// To mark this node has proposed a block as proposer.
	SetProposedAt(height int64)
	// Reset propose status of this node.
	ResetProposedAt(height int64)
	// Remove proposed block in height except the specific block.
	KeepProposedBlock(hash []byte, height int64) []*common.Block
	// DiscardAboveHeight Delete blocks data greater than the baseHeight
	DiscardAboveHeight(baseHeight int64) []*common.Block
	// ClearTheBlock clean the special block in proposerCache
	ClearTheBlock(block *common.Block)
}

// Cache the latest block in ledger(DB).
type LedgerCache interface {
	// Get the latest committed block
	GetLastCommittedBlock() *common.Block
	// Set the latest committed block
	SetLastCommittedBlock(b *common.Block)
	// Return current block height
	CurrentHeight() (int64, error)
}
