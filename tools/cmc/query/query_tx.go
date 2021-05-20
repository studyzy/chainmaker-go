package query

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

func newQueryTxOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [txid]",
		Short: "query on-chain tx by txid",
		Long:  "query on-chain tx by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryTxOnChainCMD(args)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagChainId,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)

	return cmd
}

// runQueryTxOnChainCMD `query tx` command implementation
func runQueryTxOnChainCMD(args []string) error {
	//// 1.Chain Client
	cc, err := createChainClient(adminKeyFilePaths, adminCrtFilePaths, chainId)
	if err != nil {
		return err
	}
	defer cc.Stop()

	//// 2.Query tx on-chain
	var txInfo *common.TransactionInfo
	var output []byte
	txInfo, err = cc.GetTxByTxId(args[0])
	if err != nil {
		return err
	}

	output, err = json.MarshalIndent(txInfo, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
