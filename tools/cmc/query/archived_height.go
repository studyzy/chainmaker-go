package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/query/model"
)

func newArchivedHeightCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archived-height",
		Short: "query archived height",
		Long:  "query archived height",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryArchivedHeightCMD()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagChainId, flagDbType, flagDbDest,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)
	cmd.MarkFlagRequired(flagDbType)
	cmd.MarkFlagRequired(flagDbDest)

	return cmd
}

// runQueryArchivedHeightCMD `query archived height` command implementation
func runQueryArchivedHeightCMD() error {
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

	//// 3.Query tx on-chain, if it's archived on-chain then query off-chain storage.
	archivedBlkHeightOnChain, err := cc.GetArchivedBlockHeight()
	if err != nil {
		return err
	}
	archivedBlkHeightOffChain, err := model.GetArchivedBlockHeight(db)
	if err != nil {
		return err
	}

	if archivedBlkHeightOnChain != archivedBlkHeightOffChain {
		return errors.New("required archived block height off-chain == archived block height on-chain")
	}

	output, err := json.MarshalIndent(map[string]int64{"archived_height": archivedBlkHeightOnChain}, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
