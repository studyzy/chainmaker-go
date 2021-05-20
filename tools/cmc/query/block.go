package query

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker-go/tools/cmc/query/model"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/store"
)

func newBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block [height]",
		Short: "query block by height",
		Long:  "query block by height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryBlockCMD(args)
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

// runQueryBlockCMD `query block` command implementation
func runQueryBlockCMD(args []string) error {
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

	//// 2.Database
	db, err := initDb()
	if err != nil {
		return err
	}
	//locker := mysql.NewDbLocker(db, "cmc", mysql.DefaultLockLeaseAge)
	//locker.Lock()
	//defer locker.UnLock()

	//// 3.Query block on-chain, if it's archived on-chain then query off-chain storage.
	var blkWithRWSet *store.BlockWithRWSet
	var output []byte
	blkWithRWSetOnChain, err := cc.GetFullBlockByHeight(height)
	if err != nil {
		if strings.Contains(err.Error(), common.TxStatusCode_ARCHIVED_BLOCK.String()) {
			var bInfo model.BlockInfo
			err = db.Table(model.BlockInfoTableNameByBlockHeight(height)).Where(&model.BlockInfo{BlockHeight: height}).Find(&bInfo).Error
			if err != nil {
				return err
			}

			var blkWithRWSetOffChain store.BlockWithRWSet
			err = blkWithRWSetOffChain.Unmarshal(bInfo.BlockWithRWSet)
			if err != nil {
				return err
			}
			blkWithRWSet = &blkWithRWSetOffChain
		} else {
			return err
		}
	} else {
		blkWithRWSet = blkWithRWSetOnChain
	}

	output, err = json.MarshalIndent(blkWithRWSet, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
