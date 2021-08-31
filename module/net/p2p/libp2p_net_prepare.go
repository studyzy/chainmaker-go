/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"strconv"
	"strings"
	"sync"

	"chainmaker.org/chainmaker-go/net/p2p/libp2pgmtls"
	"chainmaker.org/chainmaker-go/net/p2p/libp2ptls"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
)

// LibP2pNetPrepare prepare the config options.
type LibP2pNetPrepare struct {
	listenAddr              string              // listenAddr
	bootstrapsPeers         map[string]struct{} // bootstrapsPeers
	pubSubMaxMsgSize        int                 // pubSubMaxMsgSize
	peerStreamPoolSize      int                 // peerStreamPoolSize
	maxPeerCountAllow       int                 // maxPeerCountAllow
	peerEliminationStrategy int                 // peerEliminationStrategy

	keyBytes  []byte // keyBytes
	certBytes []byte // certBytes

	chainTrustRootCertsBytes map[string][][]byte // chainTrustRootCertsBytes

	blackAddresses map[string]struct{} // blackAddresses
	blackPeerIds   map[string]struct{} // blackPeerIds

	isInsecurity bool

	lock sync.Mutex
}

func (l *LibP2pNetPrepare) SetIsInsecurity(isInsecurity bool) {
	l.isInsecurity = isInsecurity
}

// SetCert set cert with pem bytes.
func (l *LibP2pNetPrepare) SetCert(certPem []byte) {
	l.certBytes = certPem
}

// SetKey set private key with pem bytes.
func (l *LibP2pNetPrepare) SetKey(keyPem []byte) {
	l.keyBytes = keyPem
}

// SetPubSubMaxMsgSize set max msg size for pub-sub service.(M)
func (l *LibP2pNetPrepare) SetPubSubMaxMsgSize(pubSubMaxMsgSize int) {
	l.pubSubMaxMsgSize = pubSubMaxMsgSize
}

// SetPeerStreamPoolSize set stream pool max size of each peer.
func (l *LibP2pNetPrepare) SetPeerStreamPoolSize(peerStreamPoolSize int) {
	l.peerStreamPoolSize = peerStreamPoolSize
}

// AddBootstrapsPeer add a node address for connecting directly. It can be a seed node address or a consensus node address.
func (l *LibP2pNetPrepare) AddBootstrapsPeer(bootstrapAddr string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.bootstrapsPeers[bootstrapAddr] = struct{}{}
}

// SetTrustRootCerts set trust root certs for chain.
func (l *LibP2pNetPrepare) SetTrustRootCerts(chainId string, rootCerts [][]byte) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.chainTrustRootCertsBytes[chainId] = rootCerts
}

// AddTrustRootCert add a trust root cert for chain.
func (l *LibP2pNetPrepare) AddTrustRootCert(chainId string, rootCert []byte) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if _, ok := l.chainTrustRootCertsBytes[chainId]; !ok {
		l.chainTrustRootCertsBytes[chainId] = make([][]byte, 0)
	}
	l.chainTrustRootCertsBytes[chainId] = append(l.chainTrustRootCertsBytes[chainId], rootCert)
}

// SetListenAddr set address that the net will listen on.
// 		example: /ip4/127.0.0.1/tcp/10001
func (l *LibP2pNetPrepare) SetListenAddr(listenAddr string) {
	l.listenAddr = listenAddr
}

// SetMaxPeerCountAllow set max count of nodes that allow to connect to us.
func (l *LibP2pNetPrepare) SetMaxPeerCountAllow(maxPeerCountAllow int) {
	l.maxPeerCountAllow = maxPeerCountAllow
}

// SetPeerEliminationStrategy set the strategy for eliminating when reach the max count.
func (l *LibP2pNetPrepare) SetPeerEliminationStrategy(peerEliminationStrategy int) {
	l.peerEliminationStrategy = peerEliminationStrategy
}

// AddBlackAddress add a black address to blacklist.
// 		example: 192.168.1.14:8080
//		example: 192.168.1.14
func (l *LibP2pNetPrepare) AddBlackAddress(address string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	address = strings.ReplaceAll(address, "：", ":")
	if _, ok := l.blackAddresses[address]; !ok {
		l.blackAddresses[address] = struct{}{}
	}
}

// AddBlackPeerId add a black node id to blacklist.
// 		example: QmcQHCuAXaFkbcsPUj7e37hXXfZ9DdN7bozseo5oX4qiC4
func (l *LibP2pNetPrepare) AddBlackPeerId(pid string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if _, ok := l.blackPeerIds[pid]; !ok {
		l.blackPeerIds[pid] = struct{}{}
	}
}

func (ln *LibP2pNet) initPeerStreamManager() {
	ln.libP2pHost.peerStreamManager = newPeerStreamManager(ln.ctx, ln.libP2pHost.host, ln.messageHandlerDistributor, ln.prepare.peerStreamPoolSize)
}

func (ln *LibP2pNet) prepareBlackList() error {
	logger.Info("[Net] preparing blacklist...")
	for addr := range ln.prepare.blackAddresses {
		s := strings.Split(addr, ":")
		ip := s[0]
		var port = -1
		var err error
		if len(s) > 1 {
			port, err = strconv.Atoi(s[1])
			if err != nil {
				logger.Errorf("[Net] parse port failed, %s", err.Error())
				return err
			}
		}
		ln.libP2pHost.blackList.AddIPAndPort(ip, port)
		logger.Infof("[Net] black address found[%s]", addr)
	}
	for pid := range ln.prepare.blackPeerIds {
		peerId, err := peer.Decode(pid)
		if err != nil {
			logger.Errorf("[Net] decode pid failed(pid:%s), %s", pid, err.Error())
			return err
		}
		ln.libP2pHost.blackList.AddPeerId(peerId)
		logger.Infof("[Net] black peer id found[%s]", pid)
	}
	logger.Info("[Net] blacklist prepared.")
	return nil
}

// createLibp2pOptions create all necessary options for libp2p.
func (ln *LibP2pNet) createLibp2pOptions() ([]libp2p.Option, error) {
	logger.Info("[Net] creating options...")
	prvKey, err := ln.prepareKey()
	if err != nil {
		logger.Errorf("[Net] prepare key failed, %s", err.Error())
		return nil, err
	}
	connGater := NewConnGater(ln.libP2pHost.connManager, ln.libP2pHost.blackList, ln.libP2pHost.revokedValidator)
	listenAddrs := strings.Split(ln.prepare.listenAddr, ",")
	options := []libp2p.Option{
		libp2p.Identity(prvKey),
		libp2p.ListenAddrStrings(listenAddrs...),
		libp2p.ConnectionGater(connGater),
		libp2p.EnableRelay(circuit.OptHop),
		//libp2p.EnableNATService(),
	}
	if ln.prepare.isInsecurity {
		logger.Warn("[Net] use insecurity option.")
		options = append(options, libp2p.NoSecurity)
		ln.libP2pHost.isTls = false
		ln.libP2pHost.isGmTls = false
	} else {
		if ln.prepare.chainTrustRootCertsBytes == nil || len(ln.prepare.chainTrustRootCertsBytes) == 0 {
			logger.Warn("[Net] no trust root certs found. use default security.")
			options = append(options, libp2p.DefaultSecurity)
			ln.libP2pHost.isTls = false
			ln.libP2pHost.isGmTls = false
		} else if prvKey.Type() == pb.KeyType_SM2 {
			logger.Info("[Net] the priv key type found[sm2]. use gm tls security.")
			ln.libP2pHost.gmTlsChainTrustRoots, err = libp2pgmtls.BuildTlsTrustRoots(ln.prepare.chainTrustRootCertsBytes)
			if err != nil {
				logger.Errorf("[Net] build gm tls trust roots failed, %s", err.Error())
				return nil, err
			}
			ln.libP2pHost.initTlsCsAndSubassemblies()
			tpt := libp2pgmtls.New(
				ln.prepare.keyBytes,
				ln.prepare.certBytes,
				ln.libP2pHost.gmTlsChainTrustRoots,
				ln.libP2pHost.revokedValidator,
				ln.libP2pHost.newTlsPeerChainIdsNotifyC,
				ln.libP2pHost.newTlsCertIdPeerIdNotifyC,
				ln.libP2pHost.addPeerIdTlsCertNotifyC,
			)
			options = append(options, libp2p.Security(libp2pgmtls.ID, tpt))
			ln.libP2pHost.isTls = true
			ln.libP2pHost.isGmTls = true
		} else {
			logger.Info("[Net] the priv key type found[not sm2]. use normal tls security.")
			ln.libP2pHost.tlsChainTrustRoots, err = libp2ptls.BuildTlsTrustRoots(ln.prepare.chainTrustRootCertsBytes)
			if err != nil {
				logger.Errorf("[Net] build normal tls trust roots failed, %s", err.Error())
				return nil, err
			}
			ln.libP2pHost.initTlsCsAndSubassemblies()
			tpt := libp2ptls.New(
				ln.prepare.keyBytes,
				ln.prepare.certBytes,
				ln.libP2pHost.tlsChainTrustRoots,
				ln.libP2pHost.revokedValidator,
				ln.libP2pHost.newTlsPeerChainIdsNotifyC,
				ln.libP2pHost.newTlsCertIdPeerIdNotifyC,
				ln.libP2pHost.addPeerIdTlsCertNotifyC,
			)
			options = append(options, libp2p.Security(libp2ptls.ID, tpt))
			ln.libP2pHost.isTls = true
			ln.libP2pHost.isGmTls = false
		}
	}
	logger.Info("[Net] options created.")
	return options, nil
}

func (ln *LibP2pNet) prepareKey() (crypto.PrivKey, error) {
	logger.Info("[Net] node key preparing...")
	var privKey crypto.PrivKey = nil
	var err error = nil
	// read file
	skPemBytes := ln.prepare.keyBytes
	privateKey, err := asym.PrivateKeyFromPEM(skPemBytes, nil)
	if err != nil {
		logger.Errorf("[Net] parse pem to private key failed, %s", err.Error())
		return nil, err
	}
	privKey, _, err = crypto.KeyPairFromStdKey(privateKey.ToStandardKey())
	if err != nil {
		logger.Errorf("[Net] parse private key to priv key failed, %s", err.Error())
		return nil, err
	}
	logger.Info("[Net] node key prepared ok.")
	return privKey, err
}
