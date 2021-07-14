/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker-go/accesscontrol"

	"chainmaker.org/chainmaker/common/ca"
	"chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/asym"

	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

func constructPayload(contractName, method string, pairs []*commonPb.KeyValuePair) ([]byte, error) {
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return payloadBytes, nil
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.Member) (protocol.SigningMember, error) {
	skPEM, err := sk3.String()
	if err != nil {
		return nil, err
	}
	//fmt.Printf("skPEM: %s\n", skPEM)

	m, err := accesscontrol.MockAccessControlWithHash(hashAlgo).NewMemberFromCertPem(sender.OrgId, string(sender.MemberInfo))
	if err != nil {
		return nil, err
	}

	signer, err := accesscontrol.MockAccessControl().NewSigningMember(m, skPEM, "")
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
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
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
