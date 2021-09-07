/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"chainmaker.org/chainmaker/utils/v2"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker-go/accesscontrol"

	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

const GRPCMaxCallRecvMsgSize = 16 * 1024 * 1024

func constructQueryPayload(chainId, contractName, method string, pairs []*commonPb.KeyValuePair) (*commonPb.Payload, error) {
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
		TxId:         "", //Query不需要TxId
		TxType:       commonPb.TxType_QUERY_CONTRACT,
		ChainId:      chainId,
	}

	return payload, nil
}
func constructInvokePayload(chainId, contractName, method string, pairs []*commonPb.KeyValuePair) (*commonPb.Payload, error) {
	payload := &commonPb.Payload{
		ContractName:   contractName,
		Method:         method,
		Parameters:     pairs,
		TxId:           utils.GetRandTxId(),
		TxType:         commonPb.TxType_INVOKE_CONTRACT,
		ChainId:        chainId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}

	return payload, nil
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.Member) (protocol.SigningMember, error) {
	skPEM, err := sk3.String()
	if err != nil {
		return nil, err
	}
	//fmt.Printf("skPEM: %s\n", skPEM)

	signer, err := accesscontrol.NewCertSigningMember(hashAlgo, sender, skPEM, "")
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func initGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
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
		return grpc.Dial(url, grpc.WithTransportCredentials(*c),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(GRPCMaxCallRecvMsgSize)))
	} else {
		return grpc.Dial(url, grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(GRPCMaxCallRecvMsgSize)))
	}
}

func acSign(msg *commonPb.Payload) ([]*commonPb.EndorsementEntry, error) {
	//msg.Endorsement = nil
	bytes, _ := proto.Marshal(msg)

	signers := make([]protocol.SigningMember, 0)
	orgIdArray := strings.Split(orgIds, ",")
	adminSignKeyArray := strings.Split(adminSignKeys, ",")
	adminSignCrtArray := strings.Split(adminSignCrts, ",")

	if len(adminSignKeyArray) != len(adminSignCrtArray) {
		return nil, errors.New("admin key len is not equal to crt len")
	}
	if len(adminSignKeyArray) != len(orgIdArray) {
		return nil, errors.New("admin key len is not equal to orgId len")
	}

	for i, key := range adminSignKeyArray {

		file, err := ioutil.ReadFile(key)
		if err != nil {
			return nil, err
		}
		sk, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			return nil, err
		}

		file2, err := ioutil.ReadFile(adminSignCrtArray[i])
		if err != nil {
			return nil, err
		}

		// 构造Sender
		sender1 := &acPb.Member{
			OrgId:      orgIdArray[i],
			MemberInfo: file2,
			//IsFullCert: true,
		}

		signer, err := getSigner(sk, sender1)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}
	endorsements, err := accesscontrol.MockSignWithMultipleNodes(bytes, signers, hashAlgo)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("endorsements:\n%v\n", endorsements)
	return endorsements, nil
}
