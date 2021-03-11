/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"sync"

	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
)

type ManagerImpl struct {
	lock            sync.Mutex
	snapshots       map[utils.BlockFingerPrint]*SnapshotImpl
	blockchainStore protocol.BlockchainStore
}

// When generating blocks, generate a Snapshot for each block, which is used as read-write set cache
func (m *ManagerImpl) NewSnapshot(prevBlock *commonPb.Block, block *commonPb.Block) protocol.Snapshot {
	m.lock.Lock()
	defer m.lock.Unlock()

	// If the corresponding Snapshot does not exist, create one
	blockHeight := block.Header.BlockHeight
	txCount := len(block.Txs) // as map init size
	snapshotImpl := &SnapshotImpl{
		blockchainStore: m.blockchainStore,
		sealed:          false,
		preSnapshot:     nil,

		txResultMap: make(map[string]*commonPb.Result, txCount),

		chainId:        block.Header.ChainId,
		blockHeight:    block.Header.BlockHeight,
		blockTimestamp: block.Header.BlockTimestamp,
		blockProposer:  block.Header.Proposer,
		preBlockHash:   block.Header.PreBlockHash,

		txTable:    nil,
		readTable:  make(map[string]*sv, txCount),
		writeTable: make(map[string]*sv, txCount),
	}

	// 计算前序指纹, 和当前指纹
	prevFingerPrint := utils.CalcBlockFingerPrint(prevBlock)
	fingerPrint := utils.CalcBlockFingerPrint(block)

	// 存储当前指纹的snapshot
	m.snapshots[fingerPrint] = snapshotImpl

	// 如果前序指纹对应的snapshot存在, 就建立snapshot的对应关系
	if prevSnapshot, ok := m.snapshots[prevFingerPrint]; ok {
		snapshotImpl.SetPreSnapshot(prevSnapshot)
	}

	log.Infof("create snapshot at height %d, fingerPrint[%v] -> prevFingerPrint[%v]", blockHeight, fingerPrint, prevFingerPrint)
	return snapshotImpl
}

func (m *ManagerImpl) NotifyBlockCommitted(block *commonPb.Block) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	log.Infof("commit snapshot at height %d", block.Header.BlockHeight)

	// 计算刚落块的区块指纹
	deleteFp := utils.CalcBlockFingerPrint(block)
	// 如果有snapshot对应的前序snapshot的指纹, 等于刚落块的区块指纹
	for _, snapshot := range m.snapshots {
		if snapshot == nil || snapshot.GetPreSnapshot() == nil {
			continue
		}
		prevFp := calcSnapshotFingerPrint(snapshot.GetPreSnapshot().(*SnapshotImpl))
		if deleteFp == prevFp {
			snapshot.SetPreSnapshot(nil)
		}
	}

	log.Infof("delete snapshot %v at height %d", deleteFp, block.Header.BlockHeight)
	delete(m.snapshots, deleteFp)

	// in case of switch-fork, gc too old snapshot
	for _, snapshot := range m.snapshots {
		if snapshot == nil || snapshot.GetPreSnapshot() == nil {
			continue
		}
		preSnapshot := snapshot.GetPreSnapshot().(*SnapshotImpl)
		if block.Header.BlockHeight-preSnapshot.GetBlockHeight() > 8 {
			deleteOldFp := calcSnapshotFingerPrint(preSnapshot)
			delete(m.snapshots, deleteOldFp)
			log.Infof("delete snapshot %v at height %d while gc", deleteFp, preSnapshot.blockHeight)
			snapshot.SetPreSnapshot(nil)
		}
	}
	return nil
}

func calcSnapshotFingerPrint(snapshot *SnapshotImpl) utils.BlockFingerPrint {
	if snapshot == nil {
		return ""
	}
	chainId := snapshot.chainId
	blockHeight := snapshot.blockHeight
	blockTimestamp := snapshot.blockTimestamp
	blockProposer := snapshot.blockProposer
	preBlockHash := snapshot.preBlockHash

	return utils.CalcFingerPrint(chainId, blockHeight, blockTimestamp, blockProposer, preBlockHash)
}
