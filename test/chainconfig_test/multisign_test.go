package native_test

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	native "chainmaker.org/chainmaker-go/test/chainconfig_test"
	"chainmaker.org/chainmaker-go/test/common"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/helper"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/stretchr/testify/require"
)

const (
	IP                  = "localhost"
	Port                = 12301
	certPathPrefix      = "../../config"
	userKeyPath         = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key"
	userCrtPath         = certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt"
	prePathFmt          = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
	AdminSignKeyPathFmt = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/admin1.sign.key"
	AdminSignCrtPathFmt = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/admin1.sign.crt"
)

var (
	OrgIdFormat     = "wx-org%s.chainmaker.org"
	txId            = ""
	memberNum       int
	timestamp       int64
	WasmPath        = "../wasm/rust-func-verify-2.0.0.wasm"
	WasmUpgradePath = WasmPath
	contractName    = "contract106"
	runtimeType     = commonPb.RuntimeType_WASMER
)
var caPaths = []string{certPathPrefix + "/crypto-config/wx-org1.chainmaker.org/ca"}

func TestMultiSignReq(t *testing.T) {
	common.SetCertPathPrefix(certPathPrefix)
	txId = utils.GetRandTxId()
	log.Printf("txId:%s\n", txId)
	timestamp = time.Now().Unix()
	log.Printf("timestamp:%d\n", timestamp)
	//client, sk3 := InitRun()
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)
	sk, _ := native.GetUserSK(1)
	payload := initPayload()

	resp := common.ProposalMultiRequest(sk, &client, payload.TxType,
		CHAIN1, payload.TxId, payload, []int{1}, timestamp)
	fmt.Println("testMultiSignReq timestamp", timestamp)
	fmt.Println(resp)

	var block *commonPb.Block
	var tx *commonPb.Transaction
	txId = testVerifyContractAccessWithCertManage(t)
	for block == nil {
		block = testGetBlockByTxId(t, client, txId)
	}
	tx = block.Txs[0]
	require.True(t, tx.Result.ContractResult.Code == 1)
	require.True(t, tx.Result.ContractResult.Message == "Access Denied")
}

func TestMultiSignVote(t *testing.T) {
	//txId = utils.GetRandTxId()
	//timestamp = time.Now().Unix()
	memberNum = 2
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)
	sk, _ := native.GetAdminSK(memberNum)
	//client, sk3 := InitRun()
	payload1 := initPayload()
	payload1.Timestamp = timestamp
	payloadBytes, err := payload1.Marshal()
	fmt.Printf("testMultiSignVote1 payload md5 is %x \n", md5.Sum(payloadBytes))
	if err != nil {
		panic(err)
	}

	ee, err := AclSignOne(payloadBytes, memberNum)
	if err != nil {
		panic(err)
	}
	msvi := &syscontract.MultiSignVoteInfo{
		Vote:        syscontract.VoteStatus_AGREE,
		Endorsement: ee,
	}
	msviByte, _ := msvi.Marshal()

	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiVote_VOTE_INFO.String(),
			Value: msviByte,
		},
		{
			Key:   syscontract.MultiVote_TX_ID.String(),
			Value: []byte(payload1.TxId),
		},
	}

	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_VOTE.String(),
		Parameters:   pairs,
	}

	resp := common.ProposalMultiRequest(sk, &client, payload.TxType,
		CHAIN1, "", payload, nil, time.Now().Unix())

	fmt.Println(resp)
}
func TestMultiSignQuery(t *testing.T) {
	//txId = utils.GetRandTxId()
	//timestamp = time.Now().Unix()
	//client, sk3 := InitRun()
	conn, err := native.InitGRPCConnect(isTls)
	require.NoError(t, err)
	client := apiPb.NewRpcNodeClient(conn)
	sk, _ := native.GetUserSK(1)
	payload1 := initPayload()
	payload1.Timestamp = timestamp
	payloadBytes, err := payload1.Marshal()
	if err != nil {
		panic(err)
	}
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "multiPayload",
			Value: payloadBytes,
		},
	}

	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_QUERY.String(),
		Parameters:   pairs,
	}

	resp := common.ProposalRequest(sk, &client, payload.TxType,
		CHAIN1, "", payload, nil)

	fmt.Println(resp)
}

func initPayload() *commonPb.Payload {
	wasmBin, _ := ioutil.ReadFile(WasmPath)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_DPOS_ERC20.String()),
		},
		{
			Key:   syscontract.MultiReq_SYS_METHOD.String(),
			Value: []byte(syscontract.ContractManageFunction_INIT_CONTRACT.String()),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_NAME.String(),
			Value: []byte(contractName),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_VERSION.String(),
			Value: []byte("1.0"),
		},
		{
			Key:   syscontract.InitContract_CONTRACT_BYTECODE.String(),
			Value: wasmBin,
		},
		{
			Key:   syscontract.InitContract_CONTRACT_RUNTIME_TYPE.String(),
			Value: []byte(runtimeType.String()),
		},
	}

	payload := &commonPb.Payload{
		TxType:       commonPb.TxType_INVOKE_CONTRACT,
		ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
		Method:       syscontract.MultiSignFunction_REQ.String(),
		Parameters:   pairs,
		TxId:         txId,
		ChainId:      CHAIN1,
	}
	return payload
}

//func InitRun()(apiPb.RpcNodeClient,crypto.PrivateKey){
//	var (
//		conn   *grpc.ClientConn
//		client apiPb.RpcNodeClient
//		sk3    crypto.PrivateKey
//		err    error
//	)
//	// init
//	{
//		conn, err = initGRPCConnect(true)
//		if err != nil {
//			fmt.Println(err)
//			return nil,nil
//		}
//		defer conn.Close()
//
//		client = apiPb.NewRpcNodeClient(conn)
//
//		file, err := ioutil.ReadFile(userKeyPath)
//		if err != nil {
//			panic(err)
//		}
//
//		sk3, err = asym.PrivateKeyFromPEM(file, nil)
//		if err != nil {
//			panic(err)
//		}
//	}
//	return client,sk3
//}
//
//
//func initGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
//	url := fmt.Sprintf("%s:%d", IP, Port)
//
//	if useTLS {
//		tlsClient := ca.CAClient{
//			ServerName: "chainmaker.org",
//			CaPaths:    caPaths,
//			CertFile:   userCrtPath,
//			KeyFile:    userKeyPath,
//		}
//
//		c, err := tlsClient.GetCredentialsByCA()
//		if err != nil {
//			log.Fatalf("GetTLSCredentialsByCA err: %v", err)
//			return nil, err
//		}
//		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
//	} else {
//		return grpc.Dial(url, grpc.WithInsecure())
//	}
//}

func AclSignOne(bytes []byte, index int) (*commonPb.EndorsementEntry, error) {
	signers := make([]protocol.SigningMember, 0)
	sk, member := GetAdminSK(index)
	signer := getSigner(sk, member)
	signers = append(signers, signer)
	return signWith(bytes, signer, crypto.CRYPTO_ALGO_SHA256)
}

// 获取admin的私钥
func GetAdminSK(index int) (crypto.PrivateKey, *acPb.Member) {
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
	//fmt.Println("node", numStr, "crt", string(file2))
	if err != nil {
		panic(err)
	}

	// 获取peerId
	peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
	fmt.Println("node", numStr, "peerId", peerId)

	// 构造Sender
	sender := &acPb.Member{
		OrgId:      fmt.Sprintf(OrgIdFormat, numStr),
		MemberInfo: file2,
		////IsFullCert: true,
	}

	return sk3, sender
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

func signWith(msg []byte, signer protocol.SigningMember, hashType string) (*commonPb.EndorsementEntry, error) {
	sig, err := signer.Sign(hashType, msg)
	if err != nil {
		return nil, err
	}
	signerSerial, err := signer.GetMember()
	if err != nil {
		return nil, err
	}
	return &commonPb.EndorsementEntry{
		Signer:    signerSerial,
		Signature: sig,
	}, nil
}
