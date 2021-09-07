/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package snapshot

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

type ManagerEvidence struct {
	delegate  *ManagerDelegate
	snapshots map[utils.BlockFingerPrint]*SnapshotEvidence
}

// When generating blocks, generate a Snapshot for each block, which is used as read-write set cache
func (m *ManagerEvidence) NewSnapshot(prevBlock *commonPb.Block, block *commonPb.Block) protocol.Snapshot {
	m.delegate.lock.Lock()
	defer m.delegate.lock.Unlock()
	blockHeight := block.Header.BlockHeight
	snapshotImpl := m.delegate.makeSnapshotImpl(block, blockHeight)
	evidenceSnapshot := &SnapshotEvidence{
		delegate: snapshotImpl,
	}

	// 计算前序指纹, 和当前指纹
	prevFingerPrint := utils.CalcBlockFingerPrint(prevBlock)
	fingerPrint := utils.CalcBlockFingerPrint(block)

	// 存储当前指纹的snapshot
	m.snapshots[fingerPrint] = evidenceSnapshot

	// 如果前序指纹对应的snapshot存在, 就建立snapshot的对应关系
	if prevSnapshot, ok := m.snapshots[prevFingerPrint]; ok {
		evidenceSnapshot.SetPreSnapshot(prevSnapshot)
	}

	log.Infof(
		"create snapshot at height %d, fingerPrint[%v] -> prevFingerPrint[%v]",
		blockHeight,
		fingerPrint,
		prevFingerPrint,
	)
	return evidenceSnapshot
}

func (m *ManagerEvidence) NotifyBlockCommitted(block *commonPb.Block) error {
	m.delegate.lock.Lock()
	defer m.delegate.lock.Unlock()

	log.Infof("commit snapshot at height %d", block.Header.BlockHeight)

	// 计算刚落块的区块指纹
	deleteFp := utils.CalcBlockFingerPrint(block)
	// 如果有snapshot对应的前序snapshot的指纹, 等于刚落块的区块指纹
	for _, snapshot := range m.snapshots {
		if snapshot == nil || snapshot.GetPreSnapshot() == nil {
			continue
		}
		prevFp := m.delegate.calcSnapshotFingerPrint(snapshot.GetPreSnapshot().(*SnapshotEvidence).delegate)
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
		preSnapshot, _ := snapshot.GetPreSnapshot().(*SnapshotEvidence)
		if block.Header.BlockHeight-preSnapshot.GetBlockHeight() > 8 {
			deleteOldFp := m.delegate.calcSnapshotFingerPrint(preSnapshot.delegate)
			delete(m.snapshots, deleteOldFp)
			log.Infof("delete snapshot %v at height %d while gc", deleteFp, preSnapshot.GetBlockHeight())
			snapshot.SetPreSnapshot(nil)
		}
	}
	return nil
}
