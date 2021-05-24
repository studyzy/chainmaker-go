// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"chainmaker.org/chainmaker-go/tools/cmc/archive/model"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/store"
)

func newQueryOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "query off-chain blockchain data",
		Long:  "query off-chain blockchain data",
	}

	cmd.AddCommand(newQueryTxOffChainCMD())
	cmd.AddCommand(newQueryBlockByHeightOffChainCMD())
	cmd.AddCommand(newQueryBlockByHashOffChainCMD())
	cmd.AddCommand(newQueryBlockByTxIdOffChainCMD())
	cmd.AddCommand(newQueryArchivedHeightOffChainCMD())

	return cmd
}

func newQueryTxOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx [txid]",
		Short: "query off-chain tx by txid",
		Long:  "query off-chain tx by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			//// 3.Query tx off-chain.
			var txInfo *common.TransactionInfo
			var output []byte
			blkHeight, err := cc.GetBlockHeightByTxId(args[0])
			if err != nil {
				return err
			}

			var bInfo model.BlockInfo
			err = db.Table(model.BlockInfoTableNameByBlockHeight(blkHeight)).Where("Fblock_height = ?", blkHeight).Find(&bInfo).Error
			if err != nil {
				return err
			}

			var blkWithRWSet store.BlockWithRWSet
			err = blkWithRWSet.Unmarshal(bInfo.BlockWithRWSet)
			if err != nil {
				return err
			}

			if blkWithRWSet.Block != nil {
				for idx, tx := range blkWithRWSet.Block.Txs {
					if tx.Header.TxId == args[0] {
						txInfo = &common.TransactionInfo{
							Transaction: tx,
							BlockHeight: uint64(blkWithRWSet.Block.Header.BlockHeight),
							BlockHash:   blkWithRWSet.Block.Header.BlockHash,
							TxIndex:     uint32(idx),
						}

						output, err = txInfo.Marshal()
						if err != nil {
							return err
						}
						break
					}
				}
			}

			if txInfo == nil {
				output, _ = json.MarshalIndent(map[string]string{"err": "tx not found in off-chain storage"}, "", "    ")
			} else {
				output, err = json.MarshalIndent(txInfo, "", "    ")
				if err != nil {
					return err
				}
			}

			fmt.Println(string(output))
			return nil
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

func newQueryBlockByHeightOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-height [height]",
		Short: "query off-chain block by height",
		Long:  "query off-chain block by height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			height, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}
			if height < 0 {
				return errors.New("block height must >= 0")
			}
			//// 1.Database
			db, err := initDb()
			if err != nil {
				return err
			}

			//// 2.Query block off-chain.
			var output []byte
			var bInfo model.BlockInfo
			err = db.Table(model.BlockInfoTableNameByBlockHeight(height)).Where("Fblock_height = ?", height).First(&bInfo).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					output, _ = json.MarshalIndent(map[string]string{"err": "block not found in off-chain storage"}, "", "    ")
				} else {
					return err
				}
			} else {
				var blkWithRWSetOffChain store.BlockWithRWSet
				err = blkWithRWSetOffChain.Unmarshal(bInfo.BlockWithRWSet)
				if err != nil {
					return err
				}
				output, err = json.MarshalIndent(blkWithRWSetOffChain, "", "    ")
				if err != nil {
					return err
				}
			}

			fmt.Println(string(output))
			return nil
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

func newQueryBlockByHashOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-hash [block hash in hex]",
		Short: "query off-chain block by hash",
		Long:  "query off-chain block by hash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			//// 3.Query block off-chain.
			height, err := cc.GetBlockHeightByHash(args[0])
			if err != nil {
				return err
			}

			var output []byte
			var bInfo model.BlockInfo
			err = db.Table(model.BlockInfoTableNameByBlockHeight(height)).Where("Fblock_height = ?", height).First(&bInfo).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					output, _ = json.MarshalIndent(map[string]string{"err": "block not found in off-chain storage"}, "", "    ")
				} else {
					return err
				}
			} else {
				var blkWithRWSetOffChain store.BlockWithRWSet
				err = blkWithRWSetOffChain.Unmarshal(bInfo.BlockWithRWSet)
				if err != nil {
					return err
				}
				output, err = json.MarshalIndent(blkWithRWSetOffChain, "", "    ")
				if err != nil {
					return err
				}
			}

			fmt.Println(string(output))
			return nil
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

func newQueryBlockByTxIdOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-by-txid [txid]",
		Short: "query off-chain block by txid",
		Long:  "query off-chain block by txid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			//// 3.Query block off-chain.
			height, err := cc.GetBlockHeightByTxId(args[0])
			if err != nil {
				return err
			}

			var output []byte
			var bInfo model.BlockInfo
			err = db.Table(model.BlockInfoTableNameByBlockHeight(height)).Where("Fblock_height = ?", height).First(&bInfo).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					output, _ = json.MarshalIndent(map[string]string{"err": "block not found in off-chain storage"}, "", "    ")
				} else {
					return err
				}
			} else {
				var blkWithRWSetOffChain store.BlockWithRWSet
				err = blkWithRWSetOffChain.Unmarshal(bInfo.BlockWithRWSet)
				if err != nil {
					return err
				}
				output, err = json.MarshalIndent(blkWithRWSetOffChain, "", "    ")
				if err != nil {
					return err
				}
			}

			fmt.Println(string(output))
			return nil
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

func newQueryArchivedHeightOffChainCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archived-height",
		Short: "query off-chain archived height",
		Long:  "query off-chain archived height",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			//// 1.Database
			db, err := initDb()
			if err != nil {
				return err
			}

			//// 2.Query archived block height off-chain.
			archivedBlkHeightOffChain, err := model.GetArchivedBlockHeight(db)
			if err != nil {
				return err
			}

			output, err := json.MarshalIndent(map[string]int64{"archived_height": archivedBlkHeightOffChain}, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(output))
			return nil
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
