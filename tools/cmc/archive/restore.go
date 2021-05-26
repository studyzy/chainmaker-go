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
	sdk "chainmaker.org/chainmaker-sdk-go"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

func newRestoreCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "restore blockchain data",
		Long:  "restore blockchain data from off-chain storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestoreCMD()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagChainId, flagDbType, flagDbDest, flagSecretKey, flagStartBlockHeight,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)
	cmd.MarkFlagRequired(flagDbType)
	cmd.MarkFlagRequired(flagDbDest)
	cmd.MarkFlagRequired(flagSecretKey)
	cmd.MarkFlagRequired(flagStartBlockHeight)

	return cmd
}

// runRestoreCMD `restore` command implementation
func runRestoreCMD() error {
	//// 1.Chain Client
	cc, err := util.CreateChainClientWithSDKConf(sdkConfPath)
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
	var barCount = archivedBlkHeightOnChain - restoreStartBlockHeight + 1
	bar := uiprogress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("\nRestoring Blocks (%d/%d)", b.Current(), barCount)
	})
	uiprogress.Start()
	for height := archivedBlkHeightOnChain; height >= restoreStartBlockHeight; height-- {
		if err := restoreBlock(cc, db, height); err != nil {
			return err
		}

		bar.Incr()
	}
	uiprogress.Stop()
	return nil
}

// validateRestore basic params validation
func validateRestore(archivedBlkHeightOnChain, restoreStartBlkHeight int64) error {
	if restoreStartBlkHeight < 0 {
		return errors.New("restore start block height must >= 0")
	}
	// restore start block height is not archived
	if archivedBlkHeightOnChain < restoreStartBlkHeight {
		return errors.New("restore start block height is not archived")
	}
	return nil
}

func restoreBlock(cc *sdk.ChainClient, db *gorm.DB, height int64) error {
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

	err = restoreBlockOnChain(cc, bInfo.BlockWithRWSet)
	if err != nil {
		return err
	}

	bInfo.IsArchived = false
	err = tx.Table(model.BlockInfoTableNameByBlockHeight(height)).Save(bInfo).Error
	if err != nil {
		return err
	}

	var archivedBlkHeight int64
	if height > 0 {
		archivedBlkHeight = height - 1
	}

	err = model.UpdateArchivedBlockHeight(tx, archivedBlkHeight)
	if err != nil {
		return err
	}

	return tx.Commit().Error
}

func restoreBlockOnChain(cc *sdk.ChainClient, fullBlock []byte) error {
	var (
		err                error
		payload            []byte
		signedPayloadBytes []byte
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

	resp, err = cc.SendRestoreBlockRequest(signedPayloadBytes, -1)
	if err != nil {
		return err
	}

	return util.CheckProposalRequestResp(resp, false)
}
