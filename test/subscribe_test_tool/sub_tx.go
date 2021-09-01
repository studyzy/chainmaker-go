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
	"strings"

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/sdk-go/v2/utils"
)

func SubscribeTxCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Subscribe Tx",
		Long:  "Subscribe Tx",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			txIdsSlice := strings.Split(txIds, ",")

			c, err := subscribeTx(ctx, startBlock, endBlock, contractName, txIdsSlice)
			if err != nil {
				return err
			}

			for {
				select {
				case txI, ok := <-c:
					if !ok {
						return errors.New("chan is close")
					}

					if txI == nil {
						return errors.New("received tx is nil")
					}

					tx, ok := txI.(*common.Transaction)
					if !ok {
						return errors.New("received data is not *common.Transaction")
					}

					fmt.Printf("recv tx [%s] => %+v\n", tx.Payload.TxId, tx)

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

func subscribeTx(ctx context.Context, startBlock, endBlock int64, contractName string, txIds []string) (<-chan interface{}, error) {

	payload := createPayload(chainId, "", common.TxType_SUBSCRIBE, syscontract.SystemContract_SUBSCRIBE_MANAGE.String(),
		syscontract.SubscribeFunction_SUBSCRIBE_TX.String(), []*common.KeyValuePair{
			{
				Key:   syscontract.SubscribeTx_START_BLOCK.String(),
				Value: utils.I64ToBytes(startBlock),
			},
			{
				Key:   syscontract.SubscribeTx_END_BLOCK.String(),
				Value: utils.I64ToBytes(endBlock),
			},
			{
				Key:   syscontract.SubscribeTx_CONTRACT_NAME.String(),
				Value: []byte(contractName),
			},
			{
				Key:   syscontract.SubscribeTx_TX_IDS.String(),
				Value: []byte(strings.Join(txIds, ",")),
			},
		}, 0,
	)

	return subscribe(ctx, payload)
}
