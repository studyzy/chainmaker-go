/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpool

import (
	"bytes"
	"errors"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/queue"
	"chainmaker.org/chainmaker/pb-go/v2/common"
)

//BlockNode save one block and its children
type BlockNode struct {
	block    *common.Block
	children []string // the blockHash with children's block
}

//GetBlock get block
func (bn *BlockNode) GetBlock() *common.Block {
	return bn.block
}

//GetChildren get children
func (bn *BlockNode) GetChildren() []string {
	return bn.children
}

//BlockTree maintains a consistent block tree of parent and children links
//this struct is not thread safety.
type BlockTree struct {
	idToNode       map[string]*BlockNode // store block and its' children blockHash
	heightToBlocks map[uint64][]*common.Block
	rootBlock      *common.Block // The latest block is committed to the chain
	prunedBlocks   []string      // Caches the block hash that will be deleted
	maxPrunedSize  int           // The maximum number of cached blocks that will be deleted
}

//NewBlockTree init a block tree with rootBlock, rootQC and maxPrunedSize
func NewBlockTree(rootBlock *common.Block, maxPrunedSize int) *BlockTree {
	blockTree := &BlockTree{
		idToNode:       make(map[string]*BlockNode, 10),
		rootBlock:      rootBlock,
		prunedBlocks:   make([]string, 0, maxPrunedSize),
		maxPrunedSize:  maxPrunedSize,
		heightToBlocks: make(map[uint64][]*common.Block),
	}
	blockTree.idToNode[string(rootBlock.Header.BlockHash)] = &BlockNode{
		block:    rootBlock,
		children: make([]string, 0),
	}
	blockTree.heightToBlocks[rootBlock.Header.BlockHeight] = append(
		blockTree.heightToBlocks[rootBlock.Header.BlockHeight], rootBlock)
	return blockTree
}

//InsertBlock insert block to tree
func (bt *BlockTree) InsertBlock(block *common.Block) error {
	if block == nil {
		return errors.New("block is nil")
	}
	if _, exist := bt.idToNode[string(block.Header.BlockHash)]; exist {
		return nil
	}
	if _, exist := bt.idToNode[string(block.Header.PreBlockHash)]; !exist {
		return errors.New("block's parent not exist")
	}

	bt.idToNode[string(block.Header.BlockHash)] = &BlockNode{
		block:    block,
		children: make([]string, 0),
	}
	preBlock := bt.idToNode[string(block.Header.PreBlockHash)]
	preBlock.children = append(preBlock.children, string(block.Header.BlockHash))
	bt.heightToBlocks[block.Header.BlockHeight] = append(bt.heightToBlocks[block.Header.BlockHeight], block)
	return nil
}

//GetRootBlock get root block from tree
func (bt *BlockTree) GetRootBlock() *common.Block {
	return bt.rootBlock
}

//GetBlockByID get block by block hash
func (bt *BlockTree) GetBlockByID(id string) *common.Block {
	if node, ok := bt.idToNode[id]; ok {
		return node.GetBlock()
	}
	return nil
}

//BranchFromRoot get branch from root to input block
func (bt *BlockTree) BranchFromRoot(block *common.Block) []*common.Block {
	if block == nil {
		return nil
	}
	var (
		cur    = block
		branch []*common.Block
	)
	//use block height to check
	for cur.Header.BlockHeight > bt.rootBlock.Header.BlockHeight {
		branch = append(branch, cur)
		if cur = bt.GetBlockByID(string(cur.Header.PreBlockHash)); cur == nil {
			break
		}
	}

	if cur == nil || !bytes.Equal(cur.Header.BlockHash, bt.rootBlock.Header.BlockHash) {
		return nil
	}
	for i, j := 0, len(branch)-1; i < j; i, j = i+1, j-1 {
		branch[i], branch[j] = branch[j], branch[i]
	}
	return branch
}

//PruneBlock prune block and update rootBlock
func (bt *BlockTree) PruneBlock(newRootID string) ([]string, error) {
	toPruned := bt.findBlockToPrune(newRootID)
	if toPruned == nil {
		return nil, nil
	}
	newRootBlock := bt.GetBlockByID(newRootID)
	if newRootBlock == nil {
		return nil, nil
	}
	bt.rootBlock = newRootBlock
	bt.prunedBlocks = append(bt.prunedBlocks, toPruned[0:]...)

	var pruned []string
	if len(bt.prunedBlocks) > bt.maxPrunedSize {
		num := len(bt.prunedBlocks) - bt.maxPrunedSize
		for i := 0; i < num; i++ {
			bt.cleanBlock(bt.prunedBlocks[i])
			pruned = append(pruned, bt.prunedBlocks[i])
		}
		bt.prunedBlocks = bt.prunedBlocks[num:]
	}
	return pruned, nil
}

//findBlockToPrune get blocks to prune by the newRootID
func (bt *BlockTree) findBlockToPrune(newRootID string) []string {
	if newRootID == "" || newRootID == string(bt.rootBlock.Header.BlockHash) {
		return nil
	}
	var (
		toPruned      []string
		toPrunedQueue = queue.NewLinkedQueue()
	)
	toPrunedQueue.PushBack(string(bt.rootBlock.Header.BlockHash))
	for !toPrunedQueue.IsEmpty() {
		var (
			curID string
			ok    bool
		)
		if curID, ok = toPrunedQueue.PollFront().(string); !ok {
			return nil
		}
		curNode := bt.idToNode[curID]
		for _, child := range curNode.GetChildren() {
			if child == newRootID {
				continue //save this branch
			}
			toPrunedQueue.PushBack(child)
		}
		toPruned = append(toPruned, curID)
	}
	return toPruned
}

//cleanBlock remove block from tree
func (bt *BlockTree) cleanBlock(blockId string) {
	blk := bt.idToNode[blockId]
	delete(bt.idToNode, blockId)
	if blk != nil {
		delete(bt.heightToBlocks, blk.block.Header.BlockHeight)
	}
}

func (bt *BlockTree) GetBlocks(height uint64) []*common.Block {
	return bt.heightToBlocks[height]
}

func (bt *BlockTree) Details() string {
	blkContents := bytes.NewBufferString(fmt.Sprintf("BlockTree blockNum: %d\n", len(bt.idToNode)))
	for _, blks := range bt.heightToBlocks {
		for _, blk := range blks {
			blkContents.WriteString(fmt.Sprintf("blkID: %x, blockHeight:%d\n", blk.Header.BlockHash, blk.Header.BlockHeight))
		}
	}
	return blkContents.String()
}
