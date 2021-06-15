/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"log"

	sdk "chainmaker.org/chainmaker-sdk-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

	adminKeyFilePaths  string
	adminCrtFilePaths  string
	clientKeyFilePaths string
	clientCrtFilePaths string

	blockInterval  int
	nodeOrgId      string
	nodeIdOld      string
	nodeId         string
	trustRootOrgId string
	trustRootPaths []string
	certFilePaths  string
	certCrlPath    string

	trustMemberOrgId    string
	trustMemberInfoPath string
	trustMemberRole     string
	trustMemberNodeId   string
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
	flagAdminKeyFilePaths      = "admin-key-file-paths"
	flagAdminCrtFilePaths      = "admin-crt-file-paths"
	flagClientKeyFilePaths     = "client-key-file-paths"
	flagClientCrtFilePaths     = "client-crt-file-paths"
	flagTimeout                = "timeout"
	flagBlockInterval          = "block-interval"
	flagNodeOrgId              = "node-org-id"
	flagNodeIdOld              = "node-id-old"
	flagNodeId                 = "node-id"
	flagTrustRootOrgId         = "trust-root-org-id"
	flagTrustRootCrtPath       = "trust-root-path"
	flagTrustMemberOrgId       = "trust-member-org-id"
	flagTrustMemberCrtPath     = "trust-member-path"
	flagTrustMemberRole        = "trust-member-role"
	flagTrustMemberNodeId      = "trust-member-node-id"
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
	flags.StringVar(&adminKeyFilePaths, flagAdminKeyFilePaths, "", "specify admin key file paths, use ',' to separate")
	flags.StringVar(&adminCrtFilePaths, flagAdminCrtFilePaths, "", "specify admin cert file paths, use ',' to separate")
	flags.StringVar(&clientKeyFilePaths, flagClientKeyFilePaths, "", "specify client key file paths, use ',' to separate")
	flags.StringVar(&clientCrtFilePaths, flagClientCrtFilePaths, "", "specify client cert file paths, use ',' to separate")

	// 链配置
	flags.IntVar(&blockInterval, flagBlockInterval, 2000, "block interval timeout in milliseconds, default 2000ms")

	flags.StringVar(&nodeOrgId, flagNodeOrgId, "", "specify node org id")
	flags.StringVar(&nodeIdOld, flagNodeIdOld, "", "specify old node id")
	flags.StringVar(&nodeId, flagNodeId, "", "specify node id(which will be added or update to")

	flags.StringVar(&trustRootOrgId, flagTrustRootOrgId, "", "specify the ca org id")
	flags.StringSliceVar(&trustRootPaths, flagTrustRootCrtPath, nil, "specify the ca file path")

	flags.StringVar(&trustMemberOrgId, flagTrustMemberOrgId, "", "specify the ca org id")
	flags.StringVar(&trustMemberInfoPath, flagTrustMemberCrtPath, "", "specify the ca file path")
	flags.StringVar(&trustMemberRole, flagTrustMemberRole, "", "specify trust member role")
	flags.StringVar(&trustMemberNodeId, flagTrustMemberNodeId, "", "specify trust member node id")

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

// 创建ChainClient（使用配置文件）
func createClientWithConfig() (*sdk.ChainClient, error) {
	chainClient, err := sdk.NewChainClient(sdk.WithConfPath(sdkConfPath), sdk.WithUserKeyFilePath(clientKeyFilePaths),
		sdk.WithUserCrtFilePath(clientCrtFilePaths), sdk.WithChainClientOrgId(orgId), sdk.WithChainClientChainId(chainId))
	if err != nil {
		return nil, err
	}

	//启用证书压缩（开启证书压缩可以减小交易包大小，提升处理性能）
	if enableCertHash {
		err = chainClient.EnableCertHash()
		if err != nil {
			log.Fatal(err)
		}
	}

	return chainClient, nil
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
		flagClientCrtFilePaths, flagClientKeyFilePaths,
	})

	cmd.MarkFlagRequired(flagSdkConfPath)

	return cmd
}

func getChainMakerServerVersion() error {
	client, err := createClientWithConfig()
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
