// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
)

func newQueryArchivedHeightOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archived-height",
		Short: "query on-chain archived height",
		Long:  "query on-chain archived height",
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
	cc, err := util.CreateChainClientWithSDKConf(sdkConfPath)
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
