/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"strconv"
	"sync"
)

// BlackList is a blacklist for controlling nodes connection.
type BlackList struct {
	ipAndPort     map[string]struct{}
	ipAndPortLock sync.RWMutex

	peerId     map[peer.ID]struct{}
	peerIdLock sync.RWMutex
}

// NewBlackList create new BlackList instance.
func NewBlackList() *BlackList {
	return &BlackList{ipAndPort: make(map[string]struct{}), peerId: make(map[peer.ID]struct{})}
}

func createKeyWithIpAndPort(ip string, port int) string {
	var key string
	if port < 1 {
		key = ip
	} else {
		key = ip + ":" + strconv.Itoa(port)
	}
	return key
}

// AddIPAndPort add new IP and Port record to blacklist.
// If you want to control IP only, set port=0 pls.
func (b *BlackList) AddIPAndPort(ip string, port int) {
	b.ipAndPortLock.Lock()
	defer b.ipAndPortLock.Unlock()
	var key = createKeyWithIpAndPort(ip, port)
	if _, ok := b.ipAndPort[key]; ok {
		return
	}
	b.ipAndPort[key] = struct{}{}
}

// RemoveIPAndPort remove IP and Port record from blacklist.
func (b *BlackList) RemoveIPAndPort(ip string, port int) {
	b.ipAndPortLock.Lock()
	defer b.ipAndPortLock.Unlock()
	var key = createKeyWithIpAndPort(ip, port)
	if _, ok := b.ipAndPort[key]; ok {
		delete(b.ipAndPort, key)
	}
}

// ContainsIPAndPort return whether IP and Port exist in blacklist.
// If not found ip+port, but found ip only, return true.
func (b *BlackList) ContainsIPAndPort(ip string, port int) bool {
	b.ipAndPortLock.RLock()
	defer b.ipAndPortLock.RUnlock()
	var key = createKeyWithIpAndPort(ip, port)
	_, ok := b.ipAndPort[key]
	if !ok {
		_, ok = b.ipAndPort[ip]
	}
	return ok
}

// AddPeerId add new peer.ID to blacklist.
func (b *BlackList) AddPeerId(pid peer.ID) {
	b.peerIdLock.Lock()
	defer b.peerIdLock.Unlock()
	if _, ok := b.peerId[pid]; ok {
		return
	}
	b.peerId[pid] = struct{}{}
}

// RemovePeerId remove peer.ID given from blacklist.
func (b *BlackList) RemovePeerId(pid peer.ID) {
	b.peerIdLock.Lock()
	defer b.peerIdLock.Unlock()
	if _, ok := b.peerId[pid]; ok {
		delete(b.peerId, pid)
	}
}

// ContainsPeerId return whether peer.ID given exist in blacklist.
func (b *BlackList) ContainsPeerId(pid peer.ID) bool {
	b.peerIdLock.RLock()
	defer b.peerIdLock.RUnlock()
	_, ok := b.peerId[pid]
	return ok
}
