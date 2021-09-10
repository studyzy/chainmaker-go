/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	CHAIN1         = "chain1"
	IP             = "localhost"
	Port           = 12301
	certPathPrefix = "../config"
	WasmPath       = "wasm/fact-rust-0.7.1.wasm"
	userKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key"
	userCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
	orgIdFormat    = "wx-org%d.chainmaker.org"
	orgId          = fmt.Sprintf(orgIdFormat, 1)
	contractName   = "contract2"
	runtimeType    = commonPb.RuntimeType_WASMER
	prePathFmt     = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"

	isTls = false
)

var caPaths = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}

func main() {
	createContract := flag.Bool("c", true, "create Contract")
	flag.Parse()

	conn, err := initGRPCConnect(isTls)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	client := apiPb.NewRpcNodeClient(conn)

	file, err := ioutil.ReadFile(userKeyPath)
	if err != nil {
		panic(err)
	}

	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}

	// 1) 合约创建
	if *createContract {
		testCreate(sk3, client, CHAIN1)
		time.Sleep(5 * time.Second)
	}
	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 4; j++ {
				var txId string
				txId = testInvoke(sk3, client, CHAIN1)
				fmt.Printf("txId: %s\n", txId)
			}
			wg.Done()
			time.Sleep(time.Millisecond * 100)
		}()
	}
	wg.Wait()
}

var marshalStr = "marshal payload failed, %s"

func testCreate(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) {

	txId := utils.GetRandTxId()

	fmt.Printf("\n============ create contract [%s] ============\n", txId)

	wasmBin, _ := ioutil.ReadFile(WasmPath)
	fmt.Println("===============wasmBin[start]===============")
	fmt.Println(hex.EncodeToString(wasmBin))
	fmt.Println("===============wasmBin[end]===============")
	wasmBin, _ = hex.DecodeString(bytesCode)

	var pairs []*commonPb.KeyValuePair

	//method := commonPb.TxType_MANAGE_USER_CONTRACT.String()
	payload, _ := utils.GenerateInstallContractPayload(contractName, "1.0.0", runtimeType, wasmBin, pairs)

	//payload := &commonPb.Payload{
	//	ChainId: chainId,
	//	Contract: &commonPb.Contract{
	//		ContractName:    contractName,
	//		ContractVersion: "1.0.0",
	//		//RuntimeType:     commonPb.RuntimeType_GASM,
	//		RuntimeType: runtimeType,
	//	},
	//	Method:     method,
	//	Parameters: pairs,
	//	ByteCode:   wasmBin,
	//}

	//if endorsement, err := acSign(payload, []int{1, 2, 3, 4}); err == nil {
	//	payload.Endorsement = endorsement
	//} else {
	//	log.Fatalf("failed to sign endorsement, %s", err.Error())
	//	return
	//}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalStr, err.Error())
		return
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	if resp != nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	}
}

func testInvoke(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "time",
			Value: []byte("counter1"),
		},
		{
			Key:   "file_hash",
			Value: []byte("counter2"),
		},
		{
			Key:   "file_name",
			Value: []byte("counter3"),
		},
		{
			Key:   "tx_id",
			Value: []byte("counter4"),
		},
	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       "save",
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalStr, err.Error())
		return txId
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payloadBytes)
	if resp != nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	}

	return txId
}

func proposalRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte) *commonPb.TxResponse {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	if txId == "" {
		txId = utils.GetRandTxId()
	}

	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		return nil
	}

	// 构造Sender
	sender := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		////IsFullCert: true,
	}
	req := &commonPb.TxRequest{
		Payload: &commonPb.Payload{
			ChainId: chainId,
			//Sender:         sender,
			TxType:         txType,
			TxId:           txId,
			Timestamp:      time.Now().Unix(),
			ExpirationTime: 0,
		},
		Sender: &commonPb.EndorsementEntry{Signer: sender},
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed, %s", err.Error())
		return nil
	}

	fmt.Errorf("################ %s", string(sender.MemberInfo))
	signBytes, err := getSigner(sk3, sender).Sign("SHA256", rawTxBytes)
	if err != nil {
		log.Fatalf("sign failed, %s", err.Error())
		return nil
	}

	req.Sender.Signature = signBytes
	result, err := (client).SendRequest(ctx, req)
	if err == nil {
		return result
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println("WARN: client.call err: deadline")
		return nil
	}
	fmt.Printf("ERROR: client.call err: %v\n", err)
	return nil
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

func initGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", IP, Port)

	if useTLS {
		tlsClient := ca.CAClient{
			ServerName: "chainmaker.org",
			CaPaths:    caPaths,
			CertFile:   userCrtPath,
			KeyFile:    userKeyPath,
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

func constructPayload(contractName, method string, pairs []*commonPb.KeyValuePair) []byte {
	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		log.Fatalf(marshalStr, err.Error())
		return nil
	}

	return payloadBytes
}

//
//func acSign(msg *commonPb.Payload, orgIdList []int) ([]*commonPb.EndorsementEntry, error) {
//	msg.Endorsement = nil
//	bytes, _ := proto.Marshal(msg)
//
//	signers := make([]protocol.SigningMember, 0)
//	for _, orgId := range orgIdList {
//
//		numStr := strconv.Itoa(orgId)
//		path := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.key"
//		file, err := ioutil.ReadFile(path)
//		if err != nil {
//			panic(err)
//		}
//		sk, err := asym.PrivateKeyFromPEM(file, nil)
//		if err != nil {
//			panic(err)
//		}
//
//		userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
//		file2, err := ioutil.ReadFile(userCrtPath)
//		fmt.Println("node", orgId, "crt", string(file2))
//		if err != nil {
//			panic(err)
//		}
//
//		// 获取peerId
//		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
//		fmt.Println("node", orgId, "peerId", peerId)
//
//		// 构造Sender
//		sender1 := &acPb.Member{
//			OrgId:      fmt.Sprintf(orgIdFormat, orgId),
//			MemberInfo: file2,
//			//IsFullCert: true,
//		}
//
//		signer := getSigner(sk, sender1)
//		signers = append(signers, signer)
//	}
//
//	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, "SHA256")
//}
