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

	"github.com/spf13/cobra"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
)

func SubscribeEventCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "Subscribe Event",
		Long:  "Subscribe Event",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c, err := subscribeContractEvent(ctx, topic, contractName)
			if err != nil {
				return err
			}

			for {
				select {
				case event, ok := <-c:
					if !ok {
						return errors.New("chan is close")
					}
					if event == nil {
						return errors.New("received block is nil")
					}
					contractEventInfo, ok := event.(*common.ContractEventInfo)
					if !ok {
						return errors.New("received data is not *common.ContractEventInfo")
					}
					fmt.Printf("recv contract event [%d] => %+v\n", contractEventInfo.BlockHeight, contractEventInfo)

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

func subscribeContractEvent(ctx context.Context, topic string, contractName string) (<-chan interface{}, error) {

	payload := createPayload(chainId, "", common.TxType_SUBSCRIBE, syscontract.SystemContract_SUBSCRIBE_MANAGE.String(),
		syscontract.SubscribeFunction_SUBSCRIBE_CONTRACT_EVENT.String(), []*common.KeyValuePair{
			{
				Key:   syscontract.SubscribeContractEvent_TOPIC.String(),
				Value: []byte(topic),
			},
			{
				Key:   syscontract.SubscribeContractEvent_CONTRACT_NAME.String(),
				Value: []byte(contractName),
			},
		}, 0,
	)

	return subscribe(ctx, payload)
}
