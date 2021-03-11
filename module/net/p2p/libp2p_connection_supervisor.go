/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
	"time"
)

// ConnSupervisor is a connections supervisor.
type ConnSupervisor struct {
	host              *LibP2pHost
	peerAddrInfos     []peer.AddrInfo
	peerAddrInfosLock sync.RWMutex
	signal            bool
	signalLock        sync.RWMutex
	startUp           bool
	tryConnectLock    sync.Mutex
	allConnected      bool
}

func (cs *ConnSupervisor) getSignal() bool {
	cs.signalLock.RLock()
	defer cs.signalLock.RUnlock()
	return cs.signal
}

func (cs *ConnSupervisor) setSignal(signal bool) {
	cs.signalLock.Lock()
	defer cs.signalLock.Unlock()
	cs.signal = signal
}

// newConnSupervisor create a new ConnSupervisor.
func newConnSupervisor(host *LibP2pHost, peerAddrInfos []peer.AddrInfo) *ConnSupervisor {
	return &ConnSupervisor{host: host, peerAddrInfos: peerAddrInfos, startUp: false, allConnected: false}
}

// getPeerAddrInfos get the addr infos of the peers for supervising.
func (cs *ConnSupervisor) getPeerAddrInfos() []peer.AddrInfo {
	cs.peerAddrInfosLock.RLock()
	defer cs.peerAddrInfosLock.RUnlock()
	return cs.peerAddrInfos
}

// refreshPeerAddrInfos refresh the addr infos of the peers for supervising.
func (cs *ConnSupervisor) refreshPeerAddrInfos(peerAddrInfos []peer.AddrInfo) {
	cs.peerAddrInfosLock.Lock()
	defer cs.peerAddrInfosLock.Unlock()
	cs.peerAddrInfos = peerAddrInfos
}

// startSupervising start a goroutine to supervise connections.
func (cs *ConnSupervisor) startSupervising() {
	if cs.startUp {
		return
	}
	cs.setSignal(true)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Error(err)
			}
		}()
		cs.startUp = true
		for cs.getSignal() {
			//if cs.host.connManager.ConnCount() < len(cs.getPeerAddrInfos()) {
			cs.try()
			//}
			time.Sleep(5 * time.Second)
		}
		cs.startUp = false
	}()
}

func (cs *ConnSupervisor) try() {
	if len(cs.peerAddrInfos) > 0 {
		cs.tryConnectLock.Lock()
		peerAddrInfos := cs.getPeerAddrInfos()
		count := len(peerAddrInfos)
		connectedCount := 0
		for _, peerInfo := range cs.getPeerAddrInfos() {
			if cs.host.host.ID() == peerInfo.ID || cs.host.HasConnected(peerInfo.ID) {
				connectedCount++
				if connectedCount == count && !cs.allConnected {
					logger.Infof("[ConnSupervisor] all necessary peers connected.")
					cs.allConnected = true
				}
				continue
			}
			cs.allConnected = false
			logger.Debugf("[ConnSupervisor] try to connect(peer:%s)", peerInfo)
			if err := cs.host.Host().Connect(cs.host.Context(), peerInfo); err != nil {
				logger.Warnf("[ConnSupervisor] try to connect to peer failed(peer:%s),%s", peerInfo, err.Error())
			}
		}
		cs.tryConnectLock.Unlock()
	}
}

// stopSupervising stop supervising.
func (cs *ConnSupervisor) stopSupervising() {
	cs.signal = false
}

// handleChanNewPeerFound handle the new peer found which got from discovery.
func (cs *ConnSupervisor) handleChanNewPeerFound(peerChan <-chan peer.AddrInfo) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("[ConnSupervisor.handleChanNewPeerFound] recover err, %s", err)
			}
		}()
		for p := range peerChan {
			cs.tryConnectLock.Lock()
			if p.ID == cs.host.Host().ID() || cs.host.HasConnected(p.ID) {
				cs.tryConnectLock.Unlock()
				continue
			}
			err := cs.host.Host().Connect(cs.host.Context(), p)
			if err != nil {
				logger.Warnf("[ConnSupervisor] new connection connect failed(remote peer id:%s, remote addr:%s),%s", p.ID.Pretty(), p.Addrs[0].String(), err.Error())
			} else {
				logger.Debug("[ConnSupervisor] new connection connected(remote peer id:%s, remote addr:%s)", p.ID.Pretty(), p.Addrs[0].String())
			}
			cs.tryConnectLock.Unlock()
		}
	}()
}
