/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/sdk-go/v2/utils"
)

func SubscribeBlockCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Subscribe Block",
		Long:  "Subscribe Block",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c, err := subscribeBlock(ctx, startBlock, endBlock, withRwSet, onlyHeader)
			if err != nil {
				return err
			}

			for {
				select {
				case block, ok := <-c:
					if !ok {
						return errors.New("chan is close")
					}

					if block == nil {
						return errors.New("received block is nil")
					}

					if onlyHeader {
						blockHeader, ok := block.(*common.BlockHeader)
						if !ok {
							return errors.New("received data is not *common.BlockHeader")
						}

						fmt.Printf("recv blockHeader [%d] => %+v\n", blockHeader.BlockHeight, blockHeader)
					} else {
						blockInfo, ok := block.(*common.BlockInfo)
						if !ok {
							return errors.New("received data is not *common.BlockInfo")
						}

						fmt.Printf("recv blockInfo [%d] => %+v\n", blockInfo.Block.Header.BlockHeight, blockInfo)
					}

					//if err := client.Stop(); err != nil {
					//	return
					//}
					//return
				case <-ctx.Done():
					return nil
				}
			}
		},
	}

	return cmd
}

func subscribeBlock(ctx context.Context, startBlock, endBlock int64, withRWSet,
	onlyHeader bool) (<-chan interface{}, error) {

	payload := createPayload(chainId, "", common.TxType_SUBSCRIBE, syscontract.SystemContract_SUBSCRIBE_MANAGE.String(),
		syscontract.SubscribeFunction_SUBSCRIBE_BLOCK.String(), []*common.KeyValuePair{
			{
				Key:   syscontract.SubscribeBlock_START_BLOCK.String(),
				Value: utils.I64ToBytes(startBlock),
			},
			{
				Key:   syscontract.SubscribeBlock_END_BLOCK.String(),
				Value: utils.I64ToBytes(endBlock),
			},
			{
				Key:   syscontract.SubscribeBlock_WITH_RWSET.String(),
				Value: []byte(strconv.FormatBool(withRWSet)),
			},
			{
				Key:   syscontract.SubscribeBlock_ONLY_HEADER.String(),
				Value: []byte(strconv.FormatBool(onlyHeader)),
			},
		}, 0,
	)

	return subscribe(ctx, payload)
}
