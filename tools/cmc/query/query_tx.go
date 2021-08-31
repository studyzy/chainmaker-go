// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package query

import (
	"fmt"

	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
)

// newQueryTxOnChainCMD `query tx` command implementation
func newQueryTxOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [txid]",
		Short: "query on-chain tx by txid",
		Long:  "query on-chain tx by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Chain Client
			cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
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

			output, err = prettyjson.Marshal(txInfo)
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
		},
	}

	util.AttachAndRequiredFlags(cmd, flags, []string{
		flagSdkConfPath, flagChainId,
	})
	return cmd
}
