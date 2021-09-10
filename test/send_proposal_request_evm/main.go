/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker-go/test/common"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/evmutils"
	evm "chainmaker.org/chainmaker/common/v2/evmutils"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	MAX_CNT         = 1
	CHAIN1          = "chain1"
	CHAIN2          = "chain2"
	IP              = "localhost"
	Port            = 12301
	certPathPrefix  = "../../config"
	ByteCodeHexPath = "../../test/wasm/evm-token.hex"
	ByteCodePath    = "../../test/wasm/evm-token.bin"
	ABIPath         = "../../test/wasm/evm-token.abi"
	userKeyPath     = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key"
	userCrtPath     = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
	adminKeyPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key"
	adminCrtPath    = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"

	orgId         = "wx-org1.chainmaker.org"
	contractName1 = "0x7162629f540a9e19eCBeEa163eB8e48eC898Ad00"
	runtimeType   = commonPb.RuntimeType_EVM
	prePathFmt    = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"

	//client1Addr = "1087848554046178479107522336262214072175637027873"
)

var (
	contractAddr, _ = evmutils.MakeAddressFromString("cont_01")
	//contractName    = contractAddr.String()
	contractName = hex.EncodeToString(contractAddr.Bytes())
)

//var caPaths = []string{certPathPrefix + "/certs/wx-org1/ca"}
var caPaths = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}
var AbiJson = ""

func main() {
	fmt.Println("contractName:", contractName)

	flag.Parse()
	common.SetCertPathPrefix(certPathPrefix)

	conn, err := initGRPCConnect(true)
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
	fmt.Println("---------------A(User1) 创建ERC20合约-------------")
	testCreate(sk3, &client, CHAIN1)
	time.Sleep(4 * time.Second)
	fmt.Println("---------------查询A(User1) B(Admin1)账户余额-------------")

	balanceA := testQueryBalance(sk3, &client, CHAIN1, userCrtPath)
	if balanceA != "1000000000000000000000000000" {
		panic("balance A not equal 1000000000000000000000000000")
	}
	balanceB := testQueryBalance(sk3, &client, CHAIN1, adminCrtPath)
	if balanceB != "0" {
		panic("balance B not equal 0")
	}
	fmt.Println("---------------发起User1给Admin1的转账-------------")
	testTransfer(sk3, &client, CHAIN1)
	time.Sleep(4 * time.Second)
	fmt.Println("---------------查询AB账户余额-------------")
	balanceA = testQueryBalance(sk3, &client, CHAIN1, userCrtPath)
	if balanceA != "999999999999999999999999990" {
		panic("balance A not equal 999999999999999999999999990")
	}
	balanceB = testQueryBalance(sk3, &client, CHAIN1, adminCrtPath)
	if balanceB != "10" {
		panic("balance B not equal 10")
	}
}

func testAddress() {

	{
		fmt.Println()
		evmInt := evmutils.FromDecimalString("123")
		fmt.Println(evmInt.String())
		fmt.Printf("%x \n", evmInt.AsStringKey())
		fmt.Printf("%x \n\n", evmInt.Bytes())
	}
	{
		name := "000000000000000000000000000000000007b"
		evmInt := evmutils.FromHexString(name)
		fmt.Println(evmInt.String())
		fmt.Printf("%x \n\n", evmInt.AsStringKey())
	}
	for i := 0; i < 100000; i++ {
		evmInt, _ := evmutils.MakeAddressFromString("contractNamecontractNamecontractNamecontractName" + strconv.Itoa(i))
		if len(evmInt.String()) != 49 && len(evmInt.String()) != 48 && len(evmInt.String()) != 47 {
			fmt.Println(evmInt.String()+" i="+strconv.Itoa(i)+" len", len(evmInt.String()))
		}
		val := "@#$%^&*()_{PKJHCVBN<" + strconv.Itoa(i)
		evmInt, _ = evmutils.MakeAddressFromString(val)
		decimalString := evmInt.String()
		if len(decimalString) != 49 && len(decimalString) != 48 && len(decimalString) != 47 {
			fmt.Println(evmInt.String()+" i="+strconv.Itoa(i)+" len", len(decimalString))

			address := evmutils.Keccak256([]byte(val))
			addr := hex.EncodeToString(address)[24:]
			if len(addr) != 40 || len(evmInt.String()) == 45 {
				fmt.Println(addr, len(addr))

				evmAddr := evmutils.FromDecimalString(decimalString)
				hexAddr := hex.EncodeToString(evmAddr.Bytes())
				fmt.Println("hexAddr", hexAddr, len(hexAddr))
				hexAddr = hex.EncodeToString([]byte(evmAddr.AsStringKey()))
				fmt.Println("hexAddr", hexAddr, len(hexAddr))
			}
		}
	}
	name := "ebda2efd8e80ade444cd77891a3551a2ccc68698"
	evmInt, _ := evmutils.MakeAddressFromHex(name)
	fmt.Println(evmInt.String())
	fmt.Printf("%x", evmInt.AsStringKey())

	if evmutils.Has0xPrefix(name) {
		name = name[2:]
	}
	fmt.Println("FromHexString")
	evmInt = evmutils.FromHexString(name)
	fmt.Println(evmInt.String())
	fmt.Printf("%x \n", evmInt.AsStringKey())

	evmInt = evmutils.FromString(name)
	fmt.Println(evmInt.String())
	fmt.Println(evmInt.AsStringKey())

	fmt.Println()
	evmInt = evmutils.FromDecimalString("4409028169148773658499342516519739136830670")
	fmt.Println(evmInt.String())
	fmt.Printf("%x \n", evmInt.AsStringKey())

	fmt.Println()
	evmInt, _ = evmutils.MakeAddressFromString("contractNamecontractNamecontractNamecontractName")
	fmt.Println(evmInt.String())
	fmt.Printf("%x \n", evmInt.AsStringKey())
}
func convertHex2Bin(hexPath, binPath string) error {
	hexBytes, err := ioutil.ReadFile(hexPath)
	if err != nil {
		return err
	}
	bin, err := hex.DecodeString(string(hexBytes))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(binPath, bin, 777)
}

func testCreate(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	convertHex2Bin(ByteCodeHexPath, ByteCodePath)
	abi, err := ioutil.ReadFile(ABIPath)
	if err != nil {
		panic(err.Error())
	}
	AbiJson = string(abi)
	common.CreateContract(sk3, client, chainId, contractName, ByteCodePath, runtimeType)
}

func testQueryBalance(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string, certPath string) string {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ query contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson))
	method0 := "balanceOf"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {

		method = method0

		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		client1Addr, err := getSKI(certPath)
		checkErr(err)
		fmt.Printf("User1 SKI:%s\n", client1Addr)
		addr, err := evm.MakeAddressFromHex(client1Addr)

		dataByte, err := myAbi.Pack(method, evm.BigToAddress(addr))

		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(data),
			},
		}

	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_QUERY_CONTRACT,
		chainId, txId, payload)
	result := ""
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
		result = fmt.Sprintf("%v", v[0])
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)
	return result
}
func getSKI(certPath string) (string, error) {
	certBytes, err := ioutil.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("read cert file [%s] failed, %s", certPath, err)
	}

	block, rest := pem.Decode(certBytes)
	if len(rest) != 0 {
		return "", errors.New("pem.Decode failed, invalid cert")
	}
	cert, err := bcx509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parseCertificate cert failed, %s", err)
	}

	ski := hex.EncodeToString(cert.SubjectKeyId)
	return ski, nil
}

func testTransfer(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, chainId string) {
	txId := utils.GetRandTxId()
	fmt.Printf("\n============ invoke contract [%s] ============\n", txId)

	// 构造Payload
	var pairs []*commonPb.KeyValuePair
	myAbi, _ := abi.JSON(strings.NewReader(AbiJson))
	method0 := "transfer"
	var method string
	if runtimeType == commonPb.RuntimeType_EVM {
		method = method0
		myAbi, err := abi.JSON(strings.NewReader(AbiJson))
		checkErr(err)
		toSki, err := getSKI(adminCrtPath)
		checkErr(err)
		addr, err := evm.MakeAddressFromHex(toSki)
		checkErr(err)
		dataByte, err := myAbi.Pack(method, evm.BigToAddress(addr), big.NewInt(10))
		checkErr(err)

		data := hex.EncodeToString(dataByte)
		fmt.Println("data 1 :", data)
		method = data[0:8]
		pairs = []*commonPb.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(data),
			},
		}

	}

	payload := &commonPb.Payload{
		ContractName: contractName,
		Method:       method,
		Parameters:   pairs,
	}

	resp := proposalRequest(sk3, client, commonPb.TxType_INVOKE_CONTRACT,
		chainId, txId, payload)
	if resp.ContractResult != nil {
		v, _ := myAbi.Unpack(method0, resp.ContractResult.Result)
		fmt.Println(method0, "->", v)
	}
	fmt.Printf("send tx resp: code:%d, msg:%s, payload:%+v\n", resp.Code, resp.Message, resp.ContractResult)

}

func proposalRequest(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, txType commonPb.TxType,
	chainId, txId string, payload *commonPb.Payload) *commonPb.TxResponse {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	if txId == "" {
		txId = utils.GetRandTxId()
	}

	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}

	// 构造Sender
	//pubKeyString, _ := sk3.PublicKey().String()
	sender := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		//IsFullCert: true,
		//MemberInfo: []byte(pubKeyString),
	}
	payload.ChainId = chainId
	payload.TxType = txType
	payload.TxId = txId
	payload.Timestamp = time.Now().Unix()

	req := &commonPb.TxRequest{
		Payload: payload,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		log.Fatalf("CalcUnsignedTxRequest failed, %s", err.Error())
		os.Exit(0)
	}

	fmt.Errorf("################ %s", string(sender.MemberInfo))

	signer := getSigner(sk3, sender)
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	//signBytes, err := signer.Sign("SM3", rawTxBytes)
	if err != nil {
		log.Fatalf("sign failed, %s", err.Error())
		os.Exit(0)
	}

	req.Sender.Signature = signBytes

	result, err := (*client).SendRequest(ctx, req)

	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok {
			if statusErr.Code() == codes.DeadlineExceeded {
				fmt.Println("WARN: client.call err: deadline")
				os.Exit(0)
			}
		}

		fmt.Printf("ERROR: client.call err: %v\n", err)
		os.Exit(0)
	}
	return result
}
func getSigner(sk3 crypto.PrivateKey, sender *acPb.Member) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}
	//fmt.Printf("skPEM: %s\n", skPEM)

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

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
