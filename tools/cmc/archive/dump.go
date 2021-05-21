package archive

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
	sdk "chainmaker.org/chainmaker-sdk-go"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/store"
)

const (
	// default 20 blocks per batch
	blocksPerBatch = 20
)

func newDumpCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "dump blockchain data",
		Long:  "dump blockchain data to off-chain storage and delete on-chain data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDumpCMD()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagChainId, flagAdminCrtFilePaths, flagAdminKeyFilePaths,
		flagDbType, flagDbDest, flagTargetBlockHeight, flagBlocks,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagDbType)
	cmd.MarkFlagRequired(flagDbDest)
	cmd.MarkFlagRequired(flagTargetBlockHeight)
	cmd.MarkFlagRequired(flagBlocks)

	return cmd
}

// runDumpCMD `dump` command implementation
func runDumpCMD() error {
	//// 1.Chain Client
	cc, err := createChainClient(adminKeyFilePaths, adminCrtFilePaths, chainId)
	if err != nil {
		return err
	}
	defer cc.Stop()

	//// 2.Database
	db, err := initDb()
	if err != nil {
		return err
	}
	//locker := mysql.NewDbLocker(db, "cmc", mysql.DefaultLockLeaseAge)
	//locker.Lock()
	//defer locker.UnLock()

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

	err = validate(archivedBlkHeightOnChain, archivedBlkHeightOffChain, currentBlkHeightOnChain)
	if err != nil {
		return err
	}

	//// 4.Store & Archive Blocks
	var barCount = targetBlkHeight - archivedBlkHeightOnChain
	if blocks < barCount {
		barCount = blocks
	}
	bar := uiprogress.AddBar(int(barCount)).AppendCompleted().PrependElapsed()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Archiving Block (%d/%d)", b.Current(), barCount)
	})
	uiprogress.Start()
	var batchStartBlkHeight, batchEndBlkHeight = archivedBlkHeightOnChain + 1, archivedBlkHeightOnChain + 1
	for processedBlocks := int64(0); targetBlkHeight >= batchEndBlkHeight && processedBlocks <= blocks; processedBlocks++ {
		if batchEndBlkHeight-batchStartBlkHeight >= blocksPerBatch {
			if err := runBatch(cc, db, batchStartBlkHeight, batchEndBlkHeight); err != nil {
				return err
			}

			batchStartBlkHeight = batchEndBlkHeight
		}

		batchEndBlkHeight++
		bar.Incr()
	}
	uiprogress.Stop()
	// do the rest of blocks
	return runBatch(cc, db, batchStartBlkHeight, batchEndBlkHeight)
}

// validate basic params validation
func validate(archivedBlkHeightOnChain, archivedBlkHeightOffChain, currentBlkHeightOnChain int64) error {
	// target block height already archived, do nothing.
	if targetBlkHeight <= archivedBlkHeightOffChain {
		return errors.New("target block height already archived")
	}

	// required archived block height off-chain == archived block height on-chain
	if archivedBlkHeightOffChain != archivedBlkHeightOnChain {
		return errors.New("required archived block height off-chain == archived block height on-chain")
	}

	// required current block height > target block height
	if currentBlkHeightOnChain <= targetBlkHeight {
		return errors.New("required current block height > target block height")
	}
	return nil
}

// batchGetFullBlocks Get full blocks start from startBlk end at endBlk.
// NOTE: Include startBlk, exclude endBlk
func batchGetFullBlocks(cc *sdk.ChainClient, startBlk, endBlk int64) ([]*store.BlockWithRWSet, error) {
	var blkWithRWSetSlice []*store.BlockWithRWSet
	for blk := startBlk; blk < endBlk; blk++ {
		blkWithRWSet, err := cc.GetFullBlockByHeight(blk)
		if err != nil {
			return nil, err
		}
		blkWithRWSetSlice = append(blkWithRWSetSlice, blkWithRWSet)
	}
	return blkWithRWSetSlice, nil
}

// batchStoreAndArchiveBlocks Store blocks to off-chain storage then archive blocks on-chain
func batchStoreAndArchiveBlocks(cc *sdk.ChainClient, db *gorm.DB, blkWithRWSetSlice []*store.BlockWithRWSet) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	// store blocks
	for _, blkWithRWSet := range blkWithRWSetSlice {
		blkWithRWSetBytes, err := blkWithRWSet.Marshal()
		if err != nil {
			return err
		}

		blkHeightBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(blkHeightBytes, uint64(blkWithRWSet.Block.Header.BlockHeight))

		sum, err := Hmac([]byte(chainId), blkHeightBytes, blkWithRWSetBytes, []byte(secretKey))
		if err != nil {
			return err
		}

		_, err = model.InsertBlockInfo(tx, chainId, blkWithRWSet.Block.Header.BlockHeight, blkWithRWSetBytes, sum)
		if err != nil {
			return err
		}
	}

	// archive blocks on-chain
	archivedBlkHeightOnChain := blkWithRWSetSlice[len(blkWithRWSetSlice)-1].Block.Header.BlockHeight
	err := archiveBlockOnChain(cc, archivedBlkHeightOnChain)
	if err != nil {
		return err
	}

	// update archived block height off-chain
	err = model.UpdateArchivedBlockHeight(tx, archivedBlkHeightOnChain)
	if err != nil {
		return err
	}

	return tx.Commit().Error
}

// runBatch Run a batch job
func runBatch(cc *sdk.ChainClient, db *gorm.DB, startBlk, endBlk int64) error {
	blkWithRWSetSlice, err := batchGetFullBlocks(cc, startBlk, endBlk)
	if err != nil {
		return err
	}

	return batchStoreAndArchiveBlocks(cc, db, blkWithRWSetSlice)
}

// archiveBlockOnChain Build & Sign & Send a ArchiveBlockRequest
func archiveBlockOnChain(cc *sdk.ChainClient, height int64) error {
	var (
		err                error
		payload            []byte
		signedPayloadBytes []byte
	)

	payload, err = cc.CreateArchiveBlockPayload(height)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignArchivePayload(payload)
	if err != nil {
		return err
	}

	_, err = cc.SendArchiveBlockRequest(signedPayloadBytes, -1, false)
	if err != nil {
		return err
	}

	return nil
}
