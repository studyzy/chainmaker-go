package archive

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/db/mysql"
	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
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

	// check if archived block height off-chain == archived block height on-chain
	archivedBlockHeightOffChain, err := model.GetArchivedBlockHeight(db)
	if err != nil {
		return err
	}
	fmt.Println("archivedBlockHeightOffChain=", archivedBlockHeightOffChain)
	heightOnChain, err := cc.GetArchivedBlockHeight()
	if err != nil {
		return err
	}
	archivedBlockHeightOnChain := uint64(heightOnChain)
	fmt.Println("archivedBlockHeightOnChain=", archivedBlockHeightOnChain)
	if archivedBlockHeightOffChain != archivedBlockHeightOnChain {
		return errors.New("archived block height off-chain != archived block height on-chain")
	}
	if targetBlockHeight <= archivedBlockHeightOnChain {
		// do nothing
		printDone()
		return nil
	}

	// archive block one by one
	var archivedBlockNum uint64
	for targetBlockHeight-archivedBlockHeightOnChain >= 0 && blockInterval <= archivedBlockNum {
		// archive block
		blkWithRWSet, err := cc.GetFullBlockByHeight(int64(archivedBlockHeightOnChain))
		if err != nil {
			return err
		}

		bz, err := blkWithRWSet.Marshal()
		if err != nil {
			return err
		}

		sig, err := Hmac()
		if err != nil {
			return err
		}

		_, err = model.InsertBlockInfo(db, chainId, blkWithRWSet.Block.Header.BlockHeight, bz, sig)
		if err != nil {
			return err
		}

		res, err := cc.ArchiveBlock(int64(archivedBlockHeightOnChain))
		if err != nil {
			return err
		}

		fmt.Println("cc.ArchiveBlock=", res)

		archivedBlockNum++
		archivedBlockHeightOnChain++
	}

	// cli.GetArchivedBlockHeight
	// cli.GetCurrentBlockHeight
	// cli.GetBlockWithRWSet
	//

	resp, err := cc.GetBlockByHeight(-1, true)
	if err != nil {
		return fmt.Errorf("get block by height failed, %s", err.Error())
	}

	fmt.Printf("\n\n\n\n\nget block by height resp: %+v\n", resp.Block.Header.BlockHeight)

	return nil
}

func printDone() {
	fmt.Printf("\nDone!\n")
}
