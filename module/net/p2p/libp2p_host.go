/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"chainmaker.org/chainmaker-go/net/p2p/libp2pgmtls"
	"chainmaker.org/chainmaker-go/net/p2p/libp2ptls"
	"chainmaker.org/chainmaker-go/net/p2p/revoke"
	"context"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
	"time"
)

// networkNotify is an implementation of network.Notifiee.
var networkNotify = func(host *LibP2pHost) network.Notifiee {
	return &network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			times := 10
			for (host.peerStreamManager == nil || host.connManager == nil) && times > 0 {
				times--
				time.Sleep(time.Second)
			}
			host.peerStreamManager.initPeerStream(c.RemotePeer())
			pid := c.RemotePeer()
			host.connManager.AddConn(pid, c)
			logger.Infof("[Host] new connection connected(remote peer-id:%s, remote multi-addr:%s)", c.RemotePeer().Pretty(), c.RemoteMultiaddr().String())

		},
		DisconnectedF: func(_ network.Network, c network.Conn) {
			times := 10
			for (host.peerStreamManager == nil || host.connManager == nil) && times > 0 {
				times--
				time.Sleep(time.Second)
			}
			logger.Infof("[Host] connection disconnected(remote peer-id:%s, remote multi-addr:%s)", c.RemotePeer().Pretty(), c.RemoteMultiaddr().String())
			host.connManager.RemoveConn(c.RemotePeer())
			pid := c.RemotePeer().Pretty()
			if host.removeTlsPeerNotifyC != nil {
				host.removeTlsPeerNotifyC <- pid
			}
			if host.removeTlsCertIdPeerIdNotifyC != nil {
				host.removeTlsCertIdPeerIdNotifyC <- pid
			}
			if host.removePeerIdTlsCertNotifyC != nil {
				host.removePeerIdTlsCertNotifyC <- pid
			}
			host.peerStreamManager.cleanPeerStream(c.RemotePeer())
		},
	}
}

// LibP2pHost is a libP2pHost which use libp2p as local net provider.
type LibP2pHost struct {
	startUp                      bool
	lock                         sync.Mutex
	ctx                          context.Context
	host                         host.Host                // host
	connManager                  *PeerConnManager         // connManager
	blackList                    *BlackList               // blackList
	revokedValidator             *revoke.RevokedValidator // revokedValidator
	peerStreamManager            *PeerStreamManager
	connSupervisor               *ConnSupervisor
	isTls                        bool
	isGmTls                      bool
	peerChainIdsRecorder         *PeerIdChainIdsRecorder
	newTlsPeerChainIdsNotifyC    chan map[string][]string
	removeTlsPeerNotifyC         chan string
	certPeerIdMapper             *CertIdPeerIdMapper
	newTlsCertIdPeerIdNotifyC    chan string
	removeTlsCertIdPeerIdNotifyC chan string
	peerIdTlsCertStore           *PeerIdTlsCertStore
	addPeerIdTlsCertNotifyC      chan map[string][]byte
	removePeerIdTlsCertNotifyC   chan string
	tlsChainTrustRoots           *libp2ptls.ChainTrustRoots
	gmTlsChainTrustRoots         *libp2pgmtls.ChainTrustRoots
	opts                         []libp2p.Option
}

func (lh *LibP2pHost) initTlsCsAndSubassemblies() {
	lh.newTlsPeerChainIdsNotifyC = make(chan map[string][]string, 50)
	lh.removeTlsPeerNotifyC = make(chan string, 50)
	lh.newTlsCertIdPeerIdNotifyC = make(chan string, 50)
	lh.removeTlsCertIdPeerIdNotifyC = make(chan string, 50)
	lh.addPeerIdTlsCertNotifyC = make(chan map[string][]byte, 50)
	lh.removePeerIdTlsCertNotifyC = make(chan string, 50)
	lh.peerChainIdsRecorder = newPeerIdChainIdsRecorder(lh.newTlsPeerChainIdsNotifyC, lh.removeTlsPeerNotifyC)
	lh.certPeerIdMapper = newCertIdPeerIdMapper(lh.newTlsCertIdPeerIdNotifyC, lh.removeTlsCertIdPeerIdNotifyC)
	lh.peerIdTlsCertStore = newPeerIdTlsCertStore(lh.addPeerIdTlsCertNotifyC, lh.removePeerIdTlsCertNotifyC)
}

// PeerStreamManager
func (lh *LibP2pHost) PeerStreamManager() *PeerStreamManager {
	return lh.peerStreamManager
}

// Context
func (lh *LibP2pHost) Context() context.Context {
	return lh.ctx
}

// Host is libp2p.Host.
func (lh *LibP2pHost) Host() host.Host {
	return lh.host
}

// HasConnected return true if the peer which id is the peerId given has connected. Otherwise return false.
func (lh *LibP2pHost) HasConnected(peerId peer.ID) bool {
	return lh.connManager.IsConnected(peerId)
}

// IsRunning return true when libp2p has started up.Otherwise return false.
func (lh *LibP2pHost) IsRunning() bool {
	return lh.startUp
}

// NewLibP2pHost create new LibP2pHost instance.
func NewLibP2pHost(ctx context.Context) *LibP2pHost {
	return &LibP2pHost{
		startUp:          false,
		ctx:              ctx,
		connManager:      NewPeerConnManager(),
		blackList:        NewBlackList(),
		revokedValidator: revoke.NewRevokedValidator(),
		opts:             make([]libp2p.Option, 0),
	}
}

// Start libP2pHost.
func (lh *LibP2pHost) Start() error {
	lh.lock.Lock()
	defer lh.lock.Unlock()
	if lh.startUp {
		logger.Warn("[Host] host is running. ignored.")
		return nil
	}
	logger.Info("[Host] stating host...")
	node, err := libp2p.New(lh.ctx, lh.opts...)
	if err != nil {
		return err
	}
	lh.host = node
	// network notify
	node.Network().Notify(networkNotify(lh))
	logger.Info("[Host] host stated.")
	for _, addr := range node.Addrs() {
		logger.Infof("[Host] host listening on address:%s/p2p/%s", addr.String(), node.ID().Pretty())
	}
	if err := lh.handleTlsPeerChainIdsNotifyC(); err != nil {
		return err
	}
	if err := lh.handleTlsCertIdPeerIdNotifyC(); err != nil {
		return err
	}
	if err := lh.handlePeerIdTlsCertStoreNotifyC(); err != nil {
		return err
	}
	lh.startUp = true
	return nil
}

func (lh *LibP2pHost) handleTlsPeerChainIdsNotifyC() error {
	if lh.peerChainIdsRecorder != nil {
		if err := lh.peerChainIdsRecorder.handleNewTlsPeerChainIdsNotifyC(); err != nil {
			return err
		}
		if err := lh.peerChainIdsRecorder.handleRemoveTlsPeerNotifyC(); err != nil {
			return err
		}
	}
	return nil
}

func (lh *LibP2pHost) handleTlsCertIdPeerIdNotifyC() error {
	if lh.certPeerIdMapper != nil {
		if err := lh.certPeerIdMapper.handleNewTlsCertIdPeerIdNotifyC(); err != nil {
			return err
		}
		if err := lh.certPeerIdMapper.handleRemoveTlsPeerNotifyC(); err != nil {
			return err
		}
	}
	return nil
}

func (lh *LibP2pHost) handlePeerIdTlsCertStoreNotifyC() error {
	if lh.peerIdTlsCertStore != nil {
		if err := lh.peerIdTlsCertStore.startHandlingNotifyC(); err != nil {
			return err
		}
	}
	return nil
}

// Stop libP2pHost.
func (lh *LibP2pHost) Stop() error {
	if lh.peerChainIdsRecorder != nil {
		lh.peerChainIdsRecorder.stopHandling()
	}
	if lh.certPeerIdMapper != nil {
		lh.certPeerIdMapper.stopHandling()
	}
	if lh.connSupervisor != nil {
		lh.connSupervisor.stopSupervising()
	}
	if lh.peerIdTlsCertStore != nil {
		lh.peerIdTlsCertStore.stopHandling()
	}
	lh.peerStreamManager.reset()
	return lh.host.Close()
}
