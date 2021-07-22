// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package query

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/hokaccha/go-prettyjson"
	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/types"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
)

// newQueryBlockByHeightOnChainCMD `query block by block height` command implementation
func newQueryBlockByHeightOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-height [height]",
		Short: "query on-chain block by height",
		Long:  "query on-chain block by height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			height, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			//// 1.Chain Client
			cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
			if err != nil {
				return err
			}
			defer cc.Stop()

			//// 2.Query block on-chain.
			blkWithRWSetOnChain, err := cc.GetFullBlockByHeight(height)
			if err != nil {
				return err
			}

			var blkWithRWSet = &types.BlockWithRWSet{
				BlockWithRWSet: blkWithRWSetOnChain,
				Block: &types.Block{
					Block: blkWithRWSetOnChain.Block,
					Header: &types.BlockHeader{
						BlockHeader: blkWithRWSetOnChain.Block.Header,
						BlockHash:   hex.EncodeToString(blkWithRWSetOnChain.Block.Header.BlockHash),
					},
				},
			}

			output, err := prettyjson.Marshal(blkWithRWSet)
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

// newQueryBlockByHashOnChainCMD `query block by block hash` command implementation
func newQueryBlockByHashOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-hash [block hash in hex]",
		Short: "query on-chain block by hash",
		Long:  "query on-chain block by hash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Chain Client
			cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
			if err != nil {
				return err
			}
			defer cc.Stop()

			//// 2.Query block on-chain.
			height, err := cc.GetBlockHeightByHash(args[0])
			if err != nil {
				return err
			}
			blkWithRWSetOnChain, err := cc.GetFullBlockByHeight(height)
			if err != nil {
				return err
			}

			var blkWithRWSet = &types.BlockWithRWSet{
				BlockWithRWSet: blkWithRWSetOnChain,
				Block: &types.Block{
					Block: blkWithRWSetOnChain.Block,
					Header: &types.BlockHeader{
						BlockHeader: blkWithRWSetOnChain.Block.Header,
						BlockHash:   hex.EncodeToString(blkWithRWSetOnChain.Block.Header.BlockHash),
					},
				},
			}

			output, err := prettyjson.Marshal(blkWithRWSet)
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

// newQueryBlockByTxIdOnChainCMD `query block by txid` command implementation
func newQueryBlockByTxIdOnChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-txid [txid]",
		Short: "query on-chain block by txid",
		Long:  "query on-chain block by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Chain Client
			cc, err := util.CreateChainClient(sdkConfPath, chainId, "", "", "", "", "")
			if err != nil {
				return err
			}
			defer cc.Stop()

			//// 2.Query block on-chain.
			height, err := cc.GetBlockHeightByTxId(args[0])
			if err != nil {
				return err
			}
			blkWithRWSetOnChain, err := cc.GetFullBlockByHeight(height)
			if err != nil {
				return err
			}

			var blkWithRWSet = &types.BlockWithRWSet{
				BlockWithRWSet: blkWithRWSetOnChain,
				Block: &types.Block{
					Block: blkWithRWSetOnChain.Block,
					Header: &types.BlockHeader{
						BlockHeader: blkWithRWSetOnChain.Block.Header,
						BlockHash:   hex.EncodeToString(blkWithRWSetOnChain.Block.Header.BlockHash),
					},
				},
			}

			output, err := prettyjson.Marshal(blkWithRWSet)
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
