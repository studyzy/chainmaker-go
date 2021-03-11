/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	configPb "chainmaker.org/chainmaker-go/pb/protogo/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateDebugConfigCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updateDebugConfig",
		Short: "Update debug config",
		Long:  "Update debug config",
		RunE: func(_ *cobra.Command, _ []string) error {
			return updateDebugConfig()
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&pairsString, "pairs", "a", "[{\"key\":\"IsHttpOpen\",\"value\":\"true\"}]", "specify pairs")

	return cmd
}

func updateDebugConfig() error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	err := json.Unmarshal([]byte(pairsString), &pairs)
	if err != nil {
		return err
	}

	request := &configPb.DebugConfigRequest{Pairs: pairs}

	r, err := client.UpdateDebugConfig(ctx, request)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return fmt.Errorf("client.call err: deadline\n")
		}
		return fmt.Errorf("client.call err: %v\n", err)
	}

	result := &Result{
		Code:    commonPb.TxStatusCode(r.Code),
		Message: r.Message,
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))

	return nil
}
