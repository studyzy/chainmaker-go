/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"chainmaker.org/chainmaker-go/common/crypto"
	"chainmaker.org/chainmaker-go/common/crypto/asym"
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	sdk "chainmaker.org/chainmaker-sdk-go"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-sdk-go/pb/protogo/common"
)

var (
	// 压测参数
	concurrency          int // 并发数
	totalCntPerGoroutine int // 每个并发协程请求数

	sdkConfPath string // SDK配置路径

	// 合约参数
	contractName   string
	version        string
	byteCodePath   string
	runtimeType    string
	timeout        int
	sendTimes      int
	method         string
	params         string
	orgId          string
	chainId        string
	syncResult     bool
	enableCertHash bool
	blockHeight    int64
	withRWSet      bool
	txId           string

	adminOrgIds       string
	adminKeyFilePaths string
	adminCrtFilePaths string

	userTlsKeyFilePath string
	userTlsCrtFilePath string

	blockInterval  int
	nodeOrgId      string
	nodeIdOld      string
	nodeId         string
	nodeIds        string
	trustRootOrgId string
	trustRootPath  string
	certFilePaths  string
	certCrlPath    string
)

const (
	flagConcurrency            = "concurrency"
	flagTotalCountPerGoroutine = "total-count-per-goroutine"
	flagSdkConfPath            = "sdk-conf-path"
	flagContractName           = "contract-name"
	flagVersion                = "version"
	flagMethod                 = "method"
	flagParams                 = "params"
	flagOrgId                  = "org-id"
	flagSyncResult             = "sync-result"
	flagEnableCertHash         = "enable-cert-hash"
	flagBlockHeight            = "block-height"
	flagWithRWSet              = "with-rw-set"
	flagTxId                   = "tx-id"
	flagByteCodePath           = "byte-code-path"
	flagRuntimeType            = "runtime-type"
	flagChainId                = "chain-id"
	flagSendTimes              = "send-times"
	flagAdminOrgIds            = "admin-org-ids"
	flagAdminKeyFilePaths      = "admin-key-file-paths"
	flagAdminCrtFilePaths      = "admin-crt-file-paths"
	flagUserTlsKeyFilePath     = "user-tlskey-file-path"
	flagUserTlsCrtFilePath     = "user-tlscrt-file-path"
	flagTimeout                = "timeout"
	flagBlockInterval          = "block-interval"
	flagNodeOrgId              = "node-org-id"
	flagNodeIdOld              = "node-id-old"
	flagNodeId                 = "node-id"
	flagNodeIds                = "node-ids"
	flagTrustRootOrgId         = "trust-root-org-id"
	flagTrustRootCrtPath       = "trust-root-path"
	flagCertFilePaths          = "cert-file-paths"
	flagCertCrlPath            = "cert-crl-path"
)

func ClientCMD() *cobra.Command {
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "client command",
		Long:  "client command",
	}

	clientCmd.AddCommand(contractCMD())
	clientCmd.AddCommand(chainConfigCMD())
	clientCmd.AddCommand(getChainMakerServerVersionCMD())
	clientCmd.AddCommand(certManageCMD())
	clientCmd.AddCommand(blockChainsCMD())

	return clientCmd
}

var flags *pflag.FlagSet

func init() {
	flags = &pflag.FlagSet{}

	// 压测参数
	flags.IntVarP(&concurrency, flagConcurrency, "c", 1, "specify concurrency count")
	flags.IntVarP(&totalCntPerGoroutine, flagTotalCountPerGoroutine, "t", 1, "specify total count per goroutine")

	// sdk配置路径
	flags.StringVar(&sdkConfPath, flagSdkConfPath, "", "specify sdk config path")

	// 用户合约
	flags.StringVar(&contractName, flagContractName, "", "specify user contract name, eg: counter-go-1")
	flags.StringVar(&version, flagVersion, "", "specify user contract version, eg: 1.0.0")
	flags.StringVar(&byteCodePath, flagByteCodePath, "", "specify user contract byte code path")
	flags.StringVar(&runtimeType, flagRuntimeType, "", "specify user contract runtime type, such as: "+
		"NATIVE | WASMER | WXVM | GASM | EVM | DOCKER_GO | DOCKER_JAVA")
	flags.StringVar(&chainId, flagChainId, "", "specify the chain id, such as: chain1, chain2 etc.")
	flags.IntVar(&sendTimes, flagSendTimes, 1, "specify SendTimes , default once")
	flags.IntVar(&timeout, flagTimeout, 10, "specify timeout in seconds, default 10s")
	flags.StringVar(&method, flagMethod, "", "specify invoke contract method")
	flags.StringVar(&params, flagParams, "", "specify invoke contract params, json format, such as: \"{\\\"key\\\":\\\"value\\\",\\\"key1\\\":\\\"value1\\\"}\"")
	flags.StringVar(&orgId, flagOrgId, "", "specify the orgId, such as wx-org1.chainmaker.com")
	flags.BoolVar(&syncResult, flagSyncResult, false, "whether wait the result of the transaction, default false")
	flags.BoolVar(&enableCertHash, flagEnableCertHash, true, "whether enable cert hash, default true")
	flags.BoolVar(&withRWSet, flagWithRWSet, true, "whether with RWSet, default true")
	flags.Int64Var(&blockHeight, flagBlockHeight, -1, "specify block height, default -1")
	flags.StringVar(&txId, flagTxId, "", "specify tx id")

	// Admin秘钥和证书列表
	//    - 使用逗号','分割
	//    - 列表中的key与crt需一一对应
	//    - 如果只有一对，将采用单签模式；如果有多对，将采用多签模式，第一对用于发起多签请求，其余的用于多签投票
	flags.StringVar(&adminOrgIds, flagAdminOrgIds, "", "specify admin org IDs, use ',' to separate")
	flags.StringVar(&adminKeyFilePaths, flagAdminKeyFilePaths, "", "specify admin key file paths, use ',' to separate")
	flags.StringVar(&adminCrtFilePaths, flagAdminCrtFilePaths, "", "specify admin cert file paths, use ',' to separate")

	flags.StringVar(&userTlsKeyFilePath, flagUserTlsKeyFilePath, "", "specify user tls key file path for chainclient tls connection")
	flags.StringVar(&userTlsCrtFilePath, flagUserTlsCrtFilePath, "", "specify user tls cert file path for chainclient tls connection")

	// 链配置
	flags.IntVar(&blockInterval, flagBlockInterval, 2000, "block interval timeout in milliseconds, default 2000ms")

	flags.StringVar(&nodeOrgId, flagNodeOrgId, "", "specify node org id")
	flags.StringVar(&nodeIdOld, flagNodeIdOld, "", "specify old node id")
	flags.StringVar(&nodeId, flagNodeId, "", "specify node id(which will be added or update to")
	flags.StringVar(&nodeIds, flagNodeIds, "", "specify node ids(which will be added or update to")

	flags.StringVar(&trustRootOrgId, flagTrustRootOrgId, "", "specify the ca org id")
	flags.StringVar(&trustRootPath, flagTrustRootCrtPath, "", "specify the ca file path")
	// 证书管理
	flags.StringVar(&certFilePaths, flagCertFilePaths, "", "specify cert file paths, use ',' to separate")
	flags.StringVar(&certCrlPath, flagCertCrlPath, "", "specify cert crl path")
}

func attachFlags(cmd *cobra.Command, names []string) {
	cmdFlags := cmd.Flags()
	for _, name := range names {
		if flag := flags.Lookup(name); flag != nil {
			cmdFlags.AddFlag(flag)
		}
	}
}

func createAdminWithConfig(adminKeyFilePath, adminCrtFilePath string) (*sdk.ChainClient, error) {
	chainClient, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfPath),
		sdk.WithUserKeyFilePath(adminKeyFilePath),
		sdk.WithUserCrtFilePath(adminCrtFilePath),
	)
	if err != nil {
		return nil, err
	}

	//启用证书压缩（开启证书压缩可以减小交易包大小，提升处理性能）
	err = chainClient.EnableCertHash()
	if err != nil {
		log.Fatal(err)
	}

	return chainClient, nil
}

func getChainMakerServerVersionCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmversion",
		Short: "get chainmaker server version",
		Long:  "get chainmaker server version",
		RunE: func(_ *cobra.Command, _ []string) error {
			return getChainMakerServerVersion()
		},
	}

	attachFlags(cmd, []string{
		flagSdkConfPath, flagOrgId,
		flagUserTlsCrtFilePath, flagUserTlsKeyFilePath,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func getChainMakerServerVersion() error {
	client, err := util.CreateChainClient(sdkConfPath, chainId, orgId, userTlsCrtFilePath, userTlsKeyFilePath)
	if err != nil {
		return fmt.Errorf("create user client failed, %s", err.Error())
	}
	defer client.Stop()
	version, err := client.GetChainMakerServerVersion()
	if err != nil {
		return fmt.Errorf("get chainmaker server version failed, %s", err.Error())
	}
	fmt.Printf("current chainmaker server version: %s \n", version)
	return nil
}

func signChainConfigPayload(payloadBytes, userCrtBytes []byte, privateKey crypto.PrivateKey, userCrt *bcx509.Certificate, orgId string) ([]byte, error) {
	payload := &common.SystemContractPayload{}
	if err := proto.Unmarshal(payloadBytes, payload); err != nil {
		return nil, fmt.Errorf("unmarshal config update payload failed, %s", err)
	}

	signBytes, err := signTx(privateKey, userCrt, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("SignPayload failed, %s", err)
	}

	sender := &accesscontrol.SerializedMember{
		OrgId:      orgId,
		MemberInfo: userCrtBytes,
		IsFullCert: true,
	}

	entry := &common.EndorsementEntry{
		Signer:    sender,
		Signature: signBytes,
	}

	payload.Endorsement = []*common.EndorsementEntry{
		entry,
	}

	signedPayloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal config update sigend payload failed, %s", err)
	}

	return signedPayloadBytes, nil
}

func signTx(privateKey crypto.PrivateKey, cert *bcx509.Certificate, msg []byte) ([]byte, error) {
	var opts crypto.SignOpts
	hashalgo, err := bcx509.GetHashFromSignatureAlgorithm(cert.SignatureAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("invalid algorithm: %v", err)
	}

	opts.Hash = hashalgo
	opts.UID = crypto.CRYPTO_DEFAULT_UID

	return privateKey.SignWithOpts(msg, &opts)
}

func dealUserCrt(userCrtFilePath string) (userCrtBytes []byte, userCrt *bcx509.Certificate, err error) {

	// 读取用户证书
	userCrtBytes, err = ioutil.ReadFile(userCrtFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read user crt file failed, %s", err)
	}

	// 将证书转换为证书对象
	userCrt, err = sdk.ParseCert(userCrtBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("ParseCert failed, %s", err)
	}
	return
}

func dealUserKey(userKeyFilePath string) (userKeyBytes []byte, privateKey crypto.PrivateKey, err error) {

	// 从私钥文件读取用户私钥，转换为privateKey对象
	userKeyBytes, err = ioutil.ReadFile(userKeyFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read user key file failed, %s", err)
	}

	privateKey, err = asym.PrivateKeyFromPEM(userKeyBytes, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("parse user key file to privateKey obj failed, %s", err)
	}
	return
}
