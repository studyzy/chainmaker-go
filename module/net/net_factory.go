/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"errors"
	"io/ioutil"

	libp2p "chainmaker.org/chainmaker/net-libp2p/libp2pnet"
	liquid "chainmaker.org/chainmaker/net-liquid/liquidnet"
	"chainmaker.org/chainmaker/protocol/v2"
)

var ErrorNetType = errors.New("error net type")

// NetFactory provide a way to create net instance.
type NetFactory struct {
	netType protocol.NetType

	n protocol.Net
}

// NetOption is a function apply options to net instance.
type NetOption func(cfg *NetFactory) error

func WithReadySignalC(signalC chan struct{}) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetReadySignalC(signalC)
		case protocol.Liquid:
			// not supported
		}
		return nil
	}
}

// WithListenAddr set addr that the local net will listen on.
func WithListenAddr(addr string) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetListenAddr(addr)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			return liquid.SetListenAddrStr(n.HostConfig(), addr)
		}
		return nil
	}
}

// WithCrypto set private key file and tls cert file for the net to create connection.
func WithCrypto(pkMode bool, keyFile string, certFile string) NetOption {
	return func(nf *NetFactory) error {
		var (
			err                 error
			keyBytes, certBytes []byte
		)
		keyBytes, err = ioutil.ReadFile(keyFile)
		if err != nil {
			return err
		}
		if !pkMode {
			certBytes, err = ioutil.ReadFile(certFile)
			if err != nil {
				return err
			}
		}
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetPubKeyModeEnable(pkMode)
			n.Prepare().SetKey(keyBytes)
			if !pkMode {
				n.Prepare().SetCert(certBytes)
			}
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.CryptoConfig().PubKeyMode = pkMode
			n.CryptoConfig().KeyBytes = keyBytes
			if !pkMode {
				n.CryptoConfig().CertBytes = certBytes
			}
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
			n, _ := nf.n.(*libp2p.LibP2pNet)
			for _, seed := range seeds {
				n.Prepare().AddBootstrapsPeer(seed)
			}
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			for _, seed := range seeds {
				e := n.HostConfig().AddDirectPeer(seed)
				if e != nil {
					return e
				}
			}
		}
		return nil
	}
}

// WithPeerStreamPoolSize set the max stream pool size for every node that connected to us.
func WithPeerStreamPoolSize(size int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetPeerStreamPoolSize(size)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().SendStreamPoolCap = int32(size)
		}
		return nil
	}
}

// WithPubSubMaxMessageSize set max message size (M) for pub/sub.
func WithPubSubMaxMessageSize(size int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetPubSubMaxMsgSize(size)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.PubSubConfig().MaxPubMessageSize = size
		}
		return nil
	}
}

// WithMaxPeerCountAllowed set max count of nodes that connected to us.
func WithMaxPeerCountAllowed(max int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetMaxPeerCountAllow(max)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().MaxPeerCountAllowed = max
		}
		return nil
	}
}

// WithMaxConnCountAllowed set max count of connections for each peer that connected to us.
func WithMaxConnCountAllowed(max int) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			// not supported
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().MaxConnCountEachPeerAllowed = max
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
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetPeerEliminationStrategy(strategy)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().ConnEliminationStrategy = strategy
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
			n, _ := nf.n.(*libp2p.LibP2pNet)
			for _, ba := range blackAddresses {
				n.Prepare().AddBlackAddress(ba)
			}
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().BlackNetAddr = blackAddresses
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
			n, _ := nf.n.(*libp2p.LibP2pNet)
			for _, bn := range blackNodeIds {
				n.Prepare().AddBlackPeerId(bn)
			}
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			return n.HostConfig().AddBlackPeers(blackNodeIds...)
		}
		return nil
	}
}

// WithMsgCompression set whether compressing the payload when sending msg.
func WithMsgCompression(enable bool) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.SetCompressMsgBytes(enable)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().MsgCompress = enable
		}
		return nil
	}
}

func WithInsecurity(isInsecurity bool) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetIsInsecurity(isInsecurity)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.HostConfig().Insecurity = isInsecurity
		}
		return nil
	}
}

func WithPktEnable(pktEnable bool) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetPktEnable(pktEnable)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.ExtensionsConfig().EnablePkt = pktEnable
		}
		return nil
	}
}

// WithPriorityControlEnable config priority controller
func WithPriorityControlEnable(priorityCtrlEnable bool) NetOption {
	return func(nf *NetFactory) error {
		switch nf.netType {
		case protocol.Libp2p:
			n, _ := nf.n.(*libp2p.LibP2pNet)
			n.Prepare().SetPriorityCtrlEnable(priorityCtrlEnable)
		case protocol.Liquid:
			n, _ := nf.n.(*liquid.LiquidNet)
			n.ExtensionsConfig().EnablePriorityCtrl = priorityCtrlEnable
		}
		return nil
	}
}

// NewNet create a new net instance.
func (nf *NetFactory) NewNet(netType protocol.NetType, opts ...NetOption) (protocol.Net, error) {
	nf.netType = netType
	switch nf.netType {
	case protocol.Libp2p:
		localNet, err := libp2p.NewLibP2pNet(GlobalNetLogger)
		if err != nil {
			return nil, err
		}
		nf.n = localNet
	case protocol.Liquid:
		liquidNet, err := liquid.NewLiquidNet()
		if err != nil {
			return nil, err
		}
		nf.n = liquidNet
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
