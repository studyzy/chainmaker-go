/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chainedbft

import consensuspb "chainmaker.org/chainmaker/pb-go/consensus"

//peer defines basic peer information required by consensus
type peer struct {
	id    string //The network id
	index int64  //The index of committee
}

type indexedPeers []*peer

//Len returns the size of indexedPeers
func (ip indexedPeers) Len() int { return len(ip) }

//Swap swaps the ith object with jth object in indexedPeers
func (ip indexedPeers) Swap(i, j int) { ip[i], ip[j] = ip[j], ip[i] }

//Less checks the ith object's index < the jth object's index
func (ip indexedPeers) Less(i, j int) bool { return ip[i].index < ip[j].index }

type contractInfo struct {
	*committee
	*consensuspb.GovernanceContract
}

func newContractInfo(governContract *consensuspb.GovernanceContract) *contractInfo {
	peers := make([]*peer, 0, len(governContract.Validators))
	lastPeers := make([]*peer, 0, len(governContract.LastValidators))
	for _, validator := range governContract.Validators {
		peers = append(peers, &peer{
			id:    validator.NodeId,
			index: validator.Index,
		})
	}
	for _, validator := range governContract.LastValidators {
		lastPeers = append(lastPeers, &peer{
			id:    validator.NodeId,
			index: validator.Index,
		})
	}
	committee := newCommittee(peers, lastPeers, governContract.NextSwitchHeight,
		governContract.MinQuorumForQc, governContract.LastMinQuorumForQc)
	return &contractInfo{committee: committee, GovernanceContract: governContract}
}

//committee manages all of peers join current consensus epoch
type committee struct {
	switchEpochHeight uint64

	// last epoch info
	lastValidators     []*peer
	lastMinQuorumForQc int

	// curr epoch info
	validators         []*peer // Consensus nodes at current epoch
	currMinQuorumForQc int
}

//newCommittee initializes a peer pool with given peer list
func newCommittee(peers, lastValidators []*peer, switchHeight uint64, quorumQc, lastQuorumQc uint64) *committee {
	return &committee{
		validators:         peers,
		lastValidators:     lastValidators,
		switchEpochHeight:  switchHeight,
		currMinQuorumForQc: int(quorumQc),
		lastMinQuorumForQc: int(lastQuorumQc),
	}
}

//getPeers returns peer list which are online
func (pp *committee) getPeers(blkHeight uint64) []*peer {
	usedPeers := pp.getUsedPeers(blkHeight)
	return usedPeers
}

//getPeerByIndex returns a peer with given index
func (pp *committee) getPeerByIndex(height uint64, index uint64) *peer {
	usedPeers := pp.getUsedPeers(height)
	for _, v := range usedPeers {
		if v.index == int64(index) {
			return v
		}
	}
	return nil
}

//isValidIdx checks whether a index is valid
func (pp *committee) isValidIdx(height uint64, index uint64) bool {
	usedPeers := pp.getUsedPeers(height)
	for _, v := range usedPeers {
		if v.index == int64(index) {
			return true
		}
	}
	return false
}

func (pp *committee) minQuorumForQc(height uint64) int {
	if height <= pp.switchEpochHeight+3 {
		return pp.lastMinQuorumForQc
	}
	return pp.currMinQuorumForQc
}

func (pp *committee) getUsedPeers(height uint64) []*peer {
	if height == 0 {
		return pp.validators
	}
	usedPeers := pp.validators
	if height <= pp.switchEpochHeight+3 {
		usedPeers = pp.lastValidators
	}
	return usedPeers
}
