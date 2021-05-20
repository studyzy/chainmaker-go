package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newQueryArchivedHeightOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archived-height",
		Short: "query archived height",
		Long:  "query archived height",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryArchivedHeightOnChainCMD()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagChainId,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)
	cmd.MarkFlagRequired(flagChainId)

	return cmd
}

// runQueryArchivedHeightOnChainCMD `query archived height` command implementation
func runQueryArchivedHeightOnChainCMD() error {
	//// 1.Chain Client
	cc, err := createChainClient(adminKeyFilePaths, adminCrtFilePaths, chainId)
	if err != nil {
		return err
	}
	defer cc.Stop()

	//// 2.Query archived height
	archivedBlkHeight, err := cc.GetArchivedBlockHeight()
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(map[string]int64{"archived_height": archivedBlkHeight}, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
