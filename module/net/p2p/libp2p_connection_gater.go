/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"chainmaker.org/chainmaker-go/net/p2p/revoke"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"strconv"
	"strings"
)

// ConnGater is an implementation of ConnectionGater interface.
type ConnGater struct {
	connManager      *PeerConnManager
	blackList        *BlackList
	revokedValidator *revoke.RevokedValidator
}

func NewConnGater(connManager *PeerConnManager, blackList *BlackList, revokedValidator *revoke.RevokedValidator) *ConnGater {
	return &ConnGater{connManager: connManager, blackList: blackList, revokedValidator: revokedValidator}
}

// InterceptPeerDial
func (cg *ConnGater) InterceptPeerDial(p peer.ID) bool {
	return true
}

// InterceptAddrDial
func (cg *ConnGater) InterceptAddrDial(p peer.ID, mu multiaddr.Multiaddr) bool {
	//bl := true
	//if bl {
	//	logger.Debug("[Gater] InterceptAddrDial -> peer:", p.String(), ",multiaddr:", mu.String(), ",result:", bl)
	//} else {
	//	logger.Info("[Gater] InterceptAddrDial -> peer:", p.String(), ",multiaddr:", mu.String(), ",result:", bl)
	//}
	//return bl
	return true
}

// InterceptAccept will be checked first when other peer connect to us.
func (cg *ConnGater) InterceptAccept(cm network.ConnMultiaddrs) bool {
	return true
}

// InterceptSecured
func (cg *ConnGater) InterceptSecured(d network.Direction, p peer.ID, cm network.ConnMultiaddrs) bool {
	remoteAddr := cm.RemoteMultiaddr().String()
	s := strings.Split(remoteAddr, "/")
	ip := s[2]
	port, _ := strconv.Atoi(s[4])
	if cg.blackList.ContainsIPAndPort(ip, port) {
		logger.Warnf("[ConnGater.InterceptSecured] connection remote address in blacklist. rejected. (remote addr:%s)", remoteAddr)
		return false
	}
	if cg.blackList.ContainsPeerId(p) {
		logger.Warnf("[ConnGater.InterceptSecured] peer in blacklist. rejected. (peer-id:%s)", p.Pretty())
		return false
	}
	if cg.revokedValidator.ContainsPeerId(p.Pretty()) {
		logger.Warnf("[ConnGater.InterceptSecured] peer id in revoked list. rejected. (peer-id:%s)", p.Pretty())
		return false
	}
	if d == network.DirInbound {
		connState := cg.connManager.IsConnected(p)
		if connState {
			logger.Warnf("[ConnGater.InterceptSecured] peer has connected. ignored. (peer-id:%s)", p.Pretty())
			return false
		}
	}
	if !cg.connManager.CanConnect(p) {
		logger.Warnf("[ConnGater.InterceptSecured] connection not allowed. ignored. (peer-id:%s)", p.Pretty())
		return false
	}
	logger.Debugf("[ConnGater.InterceptSecured] connection secured (direction:%s, remote peer-id:%s, remote multi-addr:%s)", d, p.Pretty(), cm.RemoteMultiaddr())
	return true
}

// InterceptUpgraded
func (cg *ConnGater) InterceptUpgraded(c network.Conn) (bool, control.DisconnectReason) {
	//p := c.RemotePeer()
	//connState := cg.connRecorder.IsConnected(p)
	//if connState {
	//	logger.Warnf("[Gater] InterceptUpgraded : %s peer has connected. Ignored.", p.Pretty())
	//	return false, 0
	//}
	//cg.connRecorder.AddConn(c.RemotePeer(), c)
	//logger.Debugf("[Gater] InterceptUpgraded : new connection upgraded , remote peer id:%s, remote addr:%s", p.Pretty(), c.RemoteMultiaddr().String())
	return true, 0
}
