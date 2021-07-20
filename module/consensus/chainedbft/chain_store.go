/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"bytes"
	"fmt"

	blockpool "chainmaker.org/chainmaker-go/consensus/chainedbft/block_pool"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	"chainmaker.org/chainmaker-go/logger"
	commonErrors "chainmaker.org/chainmaker/common/errors"
	"chainmaker.org/chainmaker/pb-go/common"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/consensus/chainedbft"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
)

// Access data on the chain and in the cache, commit block data on the chain
type chainStore struct {
	logger          *logger.CMLogger
	server          *ConsensusChainedBftImpl
	ledger          protocol.LedgerCache     // Query of the latest status on the chain
	blockCommitter  protocol.BlockCommitter  // Processing block committed on the chain
	blockChainStore protocol.BlockchainStore // Provide information queries on the chain

	//rwMtx            sync.RWMutex             // Only the following three elements are protected
	commitLevel      uint64                   // The latest block level on the chain
	commitHeight     uint64                   // The latest block height on the chain
	commitQuorumCert *chainedbftpb.QuorumCert // The latest committed QC on the chain

	blockPool *blockpool.BlockPool // Cache block and QC information
}

func initChainStore(server *ConsensusChainedBftImpl) (*chainStore, error) {
	bestBlock := server.ledgerCache.GetLastCommittedBlock()
	if bestBlock.Header.BlockHeight == 0 {
		if err := initGenesisBlock(bestBlock); err != nil {
			return nil, err
		}
	}
	bestBlkQCBz := utils.GetQCFromBlock(bestBlock)
	if len(bestBlkQCBz) == 0 {
		return nil, fmt.Errorf("get qc from block failed [%d:%x]", bestBlock.Header.BlockHeight, bestBlock.Header.BlockHash)
	}
	var bestBlkQC *chainedbftpb.QuorumCert
	if err := proto.Unmarshal(bestBlkQCBz, bestBlkQC); err != nil {
		return nil, err
	}

	chainStore := &chainStore{
		server:          server,
		ledger:          server.ledgerCache,
		logger:          server.logger,
		blockCommitter:  server.blockCommitter,
		blockChainStore: server.store,

		commitLevel:      bestBlkQC.GetLevel(),
		commitHeight:     bestBlock.GetHeader().GetBlockHeight(),
		commitQuorumCert: bestBlkQC,
		blockPool:        blockpool.NewBlockPool(bestBlock, bestBlkQC, 20),
	}
	chainStore.logger.Debugf("init chainStore by bestBlock, height: %d, hash: %x", bestBlock.Header.BlockHeight, bestBlock.Header.BlockHash)
	return chainStore, nil
}

func initGenesisBlock(block *common.Block) error {
	qcForGenesis := &chainedbftpb.QuorumCert{
		Votes:   []*chainedbftpb.VoteData{},
		BlockId: block.Header.BlockHash,
	}
	qcData, err := proto.Marshal(qcForGenesis)
	if err != nil {
		return fmt.Errorf("openChainStore failed, marshal genesis qc, err %v", err)
	}
	if err = utils.AddQCtoBlock(block, qcData); err != nil {
		return fmt.Errorf("openChainStore failed, add genesis qc, err %v", err)
	}
	if err = utils.AddConsensusArgstoBlock(block, 0, nil); err != nil {
		return fmt.Errorf("openChainStore failed, add genesis args, err %v", err)
	}
	return nil
}

func (cs *chainStore) updateCommitCacheInfo(bestBlock *common.Block) error {
	qc := cs.blockPool.GetQCByID(string(bestBlock.Header.BlockHash))
	if qc == nil {
		return fmt.Errorf("not find committed block's qc from block[%d:%x]",
			bestBlock.Header.BlockHeight, bestBlock.Header.BlockHash)
	}
	cs.commitLevel = qc.GetLevel()
	cs.commitHeight = bestBlock.GetHeader().GetBlockHeight()
	cs.commitQuorumCert = qc
	return nil
}

func (cs *chainStore) getCommitQC() *chainedbftpb.QuorumCert {
	return cs.commitQuorumCert
}

func (cs *chainStore) getCommitHeight() uint64 {
	return cs.commitHeight
}

func (cs *chainStore) getCommitLevel() uint64 {
	return cs.commitLevel
}

func (cs *chainStore) insertBlock(block *common.Block, curLevel uint64) error {
	if block == nil {
		return fmt.Errorf("insertBlock failed, nil block")
	}
	if exist := cs.blockPool.GetBlockByID(string(block.GetHeader().GetBlockHash())); exist != nil {
		return nil
	}
	var (
		err       error
		prevBlock *common.Block
	)
	if rootBlockQc := cs.blockPool.GetRootQC(); curLevel <= rootBlockQc.GetLevel() {
		return fmt.Errorf("insertBlock failed, older block, blkLevel: %d, rootLevel: %d", curLevel, rootBlockQc.Level)
	}
	if prevBlock = cs.blockPool.GetBlockByID(string(block.GetHeader().GetPreBlockHash())); prevBlock == nil {
		return fmt.Errorf("insertBlock failed, get previous block is nil")
	}
	if prevBlock.GetHeader().GetBlockHeight()+1 != block.GetHeader().GetBlockHeight() {
		return fmt.Errorf("insertBlock failed, invalid block height [%v], expected [%v]", block.GetHeader().GetBlockHeight(),
			prevBlock.GetHeader().BlockHeight+1)
	}
	if preQc := cs.blockPool.GetQCByID(string(prevBlock.Header.BlockHash)); preQc != nil && preQc.GetLevel() >= curLevel {
		return fmt.Errorf("insertBlock failed, invalid block level, blkLevel: %d, prevQc: %v", curLevel, preQc)
	}
	if err = cs.blockPool.InsertBlock(block); err != nil {
		return fmt.Errorf("insertBlock failed: %s, failed to insert block %v", err, block.GetHeader().GetBlockHeight())
	}
	return nil
}

func (cs *chainStore) getBlocks(height uint64) []*common.Block {
	return cs.blockPool.GetBlocks(height)
}

func (cs *chainStore) commitBlock(block *common.Block) (lastCommitted *common.Block, lastCommittedLevel uint64, err error) {
	var (
		qcData []byte
		blocks []*common.Block
		qc     *chainedbftpb.QuorumCert
	)
	if blocks = cs.blockPool.BranchFromRoot(block); blocks == nil {
		return nil, 0, fmt.Errorf("commit block failed, no block to be committed")
	}
	cs.logger.Infof("commit BranchFromRoot blocks contains [%v:%v]", blocks[0].Header.BlockHeight, blocks[len(blocks)-1].Header.BlockHeight)

	for _, blk := range blocks {
		if qc = cs.blockPool.GetQCByID(string(blk.GetHeader().GetBlockHash())); qc == nil {
			return lastCommitted, lastCommittedLevel, fmt.Errorf("commit block failed, get qc for block is nil")
		}
		if qcData, err = proto.Marshal(qc); err != nil {
			return lastCommitted, lastCommittedLevel, fmt.Errorf("commit block failed, marshal qc at height [%v], err %v",
				blk.GetHeader().GetBlockHeight(), err)
		}

		newBlock := proto.Clone(blk).(*common.Block)
		if err = utils.AddQCtoBlock(newBlock, qcData); err != nil {
			cs.logger.Errorf("commit block failed, add qc to block err, %v", err)
			return lastCommitted, lastCommittedLevel, err
		}
		if err = cs.blockCommitter.AddBlock(newBlock); err == commonErrors.ErrBlockHadBeenCommited {
			hadCommitBlock, getBlockErr := cs.blockChainStore.GetBlock(newBlock.GetHeader().GetBlockHeight())
			if getBlockErr != nil {
				cs.logger.Errorf("commit block failed, block had been committed, get block err, %v",
					getBlockErr)
				return lastCommitted, lastCommittedLevel, getBlockErr
			}
			if !bytes.Equal(hadCommitBlock.GetHeader().GetBlockHash(), newBlock.GetHeader().GetBlockHash()) {
				cs.logger.Errorf("commit block failed, block had been committed, hash unequal err, %v",
					getBlockErr)
				return lastCommitted, lastCommittedLevel, fmt.Errorf("commit block failed, block had been commited, hash unequal")
			}
		} else if err != nil {
			cs.logger.Errorf("commit block failed, add block err, %v", err)
			return lastCommitted, lastCommittedLevel, err
		}
		lastCommitted = newBlock
		lastCommittedLevel = qc.Level
	}
	if err = cs.pruneBlockStore(string(block.GetHeader().GetBlockHash())); err != nil {
		cs.logger.Errorf("commit block failed, prunning block store err, %v", err)
		return lastCommitted, lastCommittedLevel, err
	}
	cs.logger.Debugf("end commit block, lastCommitBlock:[%d:%x], lastCommitLevel: %d",
		lastCommitted.Header.BlockHeight, lastCommitted.Header.BlockHash, lastCommittedLevel)
	return lastCommitted, lastCommittedLevel, nil
}

func (cs *chainStore) pruneBlockStore(nextRootID string) error {
	err := cs.blockPool.PruneBlock(nextRootID)
	return err
}

// insertQC Only the QC that has received block data will be stored
func (cs *chainStore) insertQC(qc *chainedbftpb.QuorumCert) error {
	if qc == nil {
		return fmt.Errorf("insert qc failed, input nil qc")
	}

	if qc.EpochId != cs.server.smr.getEpochId() {
		// When the generation switches, the QC of the rootBlock is added again,
		// and the rootQC is not consistent with the current generation ID of the node
		if hasQC, err := cs.getQC(string(qc.BlockId), qc.Height); hasQC != nil || err != nil {
			cs.logger.Warnf("find qc:[%x], height:[%d], err: %v", qc.BlockId, qc.Height, err)
			return nil
		}
		return fmt.Errorf("insert qc failed, input err qc.epochid: [%v], node epochID: [%v]",
			qc.EpochId, cs.server.smr.getEpochId())
	}
	if err := cs.blockPool.InsertQC(qc); err != nil {
		return fmt.Errorf("insert qc failed, err, %v", err)
	}
	return nil
}

func (cs *chainStore) insertCompletedBlock(block *common.Block) error {
	if block.GetHeader().GetBlockHeight() <= cs.getCommitHeight() {
		return nil
	}
	if err := cs.updateCommitCacheInfo(block); err != nil {
		return fmt.Errorf("insertCompleteBlock failed, update store commit cache info err %v", err)
	}
	if err := cs.blockPool.InsertBlock(block); err != nil {
		return err
	}
	// todo. may be delete the line
	if err := cs.blockPool.InsertQC(cs.commitQuorumCert); err != nil {
		return err
	}
	err := cs.pruneBlockStore(string(block.GetHeader().GetBlockHash()))
	return err
}

func (cs *chainStore) getBlock(id string, height uint64) (*common.Block, error) {
	if block := cs.blockPool.GetBlockByID(id); block != nil {
		return block, nil
	}
	block, err := cs.blockChainStore.GetBlock(height)
	return block, err
}

func (cs *chainStore) getBlockByHash(blkHash []byte) *common.Block {
	if block := cs.blockPool.GetBlockByID(string(blkHash)); block != nil {
		return block
	}
	if block, err := cs.blockChainStore.GetBlockByHash(blkHash); err == nil && block != nil {
		return block
	}
	return nil
}

func (cs *chainStore) getCurrentQC() *chainedbftpb.QuorumCert {
	return cs.blockPool.GetHighestQC()
}

func (cs *chainStore) getCurrentCertifiedBlock() *common.Block {
	return cs.blockPool.GetHighestCertifiedBlock()
}

func (cs *chainStore) getRootLevel() (uint64, error) {
	return utils.GetLevelFromQc(cs.blockPool.GetRootBlock())
}

func (cs *chainStore) getQC(id string, height uint64) (*chainedbftpb.QuorumCert, error) {
	if qc := cs.blockPool.GetQCByID(id); qc != nil {
		return qc, nil
	}
	block, err := cs.blockChainStore.GetBlock(height)
	if err != nil {
		return nil, fmt.Errorf("get qc failed, get block fail at height [%v]", height)
	}
	qcData := utils.GetQCFromBlock(block)
	if qcData == nil {
		return nil, fmt.Errorf("get qc failed, nil qc from block at height [%v]", height)
	}
	qc := new(chainedbftpb.QuorumCert)
	if err = proto.Unmarshal(qcData, qc); err != nil {
		return nil, fmt.Errorf("get qc failed, unmarshal qc from a block err %v", err)
	}
	return qc, nil
}
