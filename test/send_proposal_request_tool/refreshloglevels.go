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

func RefreshLogLevelsCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refreshLogLevels",
		Short: "Refresh log levels",
		Long:  "Refresh Log Levels",
		RunE: func(_ *cobra.Command, _ []string) error {
			return refreshLogLevelsConfig()
		},
	}

	return cmd
}

func refreshLogLevelsConfig() error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	r, err := client.RefreshLogLevelsConfig(ctx, &configPb.LogLevelsRequest{})
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return fmt.Errorf("client.call err: deadline\n")
		}
		return fmt.Errorf("client.call err: %v\n", err)
	}

	result := &SimpleRPCResult{
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
