/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"errors"
	"io/ioutil"

	"chainmaker.org/chainmaker-go/net/p2p"
	"chainmaker.org/chainmaker/protocol/v2"
)

// ErrorNetType
var ErrorNetType = errors.New("error net type")

// NetFactory provide a way to create net instance.
type NetFactory struct {
	netType protocol.NetType

	n Net
}

// NetOption is a function apply options to net instance.
type NetOption func(cfg *NetFactory) error

// WithListenAddr set addr that the local net will listen on.
func WithListenAddr(addr string) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetListenAddr(addr)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithCrypto set private key file and tls cert file for the net to create connection.
func WithCrypto(keyFile string, certFile string) NetOption {
	return func(nf *NetFactory) error {
		keyBytes, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return err
		}
		certBytes, err := ioutil.ReadFile(certFile)
		if err != nil {
			return err
		}
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetKey(keyBytes)
			n.Prepare().SetCert(certBytes)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithSeeds set addresses of discovery service node.
func WithSeeds(seeds ...string) NetOption {
	return func(nf *NetFactory) error {
		if seeds == nil {
			return nil
		}
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			for _, seed := range seeds {
				n.Prepare().AddBootstrapsPeer(seed)
			}
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithPeerStreamPoolSize set the max stream pool size for every node that connected to us.
func WithPeerStreamPoolSize(size int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetPeerStreamPoolSize(size)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithPubSubMaxMessageSize set max message size (M) for pub/sub.
func WithPubSubMaxMessageSize(size int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetPubSubMaxMsgSize(size)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithMaxPeerCountAllow set max count of nodes that connected to us.
func WithMaxPeerCountAllow(max int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetMaxPeerCountAllow(max)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithPeerEliminationStrategy set the strategy for eliminating node when the count of nodes
// that connected to us reach the max value.
func WithPeerEliminationStrategy(strategy int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetPeerEliminationStrategy(strategy)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithBlackAddresses set addresses of the nodes for blacklist.
func WithBlackAddresses(blackAddresses ...string) NetOption {
	return func(nf *NetFactory) error {
		if blackAddresses == nil {
			return nil
		}
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			for _, ba := range blackAddresses {
				n.Prepare().AddBlackAddress(ba)
			}
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// WithBlackNodeIds set ids of the nodes for blacklist.
func WithBlackNodeIds(blackNodeIds ...string) NetOption {
	return func(nf *NetFactory) error {
		if blackNodeIds == nil {
			return nil
		}
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			for _, bn := range blackNodeIds {
				n.Prepare().AddBlackPeerId(bn)
			}
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

func WithMsgCompression(enable bool) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.SetCompressMsgBytes(enable)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

func WithInsecurity(isInsecurity bool) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*p2p.LibP2pNet)
			n.Prepare().SetIsInsecurity(isInsecurity)
		case protocol.GRpc:
			//TODO gRpc
		}
		return nil
	}
}

// NewNet create a new net instance.
func (nf *NetFactory) NewNet(netType protocol.NetType, opts ...NetOption) (Net, error) {
	nf.netType = netType
	//switch网络类型
	switch nf.netType {
	case protocol.Libp2p:
		//初始化Libp2pNet实现
		localNet, err := p2p.NewLibP2pNet() //创建Libp2pNet实例
		if err != nil {
			return nil, err
		}
		nf.n = localNet
	case protocol.GRpc:
		//初始化gRpcNet实现 TODO
		return nil, nil
	default:
		return nil, ErrorNetType
	}
	if err := nf.Apply(opts...); err != nil {
		return nil, err
	}
	return nf.n, nil
}

// Apply options.
func (nf *NetFactory) Apply(opts ...NetOption) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(nf); err != nil {
			return err
		}
	}
	return nil
}
