/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker-go/common/crypto/hash"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/utils"
)

func (bp *BlockProposerImpl) generateNewBlock(proposingHeight int64, preHash []byte, txBatch []*commonpb.Transaction) (*commonpb.Block, []int64, error) {
	timeLasts := make([]int64, 0)
	currentHeight, _ := bp.ledgerCache.CurrentHeight()
	lastBlock := bp.findLastBlockFromCache(proposingHeight, preHash, currentHeight)
	if lastBlock == nil {
		return nil, nil, fmt.Errorf("no pre block found [%d] (%x)", proposingHeight-1, preHash)
	}
	block := bp.initNewBlock(lastBlock)
	if block == nil {
		bp.log.Warnf("generate new block failed, block == nil")
		return nil, timeLasts, fmt.Errorf("generate new block failed, block == nil")
	}
	//if txBatch == nil {
	//	// For ChainedBFT consensus, generate an empty block if tx batch is empty.
	//	return block, timeLasts, nil
	//}

	// validate tx and verify ACL，split into 2 slice according to result
	// validatedTxs are txs passed validate and should be executed by contract
	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	var validatedTxs = txBatch

	// txScheduler handle：
	// 1. execute transaction and fill the result, include rw set digest, and remove from txBatch
	// 2. calculate dag and fill into block
	// 3. fill txs into block
	// If only part of the txBatch is filled into the Block, consider executing it again
	ssStartTick := utils.CurrentTimeMillisSeconds()
	snapshot := bp.snapshotManager.NewSnapshot(lastBlock, block)
	vmStartTick := utils.CurrentTimeMillisSeconds()
	ssLasts := vmStartTick - ssStartTick
	if bp.chainConf.ChainConfig().Contract.EnableSqlSupport {
		snapshot.GetBlockchainStore().BeginDbTransaction(block.GetTxKey())
	}
	txRWSetMap, contractEventMap, err := bp.txScheduler.Schedule(block, validatedTxs, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
	timeLasts = append(timeLasts, ssLasts, vmLasts)

	if err != nil {
		return nil, timeLasts, fmt.Errorf("schedule block(%d,%x) error %s",
			block.Header.BlockHeight, block.Header.BlockHash, err)
	}

	//if len(block.Txs) == 0 {
	//	return nil, timeLasts, fmt.Errorf("no txs in scheduled block, proposing block ends")
	//}

	err = bp.finalizeBlock(block, txRWSetMap, aclFailTxs)
	if err != nil {
		return nil, timeLasts, fmt.Errorf("finalizeBlock block(%d,%s) error %s",
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash), err)
	}
	// get txs schedule timeout and put back to txpool
	var txsTimeout = make([]*commonpb.Transaction, 0)
	if len(txRWSetMap) < len(txBatch) {
		// if tx not in txRWSetMap, tx should be put back to txpool
		for _, tx := range txBatch {
			if _, ok := txRWSetMap[tx.Header.TxId]; !ok {
				txsTimeout = append(txsTimeout, tx)
			}
		}
		bp.txPool.RetryAndRemoveTxs(txsTimeout, nil)
	}

	// cache proposed block
	bp.log.Debugf("set proposed block(%d,%x)", block.Header.BlockHeight, block.Header.BlockHash)
	if err = bp.proposalCache.SetProposedBlock(block, txRWSetMap, contractEventMap, true); err != nil {
		return block, timeLasts, err
	}
	bp.proposalCache.SetProposedAt(block.Header.BlockHeight)

	return block, timeLasts, nil
}

func (bp *BlockProposerImpl) findLastBlockFromCache(proposingHeight int64, preHash []byte, currentHeight int64) *commonpb.Block {
	var lastBlock *commonpb.Block
	if currentHeight+1 == proposingHeight {
		lastBlock = bp.ledgerCache.GetLastCommittedBlock()
	} else {
		lastBlock, _ = bp.proposalCache.GetProposedBlockByHashAndHeight(preHash, proposingHeight-1)
	}
	return lastBlock
}

func (bp *BlockProposerImpl) finalizeBlock(block *commonpb.Block,
	txRWSetMap map[string]*commonpb.TxRWSet,
	aclFailTxs []*commonpb.Transaction) error {
	hashType := bp.chainConf.ChainConfig().Crypto.Hash

	if aclFailTxs != nil && len(aclFailTxs) > 0 {
		// append acl check failed txs to the end of block.Txs
		block.Txs = append(block.Txs, aclFailTxs...)
	}

	// TxCount contains acl verify failed txs and invoked contract txs
	txCount := len(block.Txs)
	block.Header.TxCount = int64(txCount)

	// TxRoot/RwSetRoot
	var err error
	txHashes := make([][]byte, txCount)
	for i, tx := range block.Txs {
		// finalize tx, put rwsethash into tx.Result
		rwSet := txRWSetMap[tx.Header.TxId]
		if rwSet == nil {
			rwSet = &commonpb.TxRWSet{
				TxId:     tx.Header.TxId,
				TxReads:  nil,
				TxWrites: nil,
			}
		}
		rwSetHash, err := utils.CalcRWSetHash(hashType, rwSet)
		if err != nil {
			return err
		}
		if tx.Result == nil {
			// in case tx.Result is nil, avoid panic
			e := fmt.Errorf("tx(%s) result == nil", tx.Header.TxId)
			bp.log.Error(e.Error())
			return e
		}
		tx.Result.RwSetHash = rwSetHash
		// calculate complete tx hash, include tx.Header, tx.Payload, tx.Result
		txHash, err := utils.CalcTxHash(hashType, tx)
		if err != nil {
			return err
		}
		txHashes[i] = txHash
	}

	block.Header.TxRoot, err = hash.GetMerkleRoot(hashType, txHashes)
	if err != nil {
		bp.log.Warnf("get tx merkle root error %s", err)
		return err
	}
	block.Header.RwSetRoot, err = utils.CalcRWSetRoot(hashType, block.Txs)
	if err != nil {
		bp.log.Warnf("get rwset merkle root error %s", err)
		return err
	}

	// DagDigest
	dagHash, err := utils.CalcDagHash(hashType, block.Dag)
	if err != nil {
		bp.log.Warnf("get dag hash error %s", err)
		return err
	}
	block.Header.DagHash = dagHash

	return nil
}

func (bp *BlockProposerImpl) initNewBlock(lastBlock *commonpb.Block) *commonpb.Block {
	// get node pk from identity
	proposer, err := bp.identity.Serialize(true)
	if err != nil {
		bp.log.Warnf("identity serialize failed, %s", err)
		return nil
	}
	preConfHeight := lastBlock.Header.PreConfHeight
	// if last block is config block, then this block.preConfHeight is last block height
	if utils.IsConfBlock(lastBlock) {
		preConfHeight = lastBlock.Header.BlockHeight
	}

	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        bp.chainId,
			BlockHeight:    lastBlock.Header.BlockHeight + 1,
			PreBlockHash:   lastBlock.Header.BlockHash,
			BlockHash:      nil,
			PreConfHeight:  preConfHeight,
			BlockVersion:   bp.getChainVersion(),
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: utils.CurrentTimeSeconds(),
			Proposer:       proposer,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag:            &commonpb.DAG{},
		Txs:            nil,
		AdditionalData: nil,
	}
	return block
}
