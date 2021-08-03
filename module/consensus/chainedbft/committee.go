/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import (
	"sync"
)

//peer defines basic peer information required by consensus
type peer struct {
	id     string //The network id
	index  uint64 //The index of committee
	active bool   //Peer's network state:online or offline
}

type indexedPeers []*peer

//Len returns the size of indexedPeers
func (ip indexedPeers) Len() int { return len(ip) }

//Swap swaps the ith object with jth object in indexedPeers
func (ip indexedPeers) Swap(i, j int) { ip[i], ip[j] = ip[j], ip[i] }

//Less checks the ith object's index < the jth object's index
func (ip indexedPeers) Less(i, j int) bool { return ip[i].index < ip[j].index }

//committee manages all of peers join current consensus epoch
type committee struct {
	sync.RWMutex
	peers []*peer // Consensus nodes at current epoch
}

//newCommittee initializes a peer pool with given peer list
func newCommittee(peers []*peer) *committee {
	return &committee{
		peers: peers,
	}
}

//getPeers returns peer list which are online
func (pp *committee) getPeers() []*peer {
	pp.RLock()
	defer pp.RUnlock()

	peers := make([]*peer, 0)
	for _, peer := range pp.peers {
		if peer.active {
			peers = append(peers, peer)
		}
	}
	return peers
}

//getPeerByIndex returns a peer with given index
func (pp *committee) getPeerByIndex(index uint64) *peer {
	pp.RLock()
	defer pp.RUnlock()
	for _, v := range pp.peers {
		if v.index == index {
			return v
		}
	}
	return nil
}

//isValidIdx checks whether a index is valid
func (pp *committee) isValidIdx(index uint64) bool {
	pp.RLock()
	defer pp.RUnlock()
	for _, v := range pp.peers {
		if v.index == index {
			return true
		}
	}
	return false
}
