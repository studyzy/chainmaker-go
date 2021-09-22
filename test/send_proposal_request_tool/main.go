/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"chainmaker.org/chainmaker-go/accesscontrol"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	evm "chainmaker.org/chainmaker/common/v2/evmutils"
	"chainmaker.org/chainmaker/common/v2/helper"
	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	discoveryPb "chainmaker.org/chainmaker/pb-go/v2/discovery"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ip                  string
	port                int
	chainId             string
	orgId               string
	caPaths             []string
	caPathsString       string
	userCrtPath         string
	userKeyPath         string
	wasmPath            string
	contractName        string
	contractNameByte    []byte
	contractVersion     []byte
	contractRuntimeType []byte

	useShortCrt bool
	hashAlgo    string
	certPath    string

	resp   *commonPb.TxResponse
	sk3    crypto.PrivateKey
	client apiPb.RpcNodeClient

	seq           uint64 // chainConfig 的序列号
	orgIds        string // 组织列表(多个用逗号隔开)
	adminSignKeys string // 管理员私钥列表(多个用逗号隔开)
	adminSignCrts string // 管理员证书列表(多个用逗号隔开)

	hibeLocalParams string
	localId         string
	hibePrvKey      string
	symKeyType      string
	hibeMsg         string

	hibePlaintext           string
	hibeReceiverIdsFilePath string
	hibeParamsFilePath      string

	abiPath     string
	initParams  string
	prettyJson  bool
	OrgIdFormat = "wx-org%s.chainmaker.org"
	prePathFmt  = certPathPrefix + "/crypto-config/wx-org%s.chainmaker.org/user/admin1/"
)

const CHAIN1 = "chain1"
const certPathPrefix = "../../config"

type Result struct {
	Code                  commonPb.TxStatusCode           `json:"code"`
	Message               string                          `json:"message,omitempty"`
	ContractResultCode    uint32                          `json:"contractResultCode"`
	ContractResultMessage string                          `json:"contractResultMessage,omitempty"`
	ContractQueryResult   string                          `json:"contractQueryResult"`
	TxId                  string                          `json:"txId,omitempty"`
	BlockInfo             *commonPb.BlockInfo             `json:"blockInfo,omitempty"`
	TransactionInfo       *commonPb.TransactionInfo       `json:"transactionInfo,omitempty"`
	ChainInfo             *discoveryPb.ChainInfo          `json:"chainInfo,omitempty"`
	ChainList             *discoveryPb.ChainList          `json:"chainList,omitempty"`
	ContractInfo          *commonPb.Contract              `json:"contractInfo,omitempty"`
	MultiSignInfo         *commonPb.MultiSignInfo         `json:"multiSignInfo,omitempty"`
	PayloadHash           string                          `json:"payloadHash,omitempty"`
	ShortCert             string                          `json:"shortCert,omitempty"`
	GovernanceInfo        *consensusPb.GovernanceContract `json:"governanceInfo,omitempty"`
	HibeExecMsg           string                          `json:"hibe_exec_msg,omitempty"`
	CertAddress           *evm.Address                    `json:"certAddress,omitempty"`
}

func (result *Result) ToJsonString() string {
	rjson, _ := json.Marshal(result)
	if prettyJson {
		rjson = pretty.Color(rjson, nil)
	}
	return string(rjson)
}

type SimpleRPCResult struct {
	Code    commonPb.TxStatusCode `json:"code"`
	Message string                `json:"message,omitempty"`
}

func (result *SimpleRPCResult) ToJsonString() string {
	rjson, _ := json.Marshal(result)
	if prettyJson {
		rjson = pretty.Color(rjson, nil)
	}
	return string(rjson)
}

func main() {
	var conn *grpc.ClientConn
	var err error
	mainCmd := &cobra.Command{
		Use: "tool",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			caPaths = strings.Split(caPathsString, ",")

			conn, err = initGRPCConnect(useTLS)
			if err != nil {
				panic(err)
			}
			client = apiPb.NewRpcNodeClient(conn)

			file, err := ioutil.ReadFile(userKeyPath)
			if err != nil {
				panic(err)
			}

			sk3, err = asym.PrivateKeyFromPEM(file, nil)
			if err != nil {
				panic(err)
			}
		},
	}

	mainFlags := mainCmd.PersistentFlags()
	mainFlags.StringVarP(&ip, "ip", "i", "localhost", "specify ip")
	mainFlags.IntVarP(&port, "port", "p", 12301, "specify port")
	mainFlags.StringVarP(&userKeyPath, "user-key", "k", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key", "specify user key path")
	mainFlags.StringVarP(&userCrtPath, "user-crt", "c", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt", "specify user crt path")
	mainFlags.StringVarP(&caPathsString, "ca-path", "P", "../../config/crypto-config/wx-org1.chainmaker.org/ca,../../config/crypto-config/wx-org2.chainmaker.org/ca", "specify ca path")
	mainFlags.BoolVarP(&useTLS, "use-tls", "t", false, "specify whether use tls")
	mainFlags.StringVarP(&chainId, "chain-id", "C", "chain1", "specify chain id")
	mainFlags.StringVarP(&orgId, "org-id", "O", "wx-org1", "specify org id")
	mainFlags.StringVarP(&contractName, "contract-name", "n", "contract1", "specify contract name")
	mainFlags.StringVar(&orgIds, "org-ids", "wx-org1,wx-org2,wx-org3,wx-org4", "orgIds of admin")
	//mainFlags.StringVar(&orgIds, "org-ids", "wx-org1,wx-org2,wx-org3,wx-org4", "orgIds of admin")
	mainFlags.StringVar(&adminSignKeys, "admin-sign-keys", "../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key", "adminSignKeys of admin")
	mainFlags.StringVar(&adminSignCrts, "admin-sign-crts", "../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt", "adminSignCrts of admin")
	mainFlags.Uint64Var(&seq, "seq", 1, "sequence of chainConfig")
	mainFlags.IntVar(&requestTimeout, "requestTimeout", 5, "specify request timeout(unit: s)")

	mainFlags.BoolVar(&useShortCrt, "use-short-crt", false, "use compressed certificate in transactions")
	mainFlags.StringVar(&hashAlgo, "hash-algorithm", "SHA256", "hash algorithm set in chain configuration")
	mainFlags.BoolVar(&prettyJson, "pretty", false, "specify whether pretty json result")

	mainCmd.AddCommand(CreateContractCMD())
	mainCmd.AddCommand(UpgradeContractCMD())
	mainCmd.AddCommand(InvokeCMD())
	mainCmd.AddCommand(QueryCMD())
	mainCmd.AddCommand(GetTxByTxIdCMD())
	mainCmd.AddCommand(GetBlockByHeightCMD())
	mainCmd.AddCommand(GetBlockWithRWSetsByHeightCMD())
	mainCmd.AddCommand(GetBlockByHashCMD())
	mainCmd.AddCommand(GetBlockWithRWSetsByHashCMD())
	mainCmd.AddCommand(GetBlockByTxIdCMD())
	mainCmd.AddCommand(GetLastConfigBlockCMD())
	mainCmd.AddCommand(GetLastBlockCMD())
	mainCmd.AddCommand(GetChainInfoCMD())
	mainCmd.AddCommand(GetNodeChainListCMD())
	mainCmd.AddCommand(GetContractInfoCMD())
	mainCmd.AddCommand(UpdateDebugConfigCMD())

	mainCmd.AddCommand(ChainConfigGetChainConfigCMD())
	mainCmd.AddCommand(ChainConfigGetChainConfigByBlockHeightCMD())

	mainCmd.AddCommand(ChainConfigCoreUpdateCMD())
	mainCmd.AddCommand(ChainConfigBlockUpdateCMD())

	mainCmd.AddCommand(ChainConfigTrustRootAddCMD())
	mainCmd.AddCommand(ChainConfigTrustRootUpdateCMD())
	mainCmd.AddCommand(ChainConfigTrustRootDeleteCMD())

	mainCmd.AddCommand(ChainConfigNodeAddrAddCMD())
	mainCmd.AddCommand(ChainConfigNodeAddrUpdateCMD())
	mainCmd.AddCommand(ChainConfigNodeAddrDeleteCMD())

	mainCmd.AddCommand(ChainConfigNodeOrgAddCMD())
	mainCmd.AddCommand(ChainConfigNodeOrgUpdateCMD())
	mainCmd.AddCommand(ChainConfigNodeOrgDeleteCMD())

	mainCmd.AddCommand(ChainConfigConsensusExtAddCMD())
	mainCmd.AddCommand(ChainConfigConsensusExtUpdateCMD())
	mainCmd.AddCommand(ChainConfigConsensusExtDeleteCMD())

	mainCmd.AddCommand(ChainConfigPermissionAddCMD())
	mainCmd.AddCommand(ChainConfigPermissionUpdateCMD())
	mainCmd.AddCommand(ChainConfigPermissionDeleteCMD())

	mainCmd.AddCommand(CertManageAddCMD())
	mainCmd.AddCommand(CertManageDeleteCMD())
	mainCmd.AddCommand(CertManageQueryCMD())
	mainCmd.AddCommand(CertManageFrozenCMD())
	mainCmd.AddCommand(CertManageUnfrozenCMD())
	mainCmd.AddCommand(CertManageRevocationCMD())

	mainCmd.AddCommand(ParallelCMD())
	mainCmd.AddCommand(RefreshLogLevelsCMD())

	mainCmd.AddCommand(GetShortCertBase64())

	mainCmd.AddCommand(FreezeContractCMD())
	mainCmd.AddCommand(UnfreezeContractCMD())
	mainCmd.AddCommand(RevokeContractCMD())

	mainCmd.AddCommand(ChainConfigGetGovernanceContractCMD())

	mainCmd.AddCommand(MultiSignReqCMD())
	mainCmd.AddCommand(MultiSignVoteCMD())
	mainCmd.AddCommand(MultiSignQueryCMD())

	mainCmd.AddCommand(HibeDecryptCMD())
	mainCmd.AddCommand(HibeEncryptCMD())
	mainCmd.AddCommand(CertToAddressCMD())
	mainCmd.AddCommand(ContractNameToAddressCMD())

	//private contract
	mainCmd.AddCommand(SaveCertCMD())
	mainCmd.AddCommand(SaveDirCMD())
	mainCmd.AddCommand(GetContractCMD())
	mainCmd.AddCommand(SaveDataCMD())
	mainCmd.AddCommand(GetDataCMD())
	mainCmd.AddCommand(GetCertCMD())
	mainCmd.AddCommand(GetDirCMD())

	//generate hash code
	mainCmd.AddCommand(GenerateHashCMD())

	//paillier
	mainCmd.AddCommand(PaillierCMD())

	//DPoS.erc20
	mainCmd.AddCommand(ERC20Mint())
	mainCmd.AddCommand(ERC20Transfer())
	mainCmd.AddCommand(ERC20BalanceOf())
	mainCmd.AddCommand(ERC20Owner())
	mainCmd.AddCommand(ERC20Decimals())
	mainCmd.AddCommand(ERC20Cert2Address())
	mainCmd.AddCommand(ERC20Total())

	//DPoS.Stake
	mainCmd.AddCommand(StakeGetAllCandidates())
	mainCmd.AddCommand(StakeDelegate())
	mainCmd.AddCommand(StakeUnDelegate())
	mainCmd.AddCommand(StakeSetNodeID())
	mainCmd.AddCommand(StakeGetNodeID())
	mainCmd.AddCommand(StakeGetEpochByID())
	mainCmd.AddCommand(StakeGetSystemAddr())
	mainCmd.AddCommand(StakeGetLatestEpoch())
	mainCmd.AddCommand(StakeGetEpochBlockNumber())
	mainCmd.AddCommand(StakeGetMinSelfDelegation())
	mainCmd.AddCommand(StakeGetEpochValidatorNumber())
	mainCmd.AddCommand(StakeGetUnbondingEpochNumber())
	mainCmd.AddCommand(StakeGetDelegationsByAddress())
	mainCmd.AddCommand(StakeGetDelegationByValidator())

	// bulletproofs
	mainCmd.AddCommand(BulletproofsCMD())

	mainCmd.Execute()

	if conn != nil {
		conn.Close()
	}
}

func proposalRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, payload *commonPb.Payload) (*commonPb.TxResponse, error) {
	return proposalRequestWithMultiSign(sk3, client, payload, nil)
}
func proposalRequestWithMultiSign(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, payload *commonPb.Payload, endorsers []*commonPb.EndorsementEntry) (*commonPb.TxResponse, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(time.Duration(requestTimeout)*time.Second)))
	defer cancel()

	if txId == "" {
		txId = utils.GetRandTxId()
	}

	file, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		return nil, err
	}

	// 构造Sender
	senderFull := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file,
		//IsFullCert: true,
	}

	var sender *acPb.Member
	if useShortCrt {
		certId, err := utils.GetCertificateId(senderFull.MemberInfo, hashAlgo)
		if err != nil {
			return nil, fmt.Errorf("fail to compute the identity for certificate [%v]", err)
		}
		sender = &acPb.Member{
			OrgId:      senderFull.OrgId,
			MemberInfo: certId,
			MemberType: acPb.MemberType_CERT_HASH,
		}
	} else {
		sender = senderFull
	}

	// 构造TxRequest
	req := &commonPb.TxRequest{
		Payload: payload,
		Sender:  &commonPb.EndorsementEntry{Signer: sender},
	}
	if len(endorsers) > 0 {
		req.Endorsers = endorsers
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, err
	}

	signer, err := getSigner(sk3, senderFull)
	if err != nil {
		return nil, err
	}
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	if err != nil {
		return nil, err
	}

	req.Sender.Signature = signBytes

	result, err := client.SendRequest(ctx, req)
	return processResp(result, err)
}

func processResp(result *commonPb.TxResponse, err error) (*commonPb.TxResponse, error) {
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return nil, fmt.Errorf("client.call err: deadline\n")
		}
		return nil, fmt.Errorf("client.call err: %v\n", err)
	}
	return result, nil
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

// 配置更新
func configUpdateRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, msg *InvokerMsg) (*commonPb.TxResponse, string, error) {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(5*time.Second)))
	defer cancel()

	txId = utils.GetRandTxId()
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
	var sender = senderFull
	if useShortCrt {
		id, err := utils.GetCertificateId(file, hashAlgo)
		if err != nil {
			return nil, "", err
		}
		sender = &acPb.Member{
			OrgId:      orgId,
			MemberInfo: id,
			MemberType: acPb.MemberType_CERT_HASH,
		}
	}

	// 构造Header
	payload := &commonPb.Payload{
		ChainId: chainId,
		//Sender:         sender,
		TxType:         msg.txType,
		TxId:           txId,
		Timestamp:      time.Now().Unix(),
		ExpirationTime: 0,

		ContractName: msg.contractName,
		Method:       msg.method,
		Parameters:   msg.pairs,
		Sequence:     msg.oldSeq + 1,
	}

	entries, err := aclSign(*payload, orgIds, adminSignKeys, adminSignCrts)
	if err != nil {
		panic(err)
	}
	//payload.Endorsement = entries

	//payloadBytes, err := proto.Marshal(payload)
	//if err != nil {
	//	return nil, "", err
	//}
	req := &commonPb.TxRequest{
		Payload:   payload,
		Sender:    &commonPb.EndorsementEntry{Signer: sender},
		Endorsers: entries,
	}

	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, "", err
	}

	signer, err := getSigner(sk3, senderFull)
	if err != nil {
		return nil, "", err
	}
	signBytes, err := signer.Sign("SHA256", rawTxBytes)
	if err != nil {
		return nil, "", err
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

// 签名【需要一一对应】
// orgIds 组织列表(多个用逗号隔开)
// adminSignKeys 管理员私钥列表(多个用逗号隔开)
// adminSignCrts 管理员证书列表(多个用逗号隔开)
func aclSign(msg commonPb.Payload, orgIds, adminSignKeys, adminSignCrts string) ([]*commonPb.EndorsementEntry, error) {
	//msg.Endorsement = nil
	bytes, _ := proto.Marshal(&msg)

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
			panic(err)
		}
		sk3, err = asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			panic(err)
		}

		file2, err := ioutil.ReadFile(adminSignCrtArray[i])
		fmt.Println("node", i, "crt", string(file2))
		if err != nil {
			panic(err)
		}

		// 获取peerId
		peerId, err := helper.GetLibp2pPeerIdFromCert(file2)
		fmt.Println("node", i, "peerId", peerId)

		// 构造Sender
		sender1 := &acPb.Member{
			OrgId:      orgIdArray[i],
			MemberInfo: file2,
			//IsFullCert: true,
		}

		signer, err := getSigner(sk3, sender1)
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}

	endorsements, err := accesscontrol.MockSignWithMultipleNodes(bytes, signers, hashAlgo)
	if err != nil {
		return nil, err
	}
	fmt.Printf("endorsements:\n%v\n", endorsements)
	return endorsements, nil
}

func aclSignOne(bytes []byte, orgId, adminSignKey, adminSignCrt string) (*commonPb.EndorsementEntry, error) {
	file, err := ioutil.ReadFile(adminSignKey)
	if err != nil {
		panic(err)
	}
	sk3, err = asym.PrivateKeyFromPEM(file, nil)
	if err != nil {
		panic(err)
	}

	file2, err := ioutil.ReadFile(adminSignCrt)
	if err != nil {
		panic(err)
	}

	// 构造Sender
	sender1 := &acPb.Member{
		OrgId:      orgId,
		MemberInfo: file2,
		//IsFullCert: true,
	}

	signer, err := getSigner(sk3, sender1)
	if err != nil {
		return nil, err
	}

	return signWith(bytes, signer, "SHA256")
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
