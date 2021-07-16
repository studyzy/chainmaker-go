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
	switchEpochHeight uint64

	// last epoch info
	lastValidators     []*peer
	lastMinQuorumForQc int

	// curr epoch info
	peers              []*peer // Consensus nodes at current epoch
	currMinQuorumForQc int
}

//newCommittee initializes a peer pool with given peer list
func newCommittee(peers, lastValidators []*peer, switchHeight uint64) *committee {
	return &committee{
		peers:             peers,
		lastValidators:    lastValidators,
		switchEpochHeight: switchHeight,
	}
}

//getPeers returns peer list which are online
func (pp *committee) getPeers(blkHeight uint64) []*peer {
	usedPeers := pp.getUsedPeers(blkHeight)
	peers := make([]*peer, 0, len(usedPeers))
	for _, peer := range usedPeers {
		if peer.active {
			peers = append(peers, peer)
		}
	}
	return peers
}

//getPeerByIndex returns a peer with given index
func (pp *committee) getPeerByIndex(height uint64, index uint64) *peer {
	usedPeers := pp.getUsedPeers(height)
	for _, v := range usedPeers {
		if v.index == index {
			return v
		}
	}
	return nil
}

//getPeerByID returns a peer with given id
func (pp *committee) getPeerByID(height uint64, id string) *peer {
	usedPeers := pp.getUsedPeers(height)
	for _, v := range usedPeers {
		if v.id == id {
			return v
		}
	}
	return nil
}

//isValidIdx checks whether a index is valid
func (pp *committee) isValidIdx(height uint64, index uint64) bool {
	usedPeers := pp.getUsedPeers(height)
	for _, v := range usedPeers {
		if v.index == index {
			return true
		}
	}
	return false
}

//peerCount returns the size of core peers at current consensus epoch
func (pp *committee) peerCount(height uint64) int {
	usedPeers := pp.getUsedPeers(height)
	return len(usedPeers)
}

func (pp *committee) getUsedPeers(height uint64) []*peer {
	usedPeers := pp.peers
	if height <= pp.switchEpochHeight+3 {
		usedPeers = pp.lastValidators
	}
	return usedPeers
}

func (pp *committee) minQuorumForQc(height uint64) int {
	if height <= pp.switchEpochHeight+3 {
		return pp.lastMinQuorumForQc
	}
	return pp.currMinQuorumForQc
}
