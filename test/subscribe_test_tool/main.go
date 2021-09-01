/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	apipb "chainmaker.org/chainmaker/pb-go/v2/api"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/sdk-go/v2/utils"
)

var (
	ip           string
	port         int
	chainId      string
	orgId        string
	caPaths      []string
	userCrtPath  string
	userKeyPath  string
	useTLS       bool
	dataFile     string
	startBlock   int64
	endBlock     int64
	withRwSet    bool
	txType       int32
	txIds        string
	topic        string
	contractName string
	onlyHeader   bool

	conn   *grpc.ClientConn
	client apipb.RpcNodeClient
	sk3    crypto.PrivateKey
)

const rpcClientMaxReceiveMessageSize = 1024 * 1024 * 16

func main() {
	var err error
	mainCmd := &cobra.Command{
		Use: "subscribe",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			conn, err = initGRPCConnect(true)
			if err != nil {
				panic(err)
			}

			client = apipb.NewRpcNodeClient(conn)

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

	mainCmd.AddCommand(SubscribeBlockCMD())
	mainCmd.AddCommand(SubscribeTxCMD())
	mainCmd.AddCommand(SubscribeEventCMD())

	mainFlags := mainCmd.PersistentFlags()
	mainFlags.StringVarP(&ip, "ip", "i", "localhost", "specify ip")
	mainFlags.IntVarP(&port, "port", "p", 12301, "specify port")
	mainFlags.StringVarP(&userKeyPath, "userkey", "k", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key", "specify user key path")
	mainFlags.StringVarP(&userCrtPath, "user-crt", "c", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt", "specify user crt path")
	mainFlags.StringArrayVarP(&caPaths, "ca-path", "P", []string{"../../config/crypto-config/wx-org1.chainmaker.org/ca"}, "specify ca path")
	mainFlags.BoolVarP(&useTLS, "use-tls", "t", false, "specify whether use tls")
	mainFlags.StringVarP(&chainId, "chain-id", "C", "chain1", "specify chain id")
	mainFlags.StringVarP(&orgId, "org-id", "O", "wx-org1.chainmaker.org", "specify org id")
	mainFlags.StringVarP(&dataFile, "data-file", "f", "data.txt", "specify the data file to write blocks or tx")
	mainFlags.Int64VarP(&startBlock, "start-block", "s", -1, "specify the start block height to receive from, -1 means to receive until you stop the program")
	mainFlags.Int64VarP(&endBlock, "end-block", "e", -1, "specify the end block height to receive to, -1 means to receive until you stop the program")
	mainFlags.BoolVarP(&withRwSet, "withRWSet", "S", false, "specify withRWSet, true or false")
	mainFlags.Int32VarP(&txType, "tx-type", "T", -1, "specify transaction type you with to receive, -1 means all, other value from 0 to 7")
	mainFlags.StringVarP(&txIds, "tx-ids", "I", "", "specify the transaction ids, separated by comma, NOTICE: don't add space between ids")
	mainFlags.StringVarP(&topic, "topic", "", "topic_vx", "specify the contract event topic")
	mainFlags.StringVarP(&contractName, "contract-name", "", "claim001", "specify the contract name")
	mainFlags.BoolVarP(&onlyHeader, "only-header", "H", false, "the results of blocks only contains Header or FUll Data when subscribe block")

	if mainCmd.Execute() != nil {
		return
	}
	if conn != nil {
		conn.Close()
	}
}

func initGRPCConnect(useTLS bool) (*grpc.ClientConn, error) {
	url := fmt.Sprintf("%s:%d", ip, port)

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
		return grpc.Dial(url, grpc.WithTransportCredentials(*c), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(rpcClientMaxReceiveMessageSize)))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}

func createPayload(chainId, txId string, txType common.TxType, contractName, method string,
	kvs []*common.KeyValuePair, seq uint64) *common.Payload {
	if txId == "" {
		txId = utils.GetRandTxId()
	}

	payload := utils.NewPayload(
		utils.WithChainId(chainId),
		utils.WithTxType(txType),
		utils.WithTxId(txId),
		utils.WithTimestamp(time.Now().Unix()),
		utils.WithContractName(contractName),
		utils.WithMethod(method),
		utils.WithParameters(kvs),
		utils.WithSequence(seq),
	)

	return payload
}

func generateTxRequest(payload *common.Payload,
	endorsers []*common.EndorsementEntry) (*common.TxRequest, error) {
	userCrtBytes, err := ioutil.ReadFile(userCrtPath)
	if err != nil {
		panic(err)
	}

	// 构造Sender
	signer := &accesscontrol.Member{
		OrgId:      orgId,
		MemberInfo: userCrtBytes,
		MemberType: accesscontrol.MemberType_CERT,
	}

	req := &common.TxRequest{
		Payload: payload,
		Sender: &common.EndorsementEntry{
			Signer:    signer,
			Signature: nil,
		},
		Endorsers: endorsers,
	}

	userCrt, err := utils.ParseCert(userCrtBytes)
	if err != nil {
		return nil, err
	}
	signBytes, err := utils.SignPayload(sk3, userCrt, payload)
	if err != nil {
		return nil, fmt.Errorf("SignPayload failed, %s", err)
	}

	req.Sender.Signature = signBytes

	return req, nil
}

func subscribe(ctx context.Context, payload *common.Payload) (<-chan interface{}, error) {

	req, err := generateTxRequest(payload, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Subscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan interface{})
	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var result *common.SubscribeResult
				result, err = resp.Recv()
				if err == io.EOF {
					return
				}

				if err != nil {
					return
				}

				var ret interface{}
				switch payload.Method {
				case syscontract.SubscribeFunction_SUBSCRIBE_BLOCK.String():
					blockInfo := &common.BlockInfo{}
					if err = proto.Unmarshal(result.Data, blockInfo); err == nil {
						ret = blockInfo
						break
					}

					blockHeader := &common.BlockHeader{}
					if err = proto.Unmarshal(result.Data, blockHeader); err == nil {
						ret = blockHeader
						break
					}
					close(c)
					return
				case syscontract.SubscribeFunction_SUBSCRIBE_TX.String():
					tx := &common.Transaction{}
					if err = proto.Unmarshal(result.Data, tx); err != nil {
						close(c)
						return
					}
					ret = tx
				case syscontract.SubscribeFunction_SUBSCRIBE_CONTRACT_EVENT.String():
					events := &common.ContractEventInfoList{}
					if err = proto.Unmarshal(result.Data, events); err != nil {
						close(c)
						return
					}
					for _, event := range events.ContractEvents {
						c <- event
					}
					continue

				default:
					ret = result.Data
				}

				c <- ret
			}
		}
	}()

	return c, err
}
