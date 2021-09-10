/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"bytes"
	"fmt"
	"sync"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

var defaultHashType = "SHA256" //nolint: unused

// ProposalCache is used for cache proposal blocks
type ProposalCache struct {
	// block height -> block hash -> block with rw set
	// since one block height may have multiple block proposals
	lastProposedBlock map[uint64]map[string]*blockProposal
	rwMu              sync.RWMutex
	chainConf         protocol.ChainConf
	ledgerCache       protocol.LedgerCache
}

// blockProposal is a struct cached in ProposalCache.
// Include block, read write set map and other flags needed in Proposer module.
type blockProposal struct {
	block                *commonpb.Block              // proposal block
	rwSetMap             map[string]*commonpb.TxRWSet // read write set of this proposal block
	contractEventInfoMap map[string][]*commonpb.ContractEvent
	isSelfProposed       bool // is this block proposed by this node
	hasProposedThisRound bool // for *BFT consensus, only propose once at a round.
}

// NewProposalCache get a ProposalCache.
// One ProposalCache for one chain.
func NewProposalCache(chainConf protocol.ChainConf, ledgerCache protocol.LedgerCache) protocol.ProposalCache {
	pc := &ProposalCache{
		lastProposedBlock: make(map[uint64]map[string]*blockProposal),
		chainConf:         chainConf,
		ledgerCache:       ledgerCache,
	}
	return pc
}

// ClearProposedBlockAt clear proposed blocks with height.
func (pc *ProposalCache) ClearProposedBlockAt(height uint64) {
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()
	delete(pc.lastProposedBlock, height)
}

// GetProposedBlock get proposed block with specific block hash in current consensus height.
func (pc *ProposalCache) GetProposedBlock(b *commonpb.Block) (
	*commonpb.Block, map[string]*commonpb.TxRWSet, map[string][]*commonpb.ContractEvent) {
	if b == nil || b.Header == nil {
		return nil, nil, nil
	}
	height := b.Header.BlockHeight
	fingerPrint := utils.CalcBlockFingerPrint(b)
	// starting lock when we read the map
	pc.rwMu.RLock()
	defer pc.rwMu.RUnlock()

	if proposedBlock, ok := pc.lastProposedBlock[height][string(fingerPrint)]; ok {
		return proposedBlock.block, proposedBlock.rwSetMap, proposedBlock.contractEventInfoMap
	}
	return nil, nil, nil
}

// GetProposedBlocksAt get all proposed blocks at a specific height.
// It is possible that generate several proposal blocks in one height
// because of some unpredictable situation of consensus.
func (pc *ProposalCache) GetProposedBlocksAt(height uint64) []*commonpb.Block {
	pc.rwMu.RLock()
	defer pc.rwMu.RUnlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		blocks := make([]*commonpb.Block, 0)
		for _, proposedBlock := range proposedBlocks {
			blocks = append(blocks, proposedBlock.block)
		}
		return blocks
	}
	return nil
}

// GetProposedBlockByHashAndHeight get proposed block by block hash and block height.
func (pc *ProposalCache) GetProposedBlockByHashAndHeight(hash []byte, height uint64) (
	*commonpb.Block, map[string]*commonpb.TxRWSet) {
	if hash == nil {
		return nil, nil
	}
	// starting lock when we read the map
	pc.rwMu.RLock()
	defer pc.rwMu.RUnlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		for _, proposedBlock := range proposedBlocks {
			if bytes.Equal(proposedBlock.block.Header.BlockHash, hash) {
				return proposedBlock.block, proposedBlock.rwSetMap
			}
		}
	}
	return nil, nil
}

// SetProposedBlock set porposed block in current consensus height, after it's generated or verified.
func (pc *ProposalCache) SetProposedBlock(b *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet,
	contractEventMap map[string][]*commonpb.ContractEvent, selfPropose bool) error {
	if b == nil || b.Header == nil {
		return nil
	}
	height := b.Header.BlockHeight
	currentHeight, err := pc.ledgerCache.CurrentHeight()
	if err == nil && currentHeight >= height && height != 0 {
		// this height has committed, ignore this block
		return fmt.Errorf("block with invalid height, currentHeight: %d, blockHeight: %d", currentHeight, height)
	}
	fingerPrint := utils.CalcBlockFingerPrint(b)
	bs := &blockProposal{
		block:                b,
		rwSetMap:             rwSetMap,
		contractEventInfoMap: contractEventMap,
		isSelfProposed:       selfPropose,
		hasProposedThisRound: true,
	}
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()
	if _, ok := pc.lastProposedBlock[height]; !ok {
		pc.lastProposedBlock[height] = make(map[string]*blockProposal)
	}
	pc.lastProposedBlock[height][string(fingerPrint)] = bs
	return nil
}

func (pc *ProposalCache) ClearTheBlock(block *commonpb.Block) {
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()

	if proposedBlocks, ok := pc.lastProposedBlock[block.Header.BlockHeight]; ok {
		fingerPrint := utils.CalcBlockFingerPrint(block)
		delete(proposedBlocks, string(fingerPrint))
	}
}

// GetSelfProposedBlockAt get proposed block that is proposed by node itself.
func (pc *ProposalCache) GetSelfProposedBlockAt(height uint64) *commonpb.Block {
	pc.rwMu.RLock()
	defer pc.rwMu.RUnlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		for _, proposedBlock := range proposedBlocks {
			if proposedBlock.isSelfProposed {
				return proposedBlock.block
			}
		}
	}
	return nil
}

// HasProposedBlockAt return if a proposed block has cached in current consensus height.
func (pc *ProposalCache) HasProposedBlockAt(height uint64) bool {
	pc.rwMu.RLock()
	defer pc.rwMu.RUnlock()
	_, ok := pc.lastProposedBlock[height]
	return ok
}

// IsProposedAt return if this node has proposed a block as proposer.
func (pc *ProposalCache) IsProposedAt(height uint64) bool {
	pc.rwMu.RLock()
	defer pc.rwMu.RUnlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		for _, proposedBlock := range proposedBlocks {
			if proposedBlock.isSelfProposed && proposedBlock.hasProposedThisRound {
				return true
			}
		}
	}
	return false
}

// SetProposedAt to mark this node has proposed a block as proposer.
func (pc *ProposalCache) SetProposedAt(height uint64) {
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		for _, proposedBlock := range proposedBlocks {
			if proposedBlock.isSelfProposed {
				proposedBlock.hasProposedThisRound = true
				return
			}
		}
	}
}

// ResetProposedAt reset propose status of this node.
func (pc *ProposalCache) ResetProposedAt(height uint64) {
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		for _, proposedBlock := range proposedBlocks {
			if proposedBlock.isSelfProposed {
				proposedBlock.hasProposedThisRound = false
				return
			}
		}
	}
}

// Remove proposed block in height except the specific block.
func (pc *ProposalCache) KeepProposedBlock(hash []byte, height uint64) []*commonpb.Block {
	blocks := make([]*commonpb.Block, 0)
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()
	if proposedBlocks, ok := pc.lastProposedBlock[height]; ok {
		for _, proposedBlock := range proposedBlocks {
			if !bytes.Equal(hash, proposedBlock.block.Header.BlockHash) {
				// remove blocks except this block
				blocks = append(blocks, proposedBlock.block)
				delete(proposedBlocks, string(utils.CalcBlockFingerPrint(proposedBlock.block)))
			}
		}
	}
	return blocks
}

func (pc *ProposalCache) DiscardAboveHeight(baseHeight uint64) []*commonpb.Block {
	pc.rwMu.Lock()
	defer pc.rwMu.Unlock()
	delBlocks := make([]*commonpb.Block, 0)
	for height, blks := range pc.lastProposedBlock {
		if height <= baseHeight {
			continue
		}
		delete(pc.lastProposedBlock, height)
		for _, blkInfo := range blks {
			delBlocks = append(delBlocks, blkInfo.block)
		}
	}
	return delBlocks
}

// getHashType return hash type claimed in this chain.
func (pc *ProposalCache) getHashType() string { //nolint: unused
	if pc.chainConf == nil || pc.chainConf.ChainConfig() == nil {
		return defaultHashType
	}
	return pc.chainConf.ChainConfig().Crypto.Hash
}
