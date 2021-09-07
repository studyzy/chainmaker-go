/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	"github.com/mr-tron/base58/base58"

	"chainmaker.org/chainmaker-go/test/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"

	"chainmaker.org/chainmaker-go/accesscontrol"
	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/utils/v2"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CHAIN1           = "chain1"
	certPathPrefix   = "../../build/crypto-config"
	certWasmPath     = "../wasm/rust-fact-2.0.0.wasm"
	addWasmPath      = "../wasm/rust-counter-2.0.0.wasm"
	userKeyPath      = certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.sign.key"
	userCrtPath      = certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
	orgId            = "wx-org1.chainmaker.org"
	certContractName = "ex_fact"
	addContractName  = "add"

	prePathFmt  = certPathPrefix + "/wx-org%s.chainmaker.org/user/admin1/"
	OrgIdFormat = "wx-org%d.chainmaker.org"
	tps         = 10000 //

	userKeyPathFormat  = certPathPrefix + "/wx-org%d.chainmaker.org/user/client1/client1.sign.key"
	userCertPathFormat = certPathPrefix + "/wx-org%d.chainmaker.org/user/client1/client1.sign.crt"
)

var (
	caPaths = [][]string{
		{certPathPrefix + "/wx-org1.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org2.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org3.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org4.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org5.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org6.chainmaker.org/ca"},
		{certPathPrefix + "/wx-org7.chainmaker.org/ca"},
	}
	userKeyPaths = []string{
		certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "/wx-org2.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "/wx-org3.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "/wx-org4.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "/wx-org5.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "/wx-org6.chainmaker.org/user/client1/client1.sign.key",
		certPathPrefix + "/wx-org7.chainmaker.org/user/client1/client1.sign.key",
	}
	userCrtPaths = []string{
		certPathPrefix + "/wx-org1.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "/wx-org2.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "/wx-org3.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "/wx-org4.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "/wx-org5.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "/wx-org6.chainmaker.org/user/client1/client1.sign.crt",
		certPathPrefix + "/wx-org7.chainmaker.org/user/client1/client1.sign.crt",
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
		"127.0.0.1",
		"127.0.0.1",
		"127.0.0.1",
	}
	Ports = []int{
		12301,
		12302,
		12303,
		12304,
		12305,
		12306,
		12307,
	}
)

var (
	trustRootOrgId     = ""
	trustRootCrtPath   = ""
	nodeOrgOrgId       = ""
	nodeOrgAddresses   = ""
	consensusExtKeys   = ""
	consensusExtValues = ""

	dposParamFrom       = ""
	dposParamTo         = ""
	dposParamValue      = ""
	dposParamAllowancer = ""

	dposParamEpochId = ""
)

func main() {
	common.SetCertPathPrefix(certPathPrefix)
	var (
		step     int
		wasmType int
	)
	flag.IntVar(&step, "step", 1, "0: add certs, 1: creat contract, 2: add trustRoot, 3: add validator,"+
		" 4: get chainConfig, 5: delete validatorNode, 6: updateConsensus param, 7: mint token, 8: transfer, 9: transferFrom,"+
		" 10: allowance, 11: approve, 12: burn, 13: transferOwnership, 14: owner, 15: decimals, 16: balanceOf, 17: delegate,"+
		" 18: undelegate, 19: getAllValidator, 20: readEpochByID, 21:readLatestEpoch, 22: setRelationshipForAddrAndNodeId,")
	flag.IntVar(&wasmType, "wasm", 0, "0: cert, 1: counter")
	flag.StringVar(&trustRootCrtPath, "trust_root_crt", "", "node crt that will be added to the trust root")
	flag.StringVar(&trustRootOrgId, "trust_root_org_id", "", "node orgID that will be added to the trust root")
	flag.StringVar(&nodeOrgOrgId, "nodeOrg_org_id", "", "node orgID that will be added")
	flag.StringVar(&nodeOrgAddresses, "nodeOrg_addresses", "", "node address that will be added")
	flag.StringVar(&consensusExtKeys, "consensus_keys", "", "key1,key2,key3")
	flag.StringVar(&consensusExtValues, "consensus_Values", "", "value1,value2,value3")

	flag.StringVar(&dposParamFrom, "dpos_from", "", "sender of msg")                      // 谁来发送这笔交易，可能具有业务意义，也可能没有
	flag.StringVar(&dposParamTo, "dpos_to", "", "who will be send to")                    // 接收方，可以是一个地址或其他方式
	flag.StringVar(&dposParamValue, "dpos_value", "", "value of token")                   // token值，该参数可有可无
	flag.StringVar(&dposParamAllowancer, "dpos_allowancer", "", "value of allowanced_ID") // 世代id
	flag.StringVar(&dposParamEpochId, "dpos_epoch_id", "", "value of epoch_id")           // 世代id

	flag.Parse()

	conn, err := initGRPCConn(true, 0)
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
	switch step {
	case 0: // 0) 添加个人证书上链
		addCerts(4)
		time.Sleep(1 * time.Second)
		testCertQuery(sk3, client)
		return
	case 1: // 1) 合约创建
		testCreate(sk3, client, CHAIN1, wasmType)
		return
	case 2: // 2) 添加trustRoot
		trustRootAdd(sk3, client, CHAIN1)
	case 3: // 3) 添加节点到members
		nodeOrgAdd(sk3, client, CHAIN1)
	case 4: // 4) 查看链配置
		config := getChainConfig(sk3, client, CHAIN1)
		fmt.Println(config)
	case 5: // 5) 删除节点
		nodeOrgDelete(sk3, client, CHAIN1)
	case 6: // 6)修改链上配置
		consensusExtUpdate(sk3, client, CHAIN1)

	// DPoS_ERC20合约测试工具
	case 7: // 7)增发token
		mint() // ./main -step 7 -dpos_from="ownerID" -dpos_to="validatorID/validatorAddress" -dpos_value="250000000000000000000000"
	case 8: // 8)向某一用户转移token
		transfer() // ./main -step 8 -dpos_from="validatorID" -dpos_to="validatorID/validatorAddress" -dpos_value="250000000000000000000000"
	case 9: // 9)从某一用户向另一用户转移token
		transferFrom() // ./main -step 9 -dpos_from="allowanced_ValidatorID" -dpos_allowancer= "allowancer_Address" -dpos_to="validatorID/validatorAddress" -dpos_value="250000000000000000000000"
	case 10: // 10)查询某一用户授权另一用户额度
		allowance(sk3, client) // ./main -step 10 -dpos_from="validatorID" -dpos_to="validatorID/validatorAddress"
	case 11: // 11)授权另一用户额度
		approve() // ./main -step 11 -dpos_from="validatorID" -dpos_to="validatorID/validatorAddress" -dpos_value="250000000000000000000000"
	case 12: // 12)燃烧一定数量的代币
		burn() // ./main -step 12 -dpos_from="validatorID" -dpos_to="validatorID/validatorAddress"
	case 13: // 13)转移拥有者给其他账户
		transferOwnership() // ./main -step 13 -dpos_from="ownerID" -dpos_to="validatorID/validatorAddress"
	case 14: // 14)获得token拥有者
		owner(sk3, client) // ./main -step 14
	case 15: // 15)获得decimals
		decimals(sk3, client) // ./main -step 15
	case 16: // 16)查询指定用户余额
		balanceOf(sk3, client) // ./main -step 16 -dpos_to="validatorID/validatorAddress"

	// DPoS_Stake合约测试工具
	case 17: // 17)质押指定token
		delegate(sk3, client) // ./main -step 17 -dpos_from="" -dpos_to="validatorAddress" -dpos_value="1000000" ok
	case 18: // 18)解质押指定token
		undelegate(sk3, client) // ./main -step 18 -dpos_from="" -dpos_to="validatorAddress" -dpos_value="1000000" ok
	case 19: // 19)获得所有满足最低抵押条件验证人
		getAllValidator(sk3, client) // ./main -step 19 -dpos_from="" -dpos_to="" -dpos_value="" ok
	case 20: // 20)获得指定验证人数据
		getValidatorByAddress(sk3, client) // ./main -step 20 -dpos_from="" -dpos_to="validatorAddress" -dpos_value="" ok
	case 21: // 21)获得指定用户的所有抵押数据
		getDelagationsByAddress(sk3, client) // ./main -step 21 -dpos_from="" -dpos_to="" -dpos_value="delegatorAddress" ok
	case 22: // 22)获得指定用户在指定验证人的抵押数据
		getUserDelegationByValidator(sk3, client) // ./main -step 22 -dpos_from="" -dpos_to="validatorAddress" -dpos_value="delegatorAddress" ok
	case 23: // 23)获取指定ID的世代数据
		readEpochByID(sk3, client) // ./main -step 23 -dpos_from="" -dpos_to="" -dpos_value="1" ok
	case 24: // 24)读取当前世代数据
		readLatestEpoch(sk3, client) // ./main -step 24 -dpos_from="" -dpos_to="" -dpos_value="" ok
	case 25: // 25)设置地址和NodeID之间的关系
		setRelationshipForAddrAndNodeId(sk3, client) // ./main -step 25 -dpos_from="5" -dpos_to="5" -dpos_value="nodeID" ok
	case 26: // 26)查询地址和NodeID之间的关系
		getRelationshipForAddrAndNodeId(sk3, client) // ./main -step 26 -dpos_from="" -dpos_to="validatorAddress" -dpos_value="" ok
	// dpos 基础数据查询
	case 27: // 27)读取验证人最少抵押token数量
		readMinSelfDelegation(sk3, client) // ./main -step 27 -dpos_from=1 -dpos_to=1 ok
	case 28: // 28)更新验证人最少抵押token数量
		updateMinSelfDelegation(sk3, client) // ./main -step 28 -dpos_from=1 -dpos_to=1 -dpos_value="25000000000000000000001" ok
	case 29: // 29)读取每个世代验证人数量
		readEpochValidatorNumber(sk3, client) // ./main -step 29 -dpos_from=1 -dpos_to=1 ok
	case 30: // 30)更新每个世代验证人数量
		updateEpochValidatorNumber(sk3, client) // ./main -step 30 -dpos_from=1 -dpos_to=1 -dpos_value="5" ok
	case 31: // 31)读取世代的出块数量
		readEpochBlockNumber(sk3, client) // ./main -step 31 -dpos_from=1 -dpos_to=1 ok
	case 32: // 32)更新世代的出块数量
		updateEpochBlockNumber(sk3, client) // ./main -step 32 -dpos_from=1 -dpos_to=1 -dpos_value=2 ok

	default:
		panic("only three flag: upload cert(1), create contract(1), invoke contract(2)")
	}
}

var (
	signFailedStr    = "sign failed, %s"
	marshalFailedStr = "marshal payload failed, %s"
	deadLineErr      = "WARN: client.call err: deadline"
)

func testCertQuery(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}
	certId, err := utils.GetCertificateIdHex(file, crypto.CRYPTO_ALGO_SHA256)
	if err != nil {
		panic(err)
	}
	fmt.Println(certId)
	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "cert_hashes",
		Value: []byte(certId),
	})
	resp, err := QueryRequestWithCertID(sk3, &client, "", pairs)
	if err == nil {
		fmt.Printf("response: %v\n", resp)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
	}
	return
}

func QueryRequestWithCertID(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient,
	txId string, pairs []*commonPb.KeyValuePair) (*commonPb.TxResponse, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()
	if txId == "" {
		txId = utils.GetRandTxId()
	}
	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}
	certId, err := utils.GetCertificateId(file, crypto.CRYPTO_ALGO_SHA256)
	if err != nil {
		panic(err)
	}
	// 构造Sender
	sender := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: certId,
		MemberType: acPb.MemberType_CERT_HASH,
	}
	senderFull := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		//IsFullCert: true,
	}
	// 构造Header
	header := &commonPb.Payload{
		ChainId: CHAIN1,
		//Sender:         sender,
		TxType:         commonPb.TxType_QUERY_CONTRACT,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,

		ContractName: syscontract.SystemContract_CERT_MANAGE.String(),
		Method:       syscontract.CertManageFunction_CERTS_QUERY.String(),
		Parameters:   pairs,
	}
	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	log.Fatalf(marshalFailedStr, err.Error())
	//}
	req := &commonPb.TxRequest{
		Payload: header,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in QueryRequestWithCertID, %s", err.Error())
	}
	signer := getSigner(sk3, senderFull)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	req.Sender.Signature = signBytes
	return (*client).SendRequest(ctx, req)
}
func testCreate(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string, wasmType int) {

	wasmPath := certWasmPath
	wasmName := certContractName
	if wasmType == 1 {
		wasmPath = addWasmPath
		wasmName = addContractName
	}

	common.CreateContract(sk3, &client, CHAIN1, wasmName, wasmPath, commonPb.RuntimeType_WASMER)

}
func proposalRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payloadBytes []byte, index int) *commonPb.TxResponse {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(60*time.Second)))
	defer cancel()
	if txId == "" {
		txId = utils.GetRandTxId()
	}
	file, err := ioutil.ReadFile(userCrtPaths[index])
	if err != nil {
		panic(err)
	}
	// 构造Sender
	//pubKeyString, _ := sk3.PublicKey().String()
	sender := &acPb.Member{
		OrgId:      orgIds[index],
		MemberInfo: file,
		//IsFullCert: true,
		//MemberInfo: []byte(pubKeyString),
	}
	// 构造Header
	header := &commonPb.Payload{
		ChainId: chainId,
		//Sender:         sender,
		TxType:         txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,
	}
	req := &commonPb.TxRequest{
		Payload: header,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in proposalRequestOld, %s", err.Error())
	}
	signer := getSigner(sk3, sender)
	//signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	req.Sender.Signature = signBytes
	result, err := client.SendRequest(ctx, req)
	if err == nil {
		return result
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
	}
	fmt.Printf("ERROR: client.call err in proposalRequestOld: %v\n", err)
	os.Exit(0)
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

//func acSignWithManager(msg *commonPb.Payload, orgIdList []int) ([]*commonPb.EndorsementEntry, error) {
//	msg.Endorsement = nil
//	bytes, _ := proto.Marshal(msg)
//	signers := make([]protocol.SigningMember, 0)
//	for _, orgId := range orgIdList {
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
//		userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
//		file2, err := ioutil.ReadFile(userCrtPath)
//		fmt.Println("node", orgId, "crt", string(file2))
//		if err != nil {
//			panic(err)
//		}
//		// 获取peerId
//		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
//		fmt.Println("node", orgId, "peerId", peerId)
//		// 构造Sender
//		sender1 := &acPb.Member{
//			OrgId:      "wx-org" + numStr + ".chainmaker.org",
//			MemberInfo: file2,
//			//IsFullCert: true,
//		}
//		signer := getSigner(sk, sender1)
//		signers = append(signers, signer)
//	}
//	return accesscontrol.MockSignWithMultipleNodes(bytes, signers, crypto.CRYPTO_ALGO_SHA256)
//}

func getKeysAndCertsPath(orgIdList []int) (keysFile, certsFile []string) {
	keysFile = make([]string, 0, len(orgIdList))
	certsFile = make([]string, 0, len(orgIdList))
	for _, orgId := range orgIdList {
		numStr := strconv.Itoa(orgId)
		keyPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.key"
		userCrtPath := fmt.Sprintf(prePathFmt, numStr) + "admin1.sign.crt"
		keysFile = append(keysFile, keyPath)
		certsFile = append(certsFile, userCrtPath)
	}
	return keysFile, certsFile
}

func addCerts(count int) {
	for i := 0; i < count; i++ {
		txId := utils.GetRandTxId()
		sk, member := getUserSK(i+1, userKeyPaths[i], userCrtPaths[i])
		resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{TxId: txId, ChainId: CHAIN1,
			TxType: commonPb.TxType_INVOKE_CONTRACT, ContractName: syscontract.SystemContract_CERT_MANAGE.String(), MethodName: syscontract.CertManageFunction_CERT_ADD.String()})
		if err == nil {
			fmt.Printf("addCerts send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
			continue
		}
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			fmt.Println(deadLineErr)
			return
		}
		fmt.Printf("ERROR: client.call err in addCerts: %v\n", err)
		return
	}
}

// 获取用户私钥
func getUserSK(orgIDNum int, keyPath, certPath string) (crypto.PrivateKey, *acPb.Member) {
	file, err := ioutil.ReadFile(keyPath)
	if err != nil {
		panic(err)
	}
	sk3, err := asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}
	file2, err := ioutil.ReadFile(certPath)
	if err != nil {
		panic(err)
	}
	sender := &acPb.Member{
		OrgId:      fmt.Sprintf(OrgIdFormat, orgIDNum),
		MemberInfo: file2,
		//IsFullCert: true,
	}
	return sk3, sender
}
func updateSysRequest(sk3 crypto.PrivateKey, sender *acPb.Member, isTls bool, msg *native.InvokeContractMsg) (*commonPb.TxResponse, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalln(err)
		}
	}()

	conn, err := initGRPCConn(isTls, 0)
	if err != nil {
		panic(err)
	}
	client := apiPb.NewRpcNodeClient(conn)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()
	if msg.TxId == "" {
		msg.TxId = utils.GetRandTxId()
	}
	// 构造Header
	header := &commonPb.Payload{
		ChainId: msg.ChainId,
		//Sender:         sender,
		TxType:         msg.TxType,
		TxId:           msg.TxId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,

		ContractName: msg.ContractName,
		Method:       msg.MethodName,
		Parameters:   msg.Pairs,
	}
	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	log.Fatalf(marshalFailedStr, err.Error())
	//}
	req := &commonPb.TxRequest{
		Payload: header,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed in updateSysRequest, %s", err.Error())
	}
	signer := getSigner(sk3, sender)
	signBytes, err := signer.Sign(crypto.CRYPTO_ALGO_SHA256, rawTxBytes)
	if err != nil {
		log.Fatalf(signFailedStr, err.Error())
	}
	fmt.Println(crypto.CRYPTO_ALGO_SHA256, "signBytes"+hex.EncodeToString(signBytes), "rawTxBytes="+hex.EncodeToString(rawTxBytes))
	err = signer.Verify(crypto.CRYPTO_ALGO_SHA256, rawTxBytes, signBytes)
	if err != nil {
		panic(err)
	}
	req.Sender.Signature = signBytes
	fmt.Println(req)
	return client.SendRequest(ctx, req)
}

func getChainConfig(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) *configPb.ChainConfig {
	// 构造Payload
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_CHAIN_CONFIG.String(), syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, "", payloadBytes, 0)
	chainConfig := &configPb.ChainConfig{}
	if err = proto.Unmarshal(resp.ContractResult.Result, chainConfig); err != nil {
		log.Fatalf("unmarshal bytes failed, err: %s", err)
	}
	return chainConfig
}

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

type InvokerMsg struct {
	txType       commonPb.TxType
	chainId      string
	txId         string
	method       string
	contractName string
	oldSeq       uint64
	pairs        []*commonPb.KeyValuePair
}

func trustRootAdd(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) error {
	// 构造Payload
	if trustRootOrgId == "" || trustRootCrtPath == "" {
		log.Fatalf("the trustRoot orgId or crt is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(trustRootOrgId),
	})
	file, err := os.Open(trustRootCrtPath)
	if err != nil {
		log.Fatalf("open file failed: %s, reason: %s", trustRootCrtPath, err)
	}
	defer file.Close()
	bz, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("read content from file failed: %s", err)
	}
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "root",
		Value: bz,
	})

	config := getChainConfig(sk3, client, chainId)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), pairs: pairs, oldSeq: config.Sequence})
	if err != nil {
		log.Fatalf("create update request failed, err: %s", err)
	}
	fmt.Println("txId: ", txId, "; result: ", resp)
	return nil
}

func configUpdateRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, msg *InvokerMsg) (*commonPb.TxResponse, string, error) {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	txId := utils.GetRandTxId()
	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		return nil, "", err
	}

	// 构造Sender
	senderFull := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		//IsFullCert: true,
	}

	// 构造Header
	header := &commonPb.Payload{
		ChainId: msg.chainId,
		//Sender:         senderFull,
		TxType:         msg.txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,

		ContractName: msg.contractName,
		Method:       msg.method,
		Parameters:   msg.pairs,
		Sequence:     msg.oldSeq + 1,
	}
	//adminSignKeys, adminSignCrts := getKeysAndCertsPath([]int{1, 2, 3, 4})
	//entries, err := aclSignSystemContract(*payload, orgIds, adminSignKeys, adminSignCrts)
	//if err != nil {
	//	panic(err)
	//}
	//payload.Endorsement = entries

	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	return nil, "", err
	//}
	req := &commonPb.TxRequest{
		Payload: header,
		Sender:  &commonPb.EndorsementEntry{Signer: senderFull},
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, "", err
	}
	signer := getSigner(sk3, senderFull)
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	if err != nil {
		log.Fatalf("sign msg failed")
	}
	req.Sender.Signature = signBytes

	result, err := client.SendRequest(ctx, req)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return nil, "", fmt.Errorf("client.call err: deadline\n")
		}
		return nil, "", fmt.Errorf("client.call err: %v\n", err)
	}
	return result, txId, nil
}

//func aclSignSystemContract(msg commonPb.Payload, orgIds, adminSignKeys, adminSignCrts []string) ([]*commonPb.EndorsementEntry, error) {
//	msg.Endorsement = nil
//	bytes, _ := proto.Marshal(&msg)
//
//	signers := make([]protocol.SigningMember, 0)
//	orgIdArray := orgIds
//	adminSignKeyArray := adminSignKeys
//	adminSignCrtArray := adminSignCrts
//
//	if len(adminSignKeyArray) != len(adminSignCrtArray) {
//		return nil, errors.New(fmt.Sprintf("admin key len is not equal to crt len: %d, %d", len(adminSignKeyArray), len(adminSignCrtArray)))
//	}
//	if len(adminSignKeyArray) != len(orgIdArray) {
//		return nil, errors.New(fmt.Sprintf("admin key len:[%d] is not equal to orgId len:[%d]", len(adminSignKeyArray), len(orgIdArray)))
//	}
//
//	for i, key := range adminSignKeyArray {
//		file, err := ioutil.ReadFile(key)
//		if err != nil {
//			panic(err)
//		}
//		sk3, err := asym.PrivateKeyFromPEM(file, nil)
//		if err != nil {
//			panic(err)
//		}
//
//		file2, err := ioutil.ReadFile(adminSignCrtArray[i])
//		fmt.Println("node", i, "crt", string(file2))
//		if err != nil {
//			panic(err)
//		}
//
//		// 获取peerId
//		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
//		fmt.Println("node", i, "peerId", peerId)
//
//		// 构造Sender
//		sender1 := &acPb.Member{
//			OrgId:      orgIdArray[i],
//			MemberInfo: file2,
//			//IsFullCert: true,
//		}
//		signer := getSigner(sk3, sender1)
//		signers = append(signers, signer)
//	}
//
//	endorsements, err := accesscontrol.MockSignWithMultipleNodes(bytes, signers, crypto.CRYPTO_ALGO_SHA256)
//	if err != nil {
//		return nil, err
//	}
//	fmt.Printf("endorsements:\n%v\n", endorsements)
//	return endorsements, nil
//}

func nodeOrgAdd(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) error {
	// 构造Payload
	if nodeOrgOrgId == "" || nodeOrgAddresses == "" {
		return errors.New("the nodeOrg orgId or addresses is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeOrgOrgId),
	})
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "node_ids",
		Value: []byte(nodeOrgAddresses),
	})

	config := getChainConfig(sk3, client, chainId)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), pairs: pairs, oldSeq: config.Sequence})
	if err != nil {
		log.Fatalf("create configUpdateRequest error")
	}
	fmt.Println("txId: ", txId, ", resp: ", resp)
	return nil
}

func nodeOrgDelete(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) error {
	// 构造Payload
	if nodeOrgOrgId == "" {
		return errors.New("the nodeOrg orgId is empty")
	}
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "org_id",
		Value: []byte(nodeOrgOrgId),
	})

	config := getChainConfig(sk3, client, chainId)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), pairs: pairs, oldSeq: config.Sequence})
	if err != nil {
		log.Fatalf("create configUpdateRequest error")
	}
	fmt.Println("txId: ", txId, ", resp: ", resp)
	return nil
}

func consensusExtUpdate(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, chainId string) error {
	// 构造Payload
	if consensusExtKeys == "" || consensusExtValues == "" {
		log.Fatalf("consensusKeys: %s, consensusValues: %s\n", consensusExtKeys, consensusExtValues)
	}
	consensusExtKeyArray := strings.Split(consensusExtKeys, ",")
	consensusExtValueArray := strings.Split(consensusExtValues, ",")
	if len(consensusExtKeyArray) != len(consensusExtValueArray) {
		log.Fatalf("the consensusExt keys len is not equal to values len, "+
			"keysNum:%d, valueNum:%d", len(consensusExtKeyArray), len(consensusExtValueArray))
	}

	pairs := make([]*commonPb.KeyValuePair, 0)

	for i, key := range consensusExtKeyArray {
		pairs = append(pairs, &commonPb.KeyValuePair{
			Key:   key,
			Value: []byte(consensusExtValueArray[i]),
		})
	}

	config := getChainConfig(sk3, client, chainId)
	resp, txId, err := configUpdateRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT, chainId: chainId,
		contractName: syscontract.SystemContract_CHAIN_CONFIG.String(), method: syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), pairs: pairs, oldSeq: config.Sequence})
	if err != nil {
		log.Fatalf("send update request failed in consensusExtUpdate: %s", err)
	}
	fmt.Println("txId: ", txId, ", resp: ", resp)
	return nil
}

//mint 增发给指定用户token
func mint() {
	sk, member, toAddr, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
		{
			Key:   "value",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		MethodName:   syscontract.DPoSERC20Function_MINT.String(),
		Pairs:        params,
	})

	if err == nil {
		fmt.Printf("mint send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_erc20_mint: %v\n", err)
}

//transfer 向某一用户转移token
func transfer() {
	sk, member, toAddr, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
		{
			Key:   "value",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		MethodName:   syscontract.DPoSERC20Function_TRANSFER.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("transfer send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_erc20_transfer: %v\n", err)
}

//transferFrom 从某一用户向另一用户转移token
func transferFrom() {
	if dposParamAllowancer == "" {
		log.Fatalf("dposParamAllowanced: %s\n", dposParamAllowancer)
	}

	sk, member, toAddr, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "from",
			Value: []byte(dposParamAllowancer),
		},
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
		{
			Key:   "value",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		MethodName:   syscontract.DPoSERC20Function_TRANSFER_FROM.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("transfer_from send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_erc20_transfer_from: %v\n", err)
}

//allowance 查询某一用户授权另一用户额度
func allowance(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	fromAddr, err := loadDposParamsFrom()
	_, _, toAddr, _, err := loadDposParams()
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "from",
			Value: []byte(fromAddr),
		},
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
	}
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_ALLOWANCE.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

//approve 授权另一用户额度
func approve() {
	sk, member, toAddr, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
		{
			Key:   "value",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		MethodName:   syscontract.DPoSERC20Function_APPROVE.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("approve send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_erc20_approve: %v\n", err)
}

//burn 燃烧一定数量的代币
func burn() {
	sk, member, _, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "value",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		MethodName:   syscontract.DPoSERC20Function_BURN.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("burn send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_erc20_burn: %v\n", err)
}

// transferOwnership 转移拥有者给其他账户
func transferOwnership() {
	sk, member, toAddr, _, err := loadDposParams()
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_ERC20.String(),
		MethodName:   syscontract.DPoSERC20Function_TRANSFER_OWNERSHIP.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("transferOwnership send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_erc20_transferOwnership: %v\n", err)
}

//owner 获得token拥有者
func owner(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_OWNER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

//decimals 获得decimals
func decimals(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_DECIMALS.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

//balanceOf 查询指定用户余额
func balanceOf(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	_, _, toAddr, _, err := loadDposParams()
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "owner",
			Value: []byte(toAddr),
		},
	}
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_ERC20.String(), syscontract.DPoSERC20Function_GET_BALANCEOF.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

//delegate 质押token
func delegate(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	sk, member, toAddr, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "to",
			Value: []byte(toAddr),
		},
		{
			Key:   "amount",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		MethodName:   syscontract.DPoSStakeFunction_DELEGATE.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("delegate send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_stake_delegate: %v\n", err)
}

//undelegate 解质押token
func undelegate(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	sk, member, toAddr, value, err := loadDposParams()
	if value == "" {
		log.Fatalf("dposParamValue: %s\n", value)
	}
	params := []*commonPb.KeyValuePair{
		{
			Key:   "from",
			Value: []byte(toAddr),
		},
		{
			Key:   "amount",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId: "", ChainId: CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		MethodName:   syscontract.DPoSStakeFunction_UNDELEGATE.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("undelegate send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		return
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_stake_undelegate: %v\n", err)
}

//getAllValidator 获得所有满足最低抵押条件验证人
func getAllValidator(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_ALL_CANDIDATES.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func getValidatorByAddress(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	_, _, toAddr, _, err := loadDposParams()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: []byte(toAddr),
	})
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_VALIDATOR_BY_ADDRESS.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func getDelagationsByAddress(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: []byte(dposParamValue),
	})
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_DELEGATIONS_BY_ADDRESS.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func getUserDelegationByValidator(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	_, _, toAddress, value, err := loadDposParams()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs,
		&commonPb.KeyValuePair{
			Key:   "delegator_address",
			Value: []byte(value),
		},
		&commonPb.KeyValuePair{
			Key:   "validator_address",
			Value: []byte(toAddress),
		},
	)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_USER_DELEGATION_BY_VALIDATOR.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

//readEpochByID 获取指定ID的世代数据
func readEpochByID(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "epoch_id",
		Value: []byte(dposParamValue),
	})
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_EPOCH_BY_ID.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	if len(resp.ContractResult.Result) > 0 {
		e := &syscontract.Epoch{}
		err = proto.Unmarshal(resp.ContractResult.Result, e)
		if err != nil {
			log.Fatalf("unmarshal vc failed, err: %s", err)
		}
		fmt.Println(e)
	} else {
		fmt.Println("result is null")
	}
	fmt.Println(resp)
}

//readLatestEpoch 读取当前世代数据
func readLatestEpoch(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_LATEST_EPOCH.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	e := &syscontract.Epoch{}
	err = proto.Unmarshal(resp.ContractResult.Result, e)
	if err != nil {
		log.Fatalf("unmarshal vc failed, err: %s", err)
	}
	fmt.Println(e)
	fmt.Println(resp)
}

// setRelationshipForAddrAndNodeId 系加入节点绑定自身身份
func setRelationshipForAddrAndNodeId(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	sk, member, _, value, err := loadDposParams()
	params := []*commonPb.KeyValuePair{
		{
			Key:   "node_id",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId:         "",
		ChainId:      CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		MethodName:   syscontract.DPoSStakeFunction_SET_NODE_ID.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("setRelationshipForAddrAndNodeId send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		if resp != nil {
			return
		}
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in dpos_stake_setNodeID: %v\n", err)
}

// setRelationshipForAddrAndNodeId 系加入节点绑定自身身份
func getRelationshipForAddrAndNodeId(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	_, _, toAddr, _, err := loadDposParams()
	pairs := make([]*commonPb.KeyValuePair, 0)
	pairs = append(pairs, &commonPb.KeyValuePair{
		Key:   "address",
		Value: []byte(toAddr),
	})
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_GET_NODE_ID.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func readMinSelfDelegation(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_MIN_SELF_DELEGATION.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func updateMinSelfDelegation(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	sk, member, _, value, err := loadDposParams()
	params := []*commonPb.KeyValuePair{
		{
			Key:   "min_self_delegation",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId:         "",
		ChainId:      CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		MethodName:   syscontract.DPoSStakeFunction_UPDATE_MIN_SELF_DELEGATION.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		if resp != nil {
			return
		}
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err: %v\n", err)
}

func readEpochValidatorNumber(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_EPOCH_VALIDATOR_NUMBER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func updateEpochValidatorNumber(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	sk, member, _, value, err := loadDposParams()
	params := []*commonPb.KeyValuePair{
		{
			Key:   "epoch_validator_number",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId:         "",
		ChainId:      CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		MethodName:   syscontract.DPoSStakeFunction_UPDATE_EPOCH_VALIDATOR_NUMBER.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		if resp != nil {
			return
		}
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in: %v\n", err)
}

func readEpochBlockNumber(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	pairs := make([]*commonPb.KeyValuePair, 0)
	payloadBytes, err := constructPayload(syscontract.SystemContract_DPOS_STAKE.String(), syscontract.DPoSStakeFunction_READ_EPOCH_BLOCK_NUMBER.String(), pairs)
	if err != nil {
		log.Fatalf("create payload failed, err: %s", err)
	}
	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		CHAIN1, "", payloadBytes, 0)
	fmt.Println(resp)
}

func updateEpochBlockNumber(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient) {
	sk, member, _, value, err := loadDposParams()
	params := []*commonPb.KeyValuePair{
		{
			Key:   "epoch_block_number",
			Value: []byte(value),
		},
	}
	resp, err := updateSysRequest(sk, member, true, &native.InvokeContractMsg{
		TxId:         "",
		ChainId:      CHAIN1,
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_DPOS_STAKE.String(),
		MethodName:   syscontract.DPoSStakeFunction_UPDATE_EPOCH_BLOCK_NUMBER.String(),
		Pairs:        params,
	})
	if err == nil {
		fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
		if resp != nil {
			return
		}
	}
	if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
		fmt.Println(deadLineErr)
		return
	}
	fmt.Printf("ERROR: client.call err in: %v\n", err)
}

func loadDposParams() (crypto.PrivateKey, *acPb.Member, string, string, error) {
	if dposParamTo == "" {
		log.Fatalf("dposParamTo: %s\n", dposParamTo)
	}
	var (
		toAddr string
		toIdx  int64
		err    error
	)
	// 判断dposParams的信息
	toIdx, err = strconv.ParseInt(dposParamTo, 10, 32)
	if err != nil {
		// 判断是否为base58编码
		_, err = base58.Decode(dposParamTo)
		if err != nil {
			log.Fatalf("param is not number or base58, %s", dposParamTo)
		}
		toAddr = dposParamTo
	} else {
		// 获取证书
		userCertPath := fmt.Sprintf(userCertPathFormat, toIdx)
		// 读取内容，并转换为公钥
		userCertBytes, err := ioutil.ReadFile(userCertPath)
		if err != nil {
			panic(err)
		}
		toAddr, err = parseUserAddress(userCertBytes)
		if err != nil {
			log.Fatalf("parse cert to address error, %s", userCertPath)
		}
	}
	var skIdx = 1
	if dposParamFrom != "" {
		ownerIdx, err := strconv.ParseInt(dposParamFrom, 10, 32)
		if err == nil {
			skIdx = int(ownerIdx)
		}
	}
	sk, member := getUserSK(skIdx, userKeyPaths[skIdx-1], userCrtPaths[skIdx-1])
	return sk, member, toAddr, dposParamValue, nil
}

func loadDposParamsFrom() (string, error) {
	if dposParamFrom == "" {
		log.Fatalf("dposParamFrom: %s\n", dposParamFrom)
	}
	var (
		fromAddr string
		fromIdx  int64
		err      error
	)
	// 判断dposParams的信息
	fromIdx, err = strconv.ParseInt(dposParamFrom, 10, 32)
	if err != nil {
		// 判断是否为base58编码
		_, err = base58.Decode(dposParamFrom)
		if err != nil {
			log.Fatalf("param is not number or base58, %s", dposParamFrom)
		}
		fromAddr = dposParamFrom
	} else {
		// 获取证书
		userCertPath := fmt.Sprintf(userCertPathFormat, fromIdx)
		// 读取内容，并转换为公钥
		userCertBytes, err := ioutil.ReadFile(userCertPath)
		if err != nil {
			panic(err)
		}
		fromAddr, err = parseUserAddress(userCertBytes)
		if err != nil {
			log.Fatalf("parse cert to address error, %s", userCertPath)
		}
	}

	return fromAddr, nil
}

// parseUserAddress
func parseUserAddress(member []byte) (string, error) {
	certificate, err := utils.ParseCert(member)
	if err != nil {
		msg := fmt.Errorf("parse cert failed, err: %+v", err)
		return "", msg
	}
	pubKeyBytes, err := certificate.PublicKey.Bytes()
	if err != nil {
		msg := fmt.Errorf("load public key from cert failed, err: %+v", err)
		return "", msg
	}
	// 转换为SHA-256
	addressBytes := sha256.Sum256(pubKeyBytes)
	return base58.Encode(addressBytes[:]), nil
}
