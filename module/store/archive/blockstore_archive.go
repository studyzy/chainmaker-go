/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package archive

import (
	"chainmaker.org/chainmaker-go/store/blockdb"
	"chainmaker.org/chainmaker-go/store/serialization"
	"errors"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	logImpl "chainmaker.org/chainmaker-go/logger"
)

const defaultMinUnArchiveBlockSize = 300000 //about 7 days block produces

var (
	HeightNotReachError          = errors.New("target archive height not reach")
	LastHeightTooLowError        = errors.New("chain last height too low to archive")
	HeightTooLowError            = errors.New("target archive height too low")
	RestoreHeightNotMatchError   = errors.New("restore block height not match last archived height")
	InvalidateRestoreBlocksError = errors.New("invalidate restore blocks")
	ConfigBlockArchiveError      = errors.New("config block do not need archive")
	ArchivedTxError              = errors.New("archived transaction")
	ArchivedBlockError           = errors.New("archived block")
)

// ArchiveMgr provide handle to archive instances
type ArchiveMgr struct {
	sync.RWMutex
	archivedPivot           uint64
	minUnArchiveBlockHeight uint64
	blockDB                 blockdb.BlockDB

	logger *logImpl.CMLogger
}

// NewArchiveMgr construct a new `ArchiveMgr` with given chainId
func NewArchiveMgr(chainId string, blockDB blockdb.BlockDB) *ArchiveMgr {
	minUnArchiveBlockHeight := localconf.ChainMakerConfig.StorageConfig.MinUnArchiveBlockHeight
	if minUnArchiveBlockHeight <= 0 {
		minUnArchiveBlockHeight = defaultMinUnArchiveBlockSize
	}
	archiveMgr := &ArchiveMgr{
		archivedPivot:           0,
		minUnArchiveBlockHeight: minUnArchiveBlockHeight,
		blockDB:                 blockDB,
		logger:                  logImpl.GetLoggerByChain(logImpl.MODULE_STORAGE, chainId),
	}
	return archiveMgr
}

// ArchiveBlock cache a block with given block height and update batch
func (mgr *ArchiveMgr) ArchiveBlock(archiveHeight uint64) error {
	mgr.Lock()
	defer mgr.Unlock()

	lastHeight, err := mgr.blockDB.GetLastSavepoint()
	if err != nil {
		return err
	}

	archivedPivot, err := mgr.GetArchivedPivot()
	if err != nil {
		return err
	}

	//archiveHeight should between archivedPivot and lastHeight - minUnArchiveBlockHeight
	if lastHeight <= mgr.minUnArchiveBlockHeight {
		return LastHeightTooLowError
	} else if mgr.archivedPivot >= archiveHeight {
		return HeightTooLowError
	} else if archiveHeight >= lastHeight-mgr.minUnArchiveBlockHeight {
		return HeightNotReachError
	}

	if err := mgr.blockDB.ShrinkBlocks(archivedPivot+1, archiveHeight); err != nil {
		return err
	}

	mgr.logger.Infof("archived block from [%d] to [%d], block size:%d",
		mgr.archivedPivot, archiveHeight, archiveHeight-mgr.archivedPivot)

	return nil
}

// RestoreBlock restore block from outside block data
func (mgr *ArchiveMgr) RestoreBlock(blockInfos []*serialization.BlockWithSerializedInfo) error {
	mgr.Lock()
	defer mgr.Unlock()
	if blockInfos == nil || len(blockInfos) == 0 {
		mgr.logger.Warnf("retore block is empty")
		return nil
	}

	total := len(blockInfos)
	lastRestoreHeight := uint64(blockInfos[total-1].Block.Header.BlockHeight)
	if lastRestoreHeight != mgr.archivedPivot {
		mgr.logger.Errorf("restore last block height[%d] not match last archived height[%d]",
			blockInfos[total-1].Block.Header.BlockHeight, mgr.archivedPivot)
		return RestoreHeightNotMatchError
	}

	if blockInfos[0].Block.Header.BlockHeight < 0 {
		return InvalidateRestoreBlocksError
	}

	//restore block info should be continuous
	curHeight := int64(lastRestoreHeight)
	for i := 0; i < total; i++ {
		if blockInfos[total-i-1].Block.Header.BlockHeight != curHeight {
			return InvalidateRestoreBlocksError
		}
		curHeight = curHeight - 1
	}

	if err := mgr.blockDB.RestoreBlocks(blockInfos); err != nil {
		return err
	}

	mgr.logger.Infof("retore block from [%d] to [%d], block size:%d",
		lastRestoreHeight, blockInfos[0].Block.Header.BlockHeight, total)
	return nil
}

// GetArchivedPivot return restore block pivot
func (mgr *ArchiveMgr) GetArchivedPivot() (uint64, error) {
	return mgr.blockDB.GetArchivedPivot()
}

// GetMinUnArchiveBlockSize return minUnArchiveBlockHeight
func (mgr *ArchiveMgr) GetMinUnArchiveBlockSize() uint64 {
	return mgr.minUnArchiveBlockHeight
}

// SetArchivedPivot set restore block pivot
func (mgr *ArchiveMgr) SetArchivedPivot(pivot uint64) error {
	mgr.Lock()
	defer mgr.Unlock()

	if err := mgr.blockDB.SetArchivedPivot(pivot); err != nil {
		return err
	}

	mgr.archivedPivot = pivot
	return nil
}

// IsArchiveHeight set restore block pivot
func (mgr *ArchiveMgr) IsArchiveHeight(height uint64) bool {
	mgr.Lock()
	defer mgr.Unlock()
	return height <= mgr.archivedPivot
}
