/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package grpc_client

import (
	"fmt"
	"log"

	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"

	"chainmaker.org/chainmaker/common/v2/ca"
	"google.golang.org/grpc"
)

type GrpcAPIClient struct {
	Client     *apiPb.RpcNodeClient
	conn       *grpc.ClientConn
	addr       string
	timeoutSec int64
	Tag        string
}

func NewGrpcAPIClient(addr string, toSec int64, tag string) *GrpcAPIClient {
	c := &GrpcAPIClient{
		addr:       addr,
		timeoutSec: toSec,
		Tag:        tag,
	}
	return c
}

func (c *GrpcAPIClient) OpenConnection() error {
	c.CloseConnection()

	opts := GenerateClientDialOption()
	// opts = append(opts, grpc.WithBlock())
	// opts = append(opts, grpc.WithTimeout(time.Duration(c.timeoutSec)*time.Second))
	opts = append(opts, grpc.WithInsecure())

	var err error
	c.conn, err = grpc.Dial(c.addr, opts...)
	// c.conn, err = grpc.Dial(c.addr, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("[GRPC] fail to connect dial addr %s: %v\n", c.addr, err)
		return err
	}
	client := apiPb.NewRpcNodeClient(c.conn)
	c.Client = &client
	return nil
}

func (c *GrpcAPIClient) OpenTLSConnection(caPaths []string, userCrtPath string, userKeyPath string) error {
	c.CloseConnection()

	opts := GenerateClientDialOption()
	// opts = append(opts, grpc.WithBlock())
	// opts = append(opts, grpc.WithTimeout(time.Duration(c.timeoutSec)*time.Second))

	var err error
	tlsClient := ca.CAClient{
		ServerName: "chainmaker.org",
		CaPaths:    caPaths,
		CertFile:   userCrtPath,
		KeyFile:    userKeyPath,
	}

	cdl, err := tlsClient.GetCredentialsByCA()
	if err != nil {
		log.Fatalf("GetTLSCredentialsByCA err: %v", err)
		return err
	}
	opts = append(opts, grpc.WithTransportCredentials(*cdl))

	c.conn, err = grpc.Dial(c.addr, opts...)
	if err != nil {
		fmt.Printf("[GRPC] fail to connect dial addr %s: %v\n", c.addr, err)
		return err
	}
	client := apiPb.NewRpcNodeClient(c.conn)
	c.Client = &client
	return nil
}

func (c *GrpcAPIClient) CloseConnection() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

//-----------------------------------

// GenerateClientDialOption wrap general grpc.DialOption
func GenerateClientDialOption() []grpc.DialOption {
	opts := []grpc.DialOption{}
	return opts
}
