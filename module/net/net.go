/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"chainmaker.org/chainmaker-go/net/p2p"
	netPb "chainmaker.org/chainmaker/pb-go/v2/net"
	api "chainmaker.org/chainmaker/protocol/v2"
)

// Net is local net interface.
type Net interface {
	// GetNodeUid is the unique id of the node.
	GetNodeUid() string
	// InitPubsub will init new LibP2pPubsub instance with given chainId and maxMessageSize.
	InitPubsub(chainId string, maxMessageSize int) error
	// BroadcastWithChainId  will broadcast a msg to a PubSubTopic with the pubsub service which id is given chainId.
	BroadcastWithChainId(chainId string, topic string, netMsg *netPb.NetMsg) error
	// SubscribeWithChainId register a PubsubMsgHandler to a PubSubTopic with the pubsub service which id is given chainId.
	SubscribeWithChainId(chainId string, topic string, handler api.PubsubMsgHandler) error
	// CancelSubscribeWithChainId cancel subscribe a PubSubTopic with the pubsub service which id is given chainId.
	CancelSubscribeWithChainId(chainId string, topic string) error
	// SendMsg send msg to the node which id is given string.
	// 		msgFlag: is a flag used to distinguish msg type.
	SendMsg(chainId string, node string, msgFlag string, netMsg *netPb.NetMsg) error
	// DirectMsgHandle register a DirectMsgHandler to the net.
	// 		msgFlag: is a flag used to distinguish msg type.
	DirectMsgHandle(chainId string, msgFlag string, handler api.DirectMsgHandler) error
	// CancelDirectMsgHandle unregister a DirectMsgHandler.
	// 		msgFlag: is a flag used to distinguish msg type.
	CancelDirectMsgHandle(chainId string, msgFlag string) error
	// AddSeed add a seed node addr.
	AddSeed(seed string) error
	// RefreshSeeds refresh the seed node addr list.
	RefreshSeeds(seeds []string) error
	// AddTrustRoot add a tls root cert to the cert pool of chain.
	AddTrustRoot(chainId string, rootCertByte []byte) error
	// RefreshTrustRoots refresh the cert pool of chain.
	RefreshTrustRoots(chainId string, rootsCertsBytes [][]byte) error
	// ReVerifyTrustRoots will verify tls certs existed with the trust roots pool of the chain which id is the given chainId.
	ReVerifyTrustRoots(chainId string)
	// IsRunning return true when the net instance is running.
	IsRunning() bool
	// Start the local net.
	Start() error
	// Stop the local net.
	Stop() error
	// ChainNodesInfo return base node info list of chain which id is the given chainId.
	ChainNodesInfo(chainId string) ([]*api.ChainNodeInfo, error)
	// GetNodeUidByCertId return node uid which mapped to the given cert id. If unmapped return error.
	GetNodeUidByCertId(certId string) (string, error)
	// CheckRevokeTlsCerts check whether any tls certs revoked.
	CheckRevokeTlsCerts(ac api.AccessControlProvider, certManageSystemContractPayload []byte) error
	// AddAC add a AccessControlProvider for revoked validator.
	AddAC(chainId string, ac api.AccessControlProvider)
}

var _ Net = (*p2p.LibP2pNet)(nil)
