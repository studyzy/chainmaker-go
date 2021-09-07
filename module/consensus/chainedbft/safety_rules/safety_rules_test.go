/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package safetyrules

import (
	"testing"

	"chainmaker.org/chainmaker-go/consensus/chainedbft/consensus_mock"

	"github.com/stretchr/testify/require"

	blockpool "chainmaker.org/chainmaker-go/consensus/chainedbft/block_pool"
	bftUtils "chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"chainmaker.org/chainmaker/utils/v2"
)

//func TestSafetyRules_VoteRules(t *testing.T) {
//	// 1. init safety_rules
//	log := logger.GetLogger("safety_rules")
//	rootBlk := &common.Block{Header: &common.BlockHeader{
//		BlockHash:   []byte(utils.GetRandTxId()),
//		BlockHeight: 100,
//	}}
//	rootQc := &chainedbft.QuorumCert{BlockId: rootBlk.Header.BlockHash, Height: 100, Level: 100}
//	require.NoError(t, bftUtils.AddConsensusArgstoBlock(rootBlk, 100, nil), "add consensus args failed")
//	blkPool := blockpool.NewBlockPool(rootBlk, rootQc, 10)
//	safeRules := NewSafetyRules(log, blkPool, &consensus_mock.MockBlockchainStore{})
//
//	// 2. validate valid root QC
//	require.True(t, safeRules.SafeNode(101, rootQc))
//
//	// 3. validate qc but not have block data
//	qc101 := &chainedbft.QuorumCert{BlockId: []byte(utils.GetRandTxId()), Height: 101, Level: 101}
//	require.False(t, safeRules.VoteRules(102, qc101))
//
//	// 4. add 101 block to pool, but not have consensus data
//	block101 := &common.Block{Header: &common.BlockHeader{
//		BlockHash:    qc101.BlockId,
//		BlockHeight:  101,
//		PreBlockHash: rootBlk.Header.BlockHash,
//	}}
//	require.NoError(t, safeRules.blockPool.InsertBlock(block101))
//	require.False(t, safeRules.VoteRules(102, qc101))
//
//	// 5. add consensus data to 101 block and reVote qc101
//	require.NoError(t, bftUtils.AddConsensusArgstoBlock(block101, 101, nil))
//	safeRules.blockPool = blockpool.NewBlockPool(rootBlk, rootQc, 10)
//	require.NoError(t, safeRules.blockPool.InsertBlock(block101))
//	require.True(t, safeRules.VoteRules(102, qc101))
//
//	// 6. update lockedLevel in safeRules
//	safeRules.lockedLevel = 102
//	require.False(t, safeRules.VoteRules(102, qc101))
//}
//
//func TestSafetyRules_UpdateLockedQC(t *testing.T) {
//	// 1. init safety_rules
//	log := logger.GetLogger("safety_rules")
//	rootBlk := &common.Block{Header: &common.BlockHeader{
//		BlockHash:   []byte(utils.GetRandTxId()),
//		BlockHeight: 100,
//	}}
//	rootQc := &chainedbft.QuorumCert{BlockId: rootBlk.Header.BlockHash, Height: 100, Level: 100}
//	require.NoError(t, bftUtils.AddConsensusArgstoBlock(rootBlk, 100, nil), "add consensus args failed")
//	blkPool := blockpool.NewBlockPool(rootBlk, rootQc, 10)
//	safeRules := NewSafetyRules(log, blkPool)
//
//	// 2. generate three new block after rootBlock
//	blk101 := &common.Block{Header: &common.BlockHeader{
//		BlockHash:    []byte(utils.GetRandTxId()),
//		BlockHeight:  101,
//		PreBlockHash: rootQc.BlockId,
//	}}
//	qc101 := &chainedbft.QuorumCert{BlockId: blk101.Header.BlockHash, Height: 101, Level: 101}
//	require.NoError(t, bftUtils.AddConsensusArgstoBlock(blk101, 101, nil))
//	blk102 := &common.Block{Header: &common.BlockHeader{
//		BlockHash:    []byte(utils.GetRandTxId()),
//		BlockHeight:  102,
//		PreBlockHash: qc101.BlockId,
//	}}
//	qc102 := &chainedbft.QuorumCert{BlockId: blk102.Header.BlockHash, Height: 102, Level: 102}
//	require.NoError(t, bftUtils.AddConsensusArgstoBlock(blk102, 102, nil))
//	blk103 := &common.Block{Header: &common.BlockHeader{
//		BlockHash:    []byte(utils.GetRandTxId()),
//		BlockHeight:  103,
//		PreBlockHash: qc102.BlockId,
//	}}
//	qc103 := &chainedbft.QuorumCert{BlockId: blk103.Header.BlockHash, Height: 103, Level: 103}
//	require.NoError(t, bftUtils.AddConsensusArgstoBlock(blk103, 103, nil))
//
//	// 3. update qc101 but not have block data
//	safeRules.UpdateLockedQC(qc101)
//	require.EqualValues(t, 0, safeRules.lockedLevel)
//
//	// 4. update qc101 but rootBlock and preBlock is equal
//	require.NoError(t, safeRules.blockPool.InsertBlock(blk101))
//	require.NoError(t, safeRules.blockPool.InsertQC(qc101))
//	safeRules.UpdateLockedQC(qc101)
//	require.EqualValues(t, 0, safeRules.lockedLevel)
//
//	// 5. update qc102  when received block103
//	require.NoError(t, safeRules.blockPool.InsertBlock(blk102))
//	require.NoError(t, safeRules.blockPool.InsertQC(qc102))
//	safeRules.UpdateLockedQC(qc102)
//	require.EqualValues(t, 101, int(safeRules.lockedLevel))
//
//	// 6. update qc103 when received block104
//	require.NoError(t, safeRules.blockPool.InsertBlock(blk103))
//	safeRules.UpdateLockedQC(qc103)
//	require.EqualValues(t, 102, safeRules.lockedLevel)
//
//}

func TestSafetyRules_CommitRules(t *testing.T) {
	// 1. init safety_rules
	log := logger.GetLogger("safety_rules")
	rootBlk := &common.Block{Header: &common.BlockHeader{
		BlockHash:   []byte(utils.GetRandTxId()),
		BlockHeight: 100,
	}}
	rootQc := &chainedbft.QuorumCert{BlockId: rootBlk.Header.BlockHash, Height: 100, Level: 100}
	require.NoError(t, bftUtils.AddConsensusArgstoBlock(rootBlk, 100, nil), "add consensus args failed")
	blkPool := blockpool.NewBlockPool(rootBlk, rootQc, 10)
	safeRules := NewSafetyRules(log, blkPool, &consensus_mock.MockBlockchainStore{})

	// 2. generate three new block after rootBlock
	blk101 := &common.Block{Header: &common.BlockHeader{
		BlockHash:    []byte(utils.GetRandTxId()),
		BlockHeight:  101,
		PreBlockHash: rootQc.BlockId,
	}}
	qc101 := &chainedbft.QuorumCert{BlockId: blk101.Header.BlockHash, Height: 101, Level: 101}
	require.NoError(t, bftUtils.AddConsensusArgstoBlock(blk101, 101, nil))
	blk102 := &common.Block{Header: &common.BlockHeader{
		BlockHash:    []byte(utils.GetRandTxId()),
		BlockHeight:  102,
		PreBlockHash: qc101.BlockId,
	}}
	qc102 := &chainedbft.QuorumCert{BlockId: blk102.Header.BlockHash, Height: 102, Level: 102}
	require.NoError(t, bftUtils.AddConsensusArgstoBlock(blk102, 102, nil))
	blk103 := &common.Block{Header: &common.BlockHeader{
		BlockHash:    []byte(utils.GetRandTxId()),
		BlockHeight:  103,
		PreBlockHash: qc102.BlockId,
	}}
	qc103 := &chainedbft.QuorumCert{BlockId: blk103.Header.BlockHash, Height: 103, Level: 103}
	require.NoError(t, bftUtils.AddConsensusArgstoBlock(blk103, 103, nil))

	// 3. commit qc101 failed, because all ancestors have been submitted
	safeRules.blockPool.InsertBlock(blk101)
	safeRules.blockPool.InsertQC(qc101)
	done, commitBlk, commitLevel := safeRules.CommitRules(qc101)
	require.False(t, done)
	require.Nil(t, commitBlk)

	// 4. commit qc102 failed, because have one ancestors haven't been submitted
	safeRules.blockPool.InsertBlock(blk102)
	safeRules.blockPool.InsertQC(qc102)
	done, commitBlk, commitLevel = safeRules.CommitRules(qc102)
	require.True(t, done, "qc 100 should be committed")
	require.EqualValues(t, commitLevel, rootQc.Level)
	require.EqualValues(t, commitBlk.Header.BlockHash, rootBlk.Header.BlockHash)

	// 5. commit qc103 failed, because have two ancestors haven't been submitted
	safeRules.blockPool.InsertBlock(blk103)
	safeRules.blockPool.InsertQC(qc103)
	done, commitBlk, commitLevel = safeRules.CommitRules(qc103)
	require.True(t, done, "qc 101 should be committed")
	require.EqualValues(t, commitLevel, qc101.Level)
	require.EqualValues(t, commitBlk.Header.BlockHash, blk101.Header.BlockHash)
}
