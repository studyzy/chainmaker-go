/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/net"
)

// NetType is the type of net.
type NetType int

const (
	// Libp2p is a type of p2p net.
	Libp2p NetType = iota
	// GRpc is a type of rpc net.
	GRpc
)

// DirectMsgHandler handle the msg received from other node.
type DirectMsgHandler func(from string, netMsg *net.NetMsg) error

// PubsubMsgHandler handle the msg published by other node.
type PubsubMsgHandler func(publisher string, netMsg *net.NetMsg) error

type ChainNodeInfo struct {
	NodeUid     string
	NodeAddress []string
	NodeTlsCert []byte
}

type MsgHandler func(from string, msg []byte, msgType net.NetMsg_MsgType) error

// ChainNodesInfoProvider provide base node info list of chain.
type ChainNodesInfoProvider interface {
	// GetChainNodesInfo return base node info list of chain.
	GetChainNodesInfo() ([]*ChainNodeInfo, error)
}

//  NetService
type NetService interface {
	// BroadcastMsg broadcast a msg to the net.
	BroadcastMsg(msg []byte, msgType net.NetMsg_MsgType) error
	// Subscribe register a MsgHandler for subscribe.
	Subscribe(msgType net.NetMsg_MsgType, handler MsgHandler) error
	// CancelSubscribe cancel subscribe.
	CancelSubscribe(msgType net.NetMsg_MsgType) error
	// ConsensusBroadcastMsg broadcast a msg to the consensus nodes.
	ConsensusBroadcastMsg(msg []byte, msgType net.NetMsg_MsgType) error
	// ConsensusSubscribe register a MsgHandler handle the msg from consensus nodes for subscribe.
	ConsensusSubscribe(msgType net.NetMsg_MsgType, handler MsgHandler) error
	// CancelConsensusSubscribe cancel subscribe.
	CancelConsensusSubscribe(msgType net.NetMsg_MsgType) error
	// SendMsg send msg to any nodes.
	SendMsg(msg []byte, msgType net.NetMsg_MsgType, to ...string) error
	// ReceiveMsg register a MsgHandler to handle the msg received from other node.
	ReceiveMsg(msgType net.NetMsg_MsgType, handler MsgHandler) error

	// Start the net service.
	Start() error

	// Stop the net service.
	Stop() error

	// GetNodeUidByCertId return node uid which mapped to the given cert id. If unmapped return error.
	GetNodeUidByCertId(certId string) (string, error)

	// GetChainNodesInfoProvider return a implementation of ChainNodesInfoProvider.
	GetChainNodesInfoProvider() ChainNodesInfoProvider
}
