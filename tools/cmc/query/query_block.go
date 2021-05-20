package query

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"strconv"
)

func newQueryBlockOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block [height]",
		Short: "query on-chain block by height",
		Long:  "query on-chain block by height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryBlockOnChainCMD(args)
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagChainId,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)

	return cmd
}

// runQueryBlockOnChainCMD `query block` command implementation
func runQueryBlockOnChainCMD(args []string) error {
	height, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}
	//// 1.Chain Client
	cc, err := createChainClient(adminKeyFilePaths, adminCrtFilePaths, chainId)
	if err != nil {
		return err
	}
	defer cc.Stop()

	//// 2.Query block on-chain.
	blkWithRWSetOnChain, err := cc.GetFullBlockByHeight(height)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(blkWithRWSetOnChain, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
