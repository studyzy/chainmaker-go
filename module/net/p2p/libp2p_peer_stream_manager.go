/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
)

// PeerStreamManager is a stream manager of peers.
type PeerStreamManager struct {
	ctx               context.Context
	streamPoolCap     int
	host              host.Host
	mhd               *MessageHandlerDistributor
	peerStreamPoolMap map[peer.ID]*StreamPool
	lock              sync.RWMutex
}

func newPeerStreamManager(ctx context.Context, host host.Host, mhd *MessageHandlerDistributor, streamPoolCap int) *PeerStreamManager {
	if streamPoolCap < 1 {
		streamPoolCap = DefaultStreamPoolCap
	}
	return &PeerStreamManager{ctx: ctx, host: host, mhd: mhd, peerStreamPoolMap: make(map[peer.ID]*StreamPool), streamPoolCap: streamPoolCap}
}

func (psm *PeerStreamManager) initPeerStream(pid peer.ID) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	_, ok := psm.peerStreamPoolMap[pid]
	if ok {
		return
	}
	createStreamFunc := func() (network.Stream, error) {
		stream, err := psm.host.NewStream(psm.ctx, pid, MsgPID)
		if err != nil {
			return nil, err
		}

		// if you want to use two-way stream , open this
		//var streamReadHandlerFunc = NewStreamReadHandlerFunc(psm.mhd)
		//go streamReadHandlerFunc(stream)

		return stream, nil
	}
	streamPool := newStreamPool(psm.streamPoolCap, createStreamFunc)
	psm.peerStreamPoolMap[pid] = streamPool
}

func (psm *PeerStreamManager) borrowPeerStream(pid peer.ID) (network.Stream, error) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	streamPool, ok := psm.peerStreamPoolMap[pid]
	if !ok {
		return nil, errors.New("peer streams not init")
	}
	return streamPool.borrowStream()
}

func (psm *PeerStreamManager) returnPeerStream(pid peer.ID, stream network.Stream) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	if stream == nil {
		return
	}
	streamPool, ok := psm.peerStreamPoolMap[pid]
	if !ok {
		return
	}
	streamPool.returnStream(stream)
}

func (psm *PeerStreamManager) addPeerStream(pid peer.ID, stream network.Stream) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	if stream == nil {
		return
	}
	streamPool, ok := psm.peerStreamPoolMap[pid]
	if !ok {
		return
	}
	streamPool.addStream(stream)
}

func (psm *PeerStreamManager) dropPeerStream(pid peer.ID, stream network.Stream) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	if stream == nil {
		return
	}
	streamPool, ok := psm.peerStreamPoolMap[pid]
	if !ok {
		return
	}
	logger.Debugf("[newPeerStreamManager] drop a stream of peer[%s]", pid.Pretty())
	streamPool.dropStream(stream)
}

func (psm *PeerStreamManager) cleanPeerStream(pid peer.ID) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	streamPool, ok := psm.peerStreamPoolMap[pid]
	if !ok {
		return
	}
	delete(psm.peerStreamPoolMap, pid)
	go func() {
		streamPool.cleanAndDisable()
	}()

}

func (psm *PeerStreamManager) reset() {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	psm.peerStreamPoolMap = make(map[peer.ID]*StreamPool)
}
