// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"errors"
	"fmt"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	// Send Restore Block Request timeout
	restoreBlockRequestTimeout = 20 // 20s
)

func newRestoreCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "restore blockchain data",
		Long:  "restore blockchain data from off-chain storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbType != defaultDbType {
				return fmt.Errorf("unsupport database type %s", dbType)
			}

			return runRestoreCMD()
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId, flagDbType, flagDbDest, flagSecretKey, flagStartBlockHeight,
	})
	return cmd
}

// runRestoreCMD `restore` command implementation
func runRestoreCMD() error {
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

	err = validateRestore(archivedBlkHeightOnChain, restoreStartBlockHeight)
	if err != nil {
		return err
	}

	//// 4.Restore Blocks
	if archivedBlkHeightOnChain == 0 {
		return nil
	}
	var barCount = archivedBlkHeightOnChain - restoreStartBlockHeight + 1
	progress := uiprogress.New()
	bar := progress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Restoring Blocks (%d/%d)", b.Current(), barCount)
	})
	progress.Start()
	defer progress.Stop()
	for height := int64(archivedBlkHeightOnChain); height >= int64(restoreStartBlockHeight); height-- {
		if err := restoreBlock(cc, db, uint64(height)); err != nil {
			return err
		}

		bar.Incr()
	}
	return nil
}

// validateRestore basic params validation
func validateRestore(archivedBlkHeightOnChain, restoreStartBlkHeight uint64) error {
	// restore start block height is not archived
	if archivedBlkHeightOnChain < restoreStartBlkHeight {
		return errors.New("restore start block height is not archived")
	}
	return nil
}

func restoreBlock(cc *sdk.ChainClient, db *gorm.DB, height uint64) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	var bInfo model.BlockInfo
	err := tx.Table(model.BlockInfoTableNameByBlockHeight(height)).Where("Fblock_height = ?", height).First(&bInfo).Error
	if err != nil {
		return err
	}

	// verify hmac
	sum, err := hmac(chainId, height, bInfo.BlockWithRWSet, secretKey)
	if err != nil {
		return err
	}
	if sum != bInfo.Hmac {
		return fmt.Errorf("invalid HMAC signature, recalculate: %s from_db: %s", sum, bInfo.Hmac)
	}

	bInfo.IsArchived = false
	err = tx.Table(model.BlockInfoTableNameByBlockHeight(height)).Save(bInfo).Error
	if err != nil {
		return err
	}

	var archivedBlkHeight uint64
	if height > 0 {
		archivedBlkHeight = height - 1
	}

	err = model.UpdateArchivedBlockHeight(tx, archivedBlkHeight)
	if err != nil {
		return err
	}

	// only restore Not-Config-Block
	var blkWithRWSetOffChain store.BlockWithRWSet
	err = blkWithRWSetOffChain.Unmarshal(bInfo.BlockWithRWSet)
	if err != nil {
		return err
	}
	if !util.IsConfBlock(blkWithRWSetOffChain.Block) {
		err = restoreBlockOnChain(cc, bInfo.BlockWithRWSet)
		if err != nil {
			return err
		}
	}

	return tx.Commit().Error
}

func restoreBlockOnChain(cc *sdk.ChainClient, fullBlock []byte) error {
	var (
		err                error
		payload            *common.Payload
		signedPayloadBytes *common.Payload
		resp               *common.TxResponse
	)

	payload, err = cc.CreateRestoreBlockPayload(fullBlock)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignArchivePayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendRestoreBlockRequest(signedPayloadBytes, restoreBlockRequestTimeout)
	if err != nil {
		return err
	}

	return util.CheckProposalRequestResp(resp, false)
}
