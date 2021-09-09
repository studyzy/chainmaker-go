// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	// default 20 blocks per batch
	blocksPerBatch = 20
	// Send Archive Block Request timeout
	archiveBlockRequestTimeout = 20 // 20s
)

func newDumpCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "dump blockchain data",
		Long:  "dump blockchain data to off-chain storage and delete on-chain data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbType != defaultDbType {
				return fmt.Errorf("unsupport database type %s", dbType)
			}

			// try target is block height
			if height, err := strconv.ParseUint(target, 10, 64); err == nil {
				return runDumpByHeightCMD(height)
			}

			// try target is date
			loc, err := time.LoadLocation("Local")
			if err != nil {
				return err
			}
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", target, loc); err == nil {
				height, err := calcTargetHeightByTime(t)
				if err != nil {
					return err
				}
				return runDumpByHeightCMD(height)
			}

			return fmt.Errorf("invalid --target %s, eg. 100 (block height) or \"2006-01-02 15:04:05\" (date)", target)
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId, flagDbType, flagDbDest, flagTarget, flagBlocks, flagSecretKey,
	})

	return cmd
}

// runDumpByHeightCMD `dump` command implementation
func runDumpByHeightCMD(targetBlkHeight uint64) error {
	//// 1.Chain Client
	cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
	if err != nil {
		return err
	}
	defer cc.Stop()

	//// 2.Database
	db, err := initDb()
	if err != nil {
		return err
	}
	locker := mysql.NewDbLocker(db, "cmc", mysql.DefaultLockLeaseAge)
	locker.Lock()
	defer locker.UnLock()

	//// 3.Validation, block height etc.
	archivedBlkHeightOnChain, err := cc.GetArchivedBlockHeight()
	if err != nil {
		return err
	}
	archivedBlkHeightOffChain, err := model.GetArchivedBlockHeight(db)
	if err != nil {
		return err
	}
	currentBlkHeightOnChain, err := cc.GetCurrentBlockHeight()
	if err != nil {
		return err
	}

	err = validateDump(archivedBlkHeightOnChain, archivedBlkHeightOffChain, currentBlkHeightOnChain, targetBlkHeight)
	if err != nil {
		return err
	}

	//// 4.Store & Archive Blocks
	var barCount = targetBlkHeight - archivedBlkHeightOnChain
	if blocks < barCount {
		barCount = blocks
	}
	progress := uiprogress.New()
	bar := progress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Archiving Blocks (%d/%d)", b.Current(), barCount)
	})
	progress.Start()
	defer progress.Stop()
	var batchStartBlkHeight, batchEndBlkHeight = archivedBlkHeightOnChain + 1, archivedBlkHeightOnChain + 1
	if archivedBlkHeightOnChain == 0 {
		batchStartBlkHeight = 0
	}
	for processedBlocks := uint64(0); targetBlkHeight >= batchEndBlkHeight && processedBlocks < blocks; processedBlocks++ {
		if batchEndBlkHeight-batchStartBlkHeight >= blocksPerBatch {
			if err := runBatch(cc, db, batchStartBlkHeight, batchEndBlkHeight); err == nil {
				batchStartBlkHeight = batchEndBlkHeight
			} else if !strings.Contains(err.Error(), configBlockArchiveErrorString) {
				fmt.Printf("Warning: %s\n", err)
				return nil
			}
		}

		batchEndBlkHeight++
		bar.Incr()
	}
	// do the rest of blocks
	return runBatch(cc, db, batchStartBlkHeight, batchEndBlkHeight)
}

// validateDump basic params validation
func validateDump(archivedBlkHeightOnChain, archivedBlkHeightOffChain, currentBlkHeightOnChain,
	targetBlkHeight uint64) error {
	// target block height already archived, do nothing.
	if targetBlkHeight <= archivedBlkHeightOffChain {
		return errors.New("target block height already archived")
	}

	// required archived block height off-chain == archived block height on-chain
	if archivedBlkHeightOffChain != archivedBlkHeightOnChain {
		return errors.New("required archived block height off-chain == archived block height on-chain")
	}

	// required current block height >= target block height
	if currentBlkHeightOnChain < targetBlkHeight {
		return errors.New("required current block height >= target block height")
	}
	return nil
}

// runBatch Run a batch job
// NOTE: Include startBlk, exclude endBlk
func runBatch(cc *sdk.ChainClient, db *gorm.DB, startBlk, endBlk uint64) error {
	// check if create table
	for blk := startBlk; blk < endBlk; blk++ {
		// blk is first row of new table, create new table
		if blk%model.RowsPerBlockInfoTable() == 0 {
			err := model.CreateBlockInfoTableIfNotExists(db, model.BlockInfoTableNameByBlockHeight(blk))
			if err != nil {
				return err
			}
		}
	}

	// start db tx
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	// get & store blocks
	for blk := startBlk; blk < endBlk; blk++ {
		var bInfo model.BlockInfo
		err := db.Table(model.BlockInfoTableNameByBlockHeight(blk)).Where("Fblock_height = ?", blk).First(&bInfo).Error
		if err == nil { // this block info was already in database, just update Fis_archived to 1
			if !bInfo.IsArchived {
				bInfo.IsArchived = true
				tx.Table(model.BlockInfoTableNameByBlockHeight(blk)).Save(&bInfo)
			}
		} else if err == gorm.ErrRecordNotFound {
			blkWithRWSet, err := cc.GetFullBlockByHeight(blk)
			if err != nil {
				return err
			}

			blkWithRWSetBytes, err := blkWithRWSet.Marshal()
			if err != nil {
				return err
			}

			sum, err := hmac(chainId, blkWithRWSet.Block.Header.BlockHeight, blkWithRWSetBytes, secretKey)
			if err != nil {
				return err
			}

			err = model.InsertBlockInfo(tx, chainId, blkWithRWSet.Block.Header.BlockHeight, blkWithRWSetBytes, sum)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// archive blocks on-chain
	err := archiveBlockOnChain(cc, endBlk-1)
	if err != nil {
		return err
	}

	// update archived block height off-chain
	err = model.UpdateArchivedBlockHeight(tx, endBlk-1)
	if err != nil {
		return err
	}

	return tx.Commit().Error
}

// archiveBlockOnChain Build & Sign & Send a ArchiveBlockRequest
func archiveBlockOnChain(cc *sdk.ChainClient, height uint64) error {
	var (
		err                error
		payload            *common.Payload
		signedPayloadBytes *common.Payload
		resp               *common.TxResponse
	)

	payload, err = cc.CreateArchiveBlockPayload(height)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignArchivePayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendArchiveBlockRequest(signedPayloadBytes, archiveBlockRequestTimeout)
	if err != nil {
		return err
	}

	return util.CheckProposalRequestResp(resp, false)
}

func calcTargetHeightByTime(t time.Time) (uint64, error) {
	targetTs := t.Unix()
	cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
	if err != nil {
		return 0, err
	}
	defer cc.Stop()

	lastBlock, err := cc.GetLastBlock(false)
	if err != nil {
		return 0, err
	}
	if lastBlock.Block.Header.BlockTimestamp <= targetTs {
		return lastBlock.Block.Header.BlockHeight, nil
	}

	genesisHeader, err := cc.GetBlockHeaderByHeight(0)
	if err != nil {
		return 0, err
	}
	if genesisHeader.BlockTimestamp >= targetTs {
		return 0, fmt.Errorf("no blocks at %s", t)
	}

	targetBlkHeight, err := util.SearchU64(lastBlock.Block.Header.BlockHeight, func(i uint64) (bool, error) {
		header, err := cc.GetBlockHeaderByHeight(i)
		if err != nil {
			return false, err
		}
		return header.BlockTimestamp > targetTs, nil
	})
	if err != nil {
		return 0, err
	}

	targetHeader, err := cc.GetBlockHeaderByHeight(targetBlkHeight)
	if err != nil {
		return 0, err
	}
	if targetHeader.BlockTimestamp > targetTs {
		targetBlkHeight--
	}

	return targetBlkHeight, nil
}
