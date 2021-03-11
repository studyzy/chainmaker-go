/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"fmt"

	"chainmaker.org/chainmaker-go/common/ca"
	"google.golang.org/grpc"
)

func InitGRPCConnect(ip string, port int, useTLS bool, caPaths []string, userCrtPath, userKeyPath string) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", ip, port)

	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths,
			CertFile:   userCrtPath,
			KeyFile:    userKeyPath,
		}

		c, err := tlsClient.GetCredentialsByCA()
		if err != nil {
			return nil, err
		}
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}
