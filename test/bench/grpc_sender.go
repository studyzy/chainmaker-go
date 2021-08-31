/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker-go/test/grpc_client"
)

type BenchmarkerSender interface {
	SendTxByClientIndex(url string, tx *commonPb.TxRequest, index int64) (*commonPb.TxResponse, error)
}

type GRPCSender struct {
	ClientCount int
	ClientList  []*grpc_client.GrpcAPIClient //cache parallel clients
	addr        string
	intervalSec int
}

func NewGRPCSender(clientCount int, addr string, usetls bool) BenchmarkerSender {
	sender := &GRPCSender{
		ClientCount: clientCount,
		addr:        addr,
		intervalSec: 10,
	}

	for i := 0; i < clientCount; i++ {
		grpcClient := grpc_client.NewGrpcAPIClient(addr, 10, fmt.Sprintf("%d", i))
		sender.ClientList = append(sender.ClientList, grpcClient)
		if usetls {
			err := grpcClient.OpenTLSConnection(caPaths, userCrtPath, userKeyPath)
			if nil != err {
				panic(err)
			}
		} else {
			err := grpcClient.OpenConnection()
			if nil != err {
				panic(err)
			}
		}

	}

	return sender
}

func (s *GRPCSender) SendTxByClientIndex(url string, txReq *commonPb.TxRequest, index int64) (*commonPb.TxResponse, error) {

	client := s.ClientList[index]
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(s.intervalSec)*time.Second)
	// ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(s.intervalSec)*time.Second))
	// defer cancel()

	res, err := (*(client.Client)).SendRequest(ctx, txReq)
	return res, err
}
