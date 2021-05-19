package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/query/db/mysql"
)

func newTxCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [txid]",
		Short: "query tx by txid",
		Long:  "query tx by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryTxCMD(args)
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

// runQueryTxCMD `query tx` command implementation
func runQueryTxCMD(args []string) error {
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
	locker := mysql.NewDbLocker(db, "cmc", mysql.DefaultLockLeaseAge)
	locker.Lock()
	defer locker.UnLock()

	//// 3.Query tx on-chain
	txInfo, err := cc.GetTxByTxId(args[0])
	if err != nil {
		return err
	}

	bz, err := json.MarshalIndent(txInfo, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(bz))
	return nil
}
