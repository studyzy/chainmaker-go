/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// description: chainmaker-go
//
// @author: xwc1125
// @date: 2020/11/24
package native

import (
	acPb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	apiPb "chainmaker.org/chainmaker-go/pb/protogo/api"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/common/ca"
	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

const (
	IsTls          = true
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "../../config"
	WasmPath       = "../wasm/counter-go.wasm"
	OrgIdFormat    = "wx-org%s.chainmaker.org"
	UserKeyPathFmt = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/client1/client1.tls.key"
	UserCrtPathFmt = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/client1/client1.tls.crt"

	prePathFmt = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
)

var (
	err                error
	DefaultUserKeyPath = fmt.Sprintf(UserKeyPathFmt, "1")
	DefaultUserCrtPath = fmt.Sprintf(UserCrtPathFmt, "1")
	DefaultOrgId       = fmt.Sprintf(OrgIdFormat, "1")

	caPaths = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}
)

func InitGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", IP, Port)

	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths,
			CertFile:   DefaultUserCrtPath,
			KeyFile:    DefaultUserKeyPath,
		}

		c, err := tlsClient.GetCredentialsByCA()
		if err != nil {
			log.Fatalf("GetTLSCredentialsByCA err: %v", err)
			return nil, err
		}
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.SerializedMember) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}

	m, err := accesscontrol.MockAccessControl().NewMemberFromCertPem(sender.OrgId, string(sender.MemberInfo))
	if err != nil {
		panic(err)
	}

	signer, err := accesscontrol.MockAccessControl().NewSigningMember(m, skPEM, "")
	if err != nil {
		panic(err)
	}
	return signer
}

type InvokeContractMsg struct {
	TxType       commonPb.TxType
	ChainId      string
	TxId         string
	ContractName string
	MethodName   string
	Pairs        []*commonPb.KeyValuePair
}

func QueryRequest(sk3 crypto.PrivateKey, sender *acPb.SerializedMember, client *apiPb.RpcNodeClient, msg *InvokeContractMsg) (*commonPb.TxResponse, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	if msg.TxId == "" {
		msg.TxId = utils.GetRandTxId()
	}

	// 构造Header
	header := &commonPb.TxHeader{
		ChainId:        msg.ChainId,
		Sender:         sender,
		TxType:         msg.TxType,
		TxId:           msg.TxId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}

	payload := &commonPb.QueryPayload{
		ContractName: msg.ContractName,
		Method:       msg.MethodName,
		Parameters:   msg.Pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf("marshal payload failed, %s", err.Error())
	}

	req := &commonPb.TxRequest{
		Header:    header,
		Payload:   payloadBytes,
		Signature: nil,
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed, %s", err.Error())
	}
	fmt.Printf("################ %s", string(sender.MemberInfo))

	signer := getSigner(sk3, sender)
	signBytes, err := signer.Sign("SM3", rawTxBytes)
	if err != nil {
		log.Fatalf("sign failed, %s", err.Error())
	}

	req.Signature = signBytes
	return (*client).SendRequest(ctx, req)
}

// 获取用户私钥
func GetUserSK(index int) (crypto.PrivateKey, *acPb.SerializedMember) {
	numStr := strconv.Itoa(index)

	keyPath := fmt.Sprintf(UserKeyPathFmt, numStr)
	file, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic(err)
	}
	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}
	certPath := fmt.Sprintf(UserCrtPathFmt, numStr)
	file2, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(err)
	}

	sender := &acPb.SerializedMember{
		OrgId:      fmt.Sprintf(OrgIdFormat, numStr),
		MemberInfo: file2,
		IsFullCert: true,
	}

	return sk3, sender
}

// 获取admin的私钥
func GetAdminSK(index int) (crypto.PrivateKey, *acPb.SerializedMember) {
	numStr := strconv.Itoa(index)

	path := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.key"
	file, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}

	userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
	file2, err := ioutil.ReadFile(userCrtPath)
	fmt.Println("node", numStr, "crt", string(file2))
	if err != nil {
		panic(err)
	}

	// 获取peerId
	peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
	fmt.Println("node", numStr, "peerId", peerId)

	// 构造Sender
	sender := &acPb.SerializedMember{
		OrgId:      fmt.Sprintf(OrgIdFormat, numStr),
		MemberInfo: file2,
		IsFullCert: true,
	}

	return sk3, sender
}
