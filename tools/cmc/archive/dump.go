package archive

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
	sdk "chainmaker.org/chainmaker-sdk-go"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

func dumpCMD() *cobra.Command {
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
		flagDbType, flagDbDest, flagTargetBlockHeight, flagBlockInterval,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)
	cmd.MarkFlagRequired(flagAdminCrtFilePaths)
	cmd.MarkFlagRequired(flagAdminKeyFilePaths)
	cmd.MarkFlagRequired(flagDbType)
	cmd.MarkFlagRequired(flagDbDest)
	cmd.MarkFlagRequired(flagTargetBlockHeight)
	cmd.MarkFlagRequired(flagBlockInterval)

	return cmd
}

func runDumpCMD() error {
	cc, err := createChainClient(adminKeyFilePaths, adminCrtFilePaths, chainId)
	if err != nil {
		return err
	}
	defer cc.Stop()

	dbName := model.DBName(chainId)
	dbDestSlice := strings.Split(dbDest, ":")
	if len(dbDestSlice) != 4 {
		return errors.New("invalid database destination")
	}

	// initialize database
	db, err := mysql.InitDb(dbDestSlice[0], dbDestSlice[1], dbDestSlice[2], dbDestSlice[3], dbName, true)
	if err != nil {
		return err
	}

	// lock database
	locker := mysql.NewDbLocker(db, "cmc", mysql.DefaultLockLeaseAge)
	locker.Lock()
	defer locker.UnLock()

	// migrate blockinfo,sysinfo tables
	err = db.AutoMigrate(&model.BlockInfo{}, &model.Sysinfo{})
	if err != nil {
		return err
	}

	archivedBlockHeightOffChain, err := model.GetArchivedBlockHeight(db)
	if err != nil {
		return err
	}

	// target block height already archived, do nothing.
	if targetBlockHeight <= archivedBlockHeightOffChain {
		printDone()
		return nil
	}

	fmt.Println("archivedBlockHeightOffChain=", archivedBlockHeightOffChain)
	heightOnChain, err := cc.GetArchivedBlockHeight()
	if err != nil {
		return err
	}
	archivedBlockHeightOnChain := uint64(heightOnChain)
	fmt.Println("archivedBlockHeightOnChain=", archivedBlockHeightOnChain)

	// required archived block height off-chain == archived block height on-chain
	if archivedBlockHeightOffChain != archivedBlockHeightOnChain {
		return errors.New("required archived block height off-chain == archived block height on-chain")
	}

	// required current block height > target block height
	currentBlockHeight, err := cc.GetCurrentBlockHeight()
	if err != nil {
		return err
	}
	if uint64(currentBlockHeight) <= targetBlockHeight {
		fmt.Println("lalalalalalalalala")
		fmt.Println("currentBlockHeight=", currentBlockHeight, "targetBlockHeight=", targetBlockHeight)
		printDone()
		return nil
	}

	// archive block one by one, incremental
	var archivedBlockNum uint64
	fmt.Printf("targetBlockHeight=%d archivedBlockHeightOnChain=%d blockInterval=%d archivedBlockNum=%d \n\n", targetBlockHeight, archivedBlockHeightOnChain, blockInterval, archivedBlockNum)
	for targetBlockHeight-archivedBlockHeightOnChain >= 0 && archivedBlockNum <= blockInterval {
		archivedBlockHeightOnChain++
		fmt.Printf("archivedBlockHeightOnChain %d \n\n", archivedBlockHeightOnChain)
		// archive block
		blkWithRWSet, err := cc.GetFullBlockByHeight(int64(archivedBlockHeightOnChain))
		if err != nil {
			return err
		}

		blkWithRWSetBytes, err := blkWithRWSet.Marshal()
		if err != nil {
			return err
		}

		blkHeight := blkWithRWSet.Block.Header.BlockHeight
		blkHeightBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(blkHeightBytes, uint64(blkHeight))

		sum, err := Hmac([]byte(chainId), blkHeightBytes, blkWithRWSetBytes, []byte("123"))
		if err != nil {
			return err
		}

		_, err = model.InsertBlockInfo(db, chainId, blkWithRWSet.Block.Header.BlockHeight, blkWithRWSetBytes, sum)
		if err != nil {
			return err
		}
		fmt.Printf("model.InsertBlockInfo %s \n\n", sum)

		archivedBlockNum++
	}

	//err = archiveBlockOnChain(cc, int64(archivedBlockHeightOnChain))
	//if err != nil {
	//	return err
	//}

	// cli.GetArchivedBlockHeight
	// cli.GetCurrentBlockHeight
	// cli.GetBlockWithRWSet
	//

	return nil
}

func printDone() {
	fmt.Printf("\nDone!\n")
}

func archiveBlockOnChain(cc *sdk.ChainClient, blockNum int64) error {
	var (
		err                error
		payload            []byte
		signedPayloadBytes []byte
		resp               *common.TxResponse
		result             string
	)

	payload, err = cc.CreateArchiveBlockPayload(blockNum)
	if err != nil {
		return err
	}

	signedPayloadBytes, err = cc.SignArchivePayload(payload)
	if err != nil {
		return err
	}

	resp, err = cc.SendArchiveBlockRequest(signedPayloadBytes, -1, true)
	if err != nil {
		return err
	}

	result = string(resp.ContractResult.Result)

	fmt.Printf("resp: %+v, result:%+s\n", resp, result)
	return nil
}
