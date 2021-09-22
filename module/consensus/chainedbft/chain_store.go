/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"bytes"
	"fmt"
	"sync"

	blockpool "chainmaker.org/chainmaker-go/consensus/chainedbft/block_pool"
	"chainmaker.org/chainmaker-go/consensus/chainedbft/utils"
	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	chainedbftpb "chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

// Access data on the chain and in the cache, commit block data on the chain
type chainStore struct {
	logger          *logger.CMLogger
	server          *ConsensusChainedBftImpl
	ledger          protocol.LedgerCache     // Query of the latest status on the chain
	blockCommitter  protocol.BlockCommitter  // Processing block committed on the chain
	blockChainStore protocol.BlockchainStore // Provide information queries on the chain

	rwMtx            sync.RWMutex             // Only the following three elements are protected
	commitLevel      uint64                   // The latest block level on the chain
	commitHeight     uint64                   // The latest block height on the chain
	commitQuorumCert *chainedbftpb.QuorumCert // The latest committed QC on the chain

	blockPool *blockpool.BlockPool // Cache block and QC information
}

func openChainStore(ledger protocol.LedgerCache, blockCommitter protocol.BlockCommitter,
	store protocol.BlockchainStore, server *ConsensusChainedBftImpl, logger *logger.CMLogger) (*chainStore, error) {

	chainStore := &chainStore{
		ledger:          ledger,
		logger:          logger,
		server:          server,
		blockCommitter:  blockCommitter,
		blockChainStore: store,
	}
	bestBlock := ledger.GetLastCommittedBlock()
	if bestBlock == nil {
		return nil, fmt.Errorf("openChainStore failed, get best block from ledger")
	}
	if bestBlock.Header.BlockHeight == 0 {
		if err := initGenesisBlock(bestBlock); err != nil {
			return nil, err
		}
	}
	if err := chainStore.updateCommitCacheInfo(bestBlock); err != nil {
		return nil, fmt.Errorf("openChainStore failed, update commit cache info, err %v", err)
	}

	logger.Debugf("init chainStore by bestBlock, height: %d, hash: %x",
		bestBlock.Header.BlockHeight, bestBlock.Header.BlockHash)
	chainStore.blockPool = blockpool.NewBlockPool(bestBlock, chainStore.getCommitQC(), 20)
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
	var (
		lastQC []byte
		qc     = new(chainedbftpb.QuorumCert)
	)
	if lastQC = utils.GetQCFromBlock(bestBlock); len(lastQC) == 0 {
		return fmt.Errorf("nil qc from best block at height %v ", bestBlock.GetHeader().GetBlockHeight())
	}
	if err := proto.Unmarshal(lastQC, qc); err != nil {
		return fmt.Errorf("unmarshal qc from best block failed, err %v", err)
	}
	cs.rwMtx.Lock()
	defer cs.rwMtx.Unlock()
	cs.commitHeight = uint64(bestBlock.GetHeader().GetBlockHeight())
	cs.commitLevel = qc.GetLevel()
	cs.commitQuorumCert = qc
	return nil
}

func (cs *chainStore) getCommitQC() *chainedbftpb.QuorumCert {
	cs.rwMtx.RLock()
	defer cs.rwMtx.RUnlock()
	return cs.commitQuorumCert
}

func (cs *chainStore) getCommitHeight() uint64 {
	cs.rwMtx.RLock()
	defer cs.rwMtx.RUnlock()
	return cs.commitHeight
}

func (cs *chainStore) getCommitLevel() uint64 {
	cs.rwMtx.RLock()
	defer cs.rwMtx.RUnlock()
	return cs.commitLevel
}

func (cs *chainStore) insertBlock(block *common.Block) error {
	if block == nil {
		return fmt.Errorf("insertBlock failed, nil block")
	}
	if exist := cs.blockPool.GetBlockByID(string(block.GetHeader().GetBlockHash())); exist != nil {
		return nil
	}
	var (
		err         error
		curLevel    uint64
		prevBlock   *common.Block
		rootBlockQc = new(chainedbftpb.QuorumCert)
	)
	if curLevel, err = utils.GetLevelFromBlock(block); err != nil {
		return fmt.Errorf("insertBlock failed, get level from block fail, %v", err)
	}
	if err = proto.Unmarshal(utils.GetQCFromBlock(cs.blockPool.GetRootBlock()), rootBlockQc); err != nil {
		return fmt.Errorf("insertBlock failed, proto unmarshal fail, %v", err)
	}
	if curLevel <= rootBlockQc.GetLevel() {
		return fmt.Errorf("insertBlock failed, older block")
	}
	if prevBlock = cs.blockPool.GetBlockByID(string(block.GetHeader().GetPreBlockHash())); prevBlock == nil {
		return fmt.Errorf("insertBlock failed, get previous block is nil")
	}
	if prevBlock.GetHeader().GetBlockHeight()+1 != block.GetHeader().GetBlockHeight() {
		return fmt.Errorf("insertBlock failed, invalid block height [%v], expected [%v]", block.GetHeader().GetBlockHeight(),
			prevBlock.GetHeader().BlockHeight+1)
	}

	preQc := new(chainedbftpb.QuorumCert)
	if err = proto.Unmarshal(utils.GetQCFromBlock(prevBlock), preQc); err != nil {
		return fmt.Errorf("insertBlock failed, proto unmarshal fail, %v", err)
	}
	if preQc.GetLevel() >= curLevel {
		return fmt.Errorf("insertBlock failed, invalid block level")
	}
	if err = cs.blockPool.InsertBlock(block); err != nil {
		return fmt.Errorf("insertBlock failed: %s, failed to insert block %v", err, block.GetHeader().GetBlockHeight())
	}
	return nil
}

func (cs *chainStore) commitBlock(block *common.Block) (lastCommitted *common.Block,
	lastCommittedLevel uint64, err error) {
	var (
		qcData []byte
		blocks []*common.Block
		qc     *chainedbftpb.QuorumCert
	)
	if blocks = cs.blockPool.BranchFromRoot(block); blocks == nil {
		return nil, 0, fmt.Errorf("commit block failed, no block to be committed")
	}
	cs.logger.Infof("commit BranchFromRoot blocks contains [%v:%v]",
		blocks[0].Header.BlockHeight, blocks[len(blocks)-1].Header.BlockHeight)

	for _, blk := range blocks {
		if qc = cs.blockPool.GetQCByID(string(blk.GetHeader().GetBlockHash())); qc == nil {
			return lastCommitted, lastCommittedLevel, fmt.Errorf("commit block failed, get qc for block is nil")
		}
		if qcData, err = proto.Marshal(qc); err != nil {
			return lastCommitted, lastCommittedLevel, fmt.Errorf("commit block failed, marshal qc at height [%v], err %v",
				blk.GetHeader().GetBlockHeight(), err)
		}

		var (
			newBlock *common.Block
			ok       bool
		)
		if newBlock, ok = proto.Clone(blk).(*common.Block); !ok {
			cs.logger.Errorf("block clone error")
			return lastCommitted, lastCommittedLevel, fmt.Errorf("commit block failed, clone block failed")
		}
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
				return lastCommitted, lastCommittedLevel, fmt.Errorf(
					"commit block failed, block had been committed, hash unequal")
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
	//cs.logger.Debugf("chainStore blockPool content: %s", cs.blockPool.Details())
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
		if hasQC, _ := cs.getQC(string(qc.BlockId), qc.Height); hasQC != nil {
			cs.logger.Debugf("find qc:[%x], height:[%d]", qc.BlockId, qc.Height)
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
	if block.GetHeader().GetBlockHeight() <= uint64(cs.getCommitHeight()) {
		return nil
	}
	if err := cs.updateCommitCacheInfo(block); err != nil {
		return fmt.Errorf("insertCompleteBlock failed, update store commit cache info err %v", err)
	}
	if err := cs.blockPool.InsertBlock(block); err != nil {
		return err
	}
	if err := cs.blockPool.InsertQC(cs.getCommitQC()); err != nil {
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
