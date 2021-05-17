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

	// check the height of the block where the off-chain storage have archived.
	archivedBlockHeight, err := model.GetArchivedBlockHeight(db)
	if err != nil {
		return err
	}
	fmt.Println("archivedBlockHeight=", archivedBlockHeight)
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
