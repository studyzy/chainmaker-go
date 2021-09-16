/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	SenderNumber            = 100
	MsgSenderAndWorkerRatio = 20

	Interval            = 200 // send tx time interval
	ReceiveReqNodeIndex = 1   // [0,1,2,3]

	toSave             string
	DebugRatio         = false
	PerWorkerMsgNumber = 100000000
	SenderBuffer       = 10000
	CallContract       = 0
)

const (
	CHAIN1         = "chain1"
	certPathPrefix = "/big_space/chainmaker/chainmaker-go/build/crypto-config/"

	contractNameFact = "ex_fact"
	addContractName  = "add"
	factFuncName     = "save"
	addFuncName      = "increase"
)

var (
	caPaths = [][]string{
		{certPathPrefix + "wx-org1.chainmaker.org/ca/"},
		{certPathPrefix + "wx-org2.chainmaker.org/ca/"},
		{certPathPrefix + "wx-org3.chainmaker.org/ca/"},
		{certPathPrefix + "wx-org4.chainmaker.org/ca/"},
	}
	userKeyPaths = []string{
		certPathPrefix + "wx-org1.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "wx-org2.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "wx-org3.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "wx-org4.chainmaker.org/user/client1/client1.sign.key",
	}
	userCrtPaths = []string{
		certPathPrefix + "wx-org1.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "wx-org2.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "wx-org3.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "wx-org4.chainmaker.org/user/client1/client1.sign.crt",
	}
	orgIds = []string{
		"wx-org1.chainmaker.org",
		"wx-org2.chainmaker.org",
		"wx-org3.chainmaker.org",
		"wx-org4.chainmaker.org",
	}
	IPs = []string{
		"127.0.0.1",
		"127.0.0.1",
		"127.0.0.1",
		"127.0.0.1",
	}
	Ports = []int{
		12301,
		12302,
		12303,
		12304,
	}
)

func main() {
	flag.IntVar(&SenderNumber, "sender", 100, "The number of sender")
	flag.IntVar(&Interval, "interval", 200, "The time interval between sending the transaction, Millisecond")
	flag.IntVar(&ReceiveReqNodeIndex, "index", 1, "The index of the node to receive the request")
	flag.IntVar(&CallContract, "call", 0, "0: call cert contract, 1: call counter contract")
	flag.Parse()
	randData := make([]byte, 200)
	rand.Read(randData)
	toSave = string(randData)

	createWorkerAndSender(SenderNumber, ReceiveReqNodeIndex)
}

func createWorkerAndSender(senderNum int, nodeIndex int) {
	if senderNum == 0 || senderNum%MsgSenderAndWorkerRatio > 0 {
		log.Fatalf("sender number has to be multiples of ratio; actual sender number[%d], ratio[%d] \n", senderNum, MsgSenderAndWorkerRatio)
	}
	// 1. init sender channel to receive tx from worker
	senderChs := make([]chan *commonPb.TxRequest, senderNum/MsgSenderAndWorkerRatio)
	for i := 0; i < len(senderChs); i++ {
		senderChs[i] = make(chan *commonPb.TxRequest, SenderBuffer)
	}

	// 2. main logic
	wait := sync.WaitGroup{}
	for i := 0; i < senderNum; i++ {
		// 2.1 Assign a Sender to each woker
		if i%MsgSenderAndWorkerRatio == 0 {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				createWorker(nodeIndex, senderChs[index/MsgSenderAndWorkerRatio], nodeIndex)
				close(senderChs[index/MsgSenderAndWorkerRatio])
			}(i)
		}

		// 2.2 create sender to receive tx and send tx to node
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			createSender(nodeIndex, senderChs[index/MsgSenderAndWorkerRatio])
		}(i)
	}
	wait.Wait()
	fmt.Println("performance test is completed")
}

func createWorker(index int, senderCh chan *commonPb.TxRequest, senderIndex int) {
	keyFile, err := ioutil.ReadFile(userKeyPaths[index])
	if err != nil {
		log.Fatal(fmt.Sprintf("read user key file failed: %s\n", err))
	}
	sk, err := asym.PrivateKeyFromPEM(keyFile, nil)
	if err != nil {
		panic(err)
	}

	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		defer wait.Done()
		if DebugRatio {
			msgWorkerWithDebugRatio(index, sk, senderCh, senderIndex)
		} else {
			msgWorker(index, sk, senderCh, senderIndex)
		}
	}()
	wait.Wait()
}

func msgWorker(index int, sk3 crypto.PrivateKey, senderCh chan *commonPb.TxRequest, senderIndex int) {
	signer, certId := getSignerAndCertId(index, sk3)
	for i := 0; i < PerWorkerMsgNumber; i++ {
		senderCh <- createInvokePackage(signer, certId, index)
	}
}

func msgWorkerWithDebugRatio(index int, sk3 crypto.PrivateKey, senderCh chan *commonPb.TxRequest, senderIndex int) {
	signer, certId := getSignerAndCertId(index, sk3)
	for i := 0; i < PerWorkerMsgNumber; i++ {
	Loop:
		for {
			select {
			case senderCh <- createInvokePackage(signer, certId, index):
				break Loop
			default:
				fmt.Printf("sender too slow, need increase the multiple with worker and sender; senderIndex[%d]\n", senderIndex)
				time.Sleep(time.Millisecond)
			}
		}
	}
}

func getSignerAndCertId(index int, sk3 crypto.PrivateKey) (protocol.SigningMember, []byte) {
	file, err := ioutil.ReadFile(userCrtPaths[index])
	if err != nil {
		panic(err)
	}
	certId, err := utils.GetCertificateId(file, "SHA256")
	if err != nil {
		panic(err)
	}
	senderFull := &acPb.Member{
		OrgId:      orgIds[index],
		MemberInfo: file,
		//IsFullCert: true,
	}
	signer := getSigner(sk3, senderFull)

	return signer, certId
}

func getSigner(sk3 crypto.PrivateKey, sender *acPb.Member) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}

	signer, err := accesscontrol.NewCertSigningMember("", sender, skPEM, "")
	if err != nil {
		panic(err)
	}
	return signer
}

func createInvokePackage(signer protocol.SigningMember, certId []byte, index int) *commonPb.TxRequest {
	txId := utils.GetRandTxId()
	var (
		payload    *commonPb.Payload
		rawTxBytes []byte
		signBytes  []byte

		err error
	)

	// 1. create payload
	if CallContract == 1 {
		payload = &commonPb.Payload{
			ContractName: addContractName,
			Method:       addFuncName,
		}

	} else {
		payload = &commonPb.Payload{
			ContractName: contractNameFact,
			Method:       factFuncName,
			Parameters: []*commonPb.KeyValuePair{
				{
					Key:   "file_hash",
					Value: []byte(txId[:10]),
				},
				{
					Key:   "file_name",
					Value: []byte("长安链chainmaker"),
				},
				{
					Key:   "time",
					Value: []byte("1615188470000"),
				},
			},
		}
	}

	// 2. create request with payload
	req := &commonPb.TxRequest{
		Payload: &commonPb.Payload{
			ChainId:        CHAIN1,
			TxType:         commonPb.TxType_INVOKE_CONTRACT,
			TxId:           txId,
			Timestamp:      time.Now().Unix(),
			ExpirationTime: 0,
			ContractName:   payload.ContractName,
			Method:         payload.Method,
			Parameters:     payload.Parameters,
		},
		Sender: &commonPb.EndorsementEntry{Signer: &acPb.Member{
			OrgId:      orgIds[index],
			MemberInfo: certId,
			MemberType: acPb.MemberType_CERT_HASH,
		}},
	}

	// 3. generate the signature on request
	if rawTxBytes, err = utils.CalcUnsignedTxRequestBytes(req); err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed, %s", err.Error())
	}
	if signBytes, err = signer.Sign("SHA256", rawTxBytes); err != nil {
		log.Fatalf("sign failed, %s", err.Error())
	}

	// 4. Assemble the signature in request
	req.Sender.Signature = signBytes
	return req
}

func createSender(index int, receiveCh chan *commonPb.TxRequest) {
	conn, err := initGRPCConn(true, index)
	if err != nil {
		fmt.Println(err)
		return
	}
	client := apiPb.NewRpcNodeClient(conn)
	for {
		select {
		case req := <-receiveCh:
			if req == nil {
				return
			}
			sendReq(client, req)
		}
	}
}

func initGRPCConn(useTLS bool, orgIdIndex int) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", IPs[orgIdIndex], Ports[orgIdIndex])
	fmt.Println(url)

	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths[orgIdIndex],
			CertFile:   userCrtPaths[orgIdIndex],
			KeyFile:    userKeyPaths[orgIdIndex],
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

func sendReq(client apiPb.RpcNodeClient, req *commonPb.TxRequest) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(60*time.Second)))
	defer cancel()

	_, err := client.SendRequest(ctx, req)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			log.Fatal("WARN: client.call err: deadline")
		}
		log.Fatalf("ERROR: client.call err: %v\n", err)
	}
	//fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	time.Sleep(time.Duration(Interval) * time.Millisecond)
}
