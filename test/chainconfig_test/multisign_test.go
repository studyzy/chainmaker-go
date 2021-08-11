package native

import (
	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker-go/test/common"
	"chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/asym"
	acPb "chainmaker.org/chainmaker/pb-go/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/api"
	commonPb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var (
	WasmUpgradePath = ""
	contractName    = ""
	runtimeType     = commonPb.RuntimeType_WASMER
	multiOrgId      = "wx-org1.chainmaker.org"
	multiOrg3Id     = "wx-org3.chainmaker.org"
	txId            = ""
	timestamp       int64
	timestampBak    int64
)

const (
	CHAIN1 = "chain1"
	orgId  = "wx-org1.chainmaker.org"
)

func testMultiSign(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, contractName, chainId string) []byte {
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Println("============================================testMultiSign===============================================")
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")

	payload := initPayload()

	resp := common.ProposalMultiRequest(sk3, client, payload.TxType,
		chainId, payload.TxId, payload, []int{1}, timestamp)

	fmt.Println("testMultiSign timestamp", timestamp)
	fmt.Println(resp)
	return nil
}

func testMultiSignVote(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, contractName, chainId string) []byte {
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Println("==========================================testMultiSignVote ============================================")
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")

	payload1 := initPayload()
	payload1.Timestamp = timestamp
	payloadBytes, err := payload1.Marshal()
	fmt.Printf("testMultiSignVote1 payload md5 is %x \n", md5.Sum(payloadBytes))
	if err != nil {
		panic(err)
	}
	var (
		certPathPrefix = "../../config"
		admin1KeyPath  = certPathPrefix + "/crypto-config/" + multiOrgId + "/user/admin1/admin1.tls.key"
		admin1CrtPath  = certPathPrefix + "/crypto-config/" + multiOrgId + "/user/admin1/admin1.tls.crt"
	)

	var msviByte []byte
	{
		admin1File, err := ioutil.ReadFile(admin1CrtPath)
		if err != nil {
			panic(err)
		}
		fadminKeyFile, err := ioutil.ReadFile(admin1KeyPath)
		if err != nil {
			panic(err)
		}
		admin1 := &acPb.Member{
			OrgId:      multiOrgId,
			MemberInfo: admin1File,
		}
		skAdmin1, err := asym.PrivateKeyFromPEM(fadminKeyFile, nil)
		signerAdmin1 := GetSigner(skAdmin1, admin1)
		signerAdmin1Bytes, err := signerAdmin1.Sign("SHA256", payloadBytes) //modify
		//signerAdmin1Bytes, err := signerAdmin1.Sign("SM3", payloadBytes) //modify
		if err != nil {
			log.Fatalf("sign failed, %s", err.Error())
			os.Exit(0)
		}

		ee := &commonPb.EndorsementEntry{
			Signer:    admin1,
			Signature: signerAdmin1Bytes,
		}

		msvi := &syscontract.MultiSignVoteInfo{
			Vote:        syscontract.VoteStatus_AGREE,
			Endorsement: ee,
		}
		msviByte, _ = msvi.Marshal()

	}
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

	resp := common.ProposalMultiRequest(sk3, client, payload.TxType,
		chainId, "", payload, nil, time.Now().Unix())

	fmt.Println(resp)
	return nil
}

func testMultiSignQuery(sk3 crypto.PrivateKey, client *apiPb.RpcNodeClient, contractName, chainId string) []byte {
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")
	fmt.Println("==========================================testMultiSignQuery ===========================================")
	fmt.Println("========================================================================================================")
	fmt.Println("========================================================================================================")

	payload1 := initPayloadTimestamp()
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

	resp := common.ProposalRequest(sk3, client, payload.TxType,
		chainId, "", payload, nil)

	fmt.Println(resp)
	return nil
}

func initPayload() *commonPb.Payload {
	wasmBin, _ := ioutil.ReadFile(WasmPath)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   syscontract.MultiReq_SYS_CONTRACT_NAME.String(),
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
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

func initPayloadTimestamp() *commonPb.Payload {
	wasmBin, _ := ioutil.ReadFile(WasmPath)
	pairs := []*commonPb.KeyValuePair{
		{
			Key:   "sysContractName",
			Value: []byte(syscontract.SystemContract_CONTRACT_MANAGE.String()),
		},
		{
			Key:   "sysMethod",
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
			Key: syscontract.InitContract_CONTRACT_BYTECODE.String(),
			//Value: nil,
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
		Timestamp:    timestampBak,
		TxId:         txId,
		ChainId:      CHAIN1,
	}
	return payload
}

func GetSigner(sk3 crypto.PrivateKey, sender *acPb.Member) protocol.SigningMember {
	skPEM, err := sk3.String()
	if err != nil {
		log.Fatalf("get sk PEM failed, %s", err.Error())
	}
	//fmt.Printf("skPEM: %s\n", skPEM)

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
