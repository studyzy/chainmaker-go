package query

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/query/model"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/store"
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
	//locker := mysql.NewDbLocker(db, "cmc", mysql.DefaultLockLeaseAge)
	//locker.Lock()
	//defer locker.UnLock()

	//// 3.Query tx on-chain
	var txInfo *common.TransactionInfo
	var output []byte
	txInfo, err = cc.GetTxByTxId(args[0])
	if err != nil {
		if strings.Contains(err.Error(), "archived transaction") {
			blkHeight, err := cc.GetBlockHeightByTxId(args[0])
			if err != nil {
				return err
			}
			var bInfo model.BlockInfo
			err = db.Table(model.BlockInfoTableNameByBlockHeight(blkHeight)).Where(&model.BlockInfo{BlockHeight: blkHeight}).Find(&bInfo).Error
			if err != nil {
				return err
			}

			var blkWithRWSet store.BlockWithRWSet
			err = blkWithRWSet.Unmarshal(bInfo.BlockWithRWSet)
			if err != nil {
				return err
			}

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
		} else {
			return err
		}
	}

	output, err = json.MarshalIndent(txInfo, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
