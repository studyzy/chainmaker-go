/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"fmt"
	"sync"
	"time"
)

// PeerIdChainIdsRecorder record the chain ids of peer .
type PeerIdChainIdsRecorder struct {
	lock                                 sync.Mutex
	newTlsPeerChainIdsNotifyChanHandling bool
	removeTlsPeerNotifyChanHandling      bool
	records                              map[string]*StringMapList
	addC                                 <-chan map[string][]string
	onAddC                               chan<- string
	removeC                              <-chan string
	onRemoveC                            chan<- string
	stopHandleAddSignalC                 chan struct{}
	stopHandleRemoveSignalC              chan struct{}
}

func newPeerIdChainIdsRecorder(addC <-chan map[string][]string, removeC <-chan string) *PeerIdChainIdsRecorder {
	return &PeerIdChainIdsRecorder{records: make(map[string]*StringMapList), addC: addC, removeC: removeC, stopHandleAddSignalC: make(chan struct{}), stopHandleRemoveSignalC: make(chan struct{})}
}

func (pcr *PeerIdChainIdsRecorder) onAddNotifyC(onAddC chan<- string) {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	pcr.onAddC = onAddC
}

func (pcr *PeerIdChainIdsRecorder) onRemoveNotifyC(onRemoveC chan<- string) {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	pcr.onRemoveC = onRemoveC
}

func (pcr *PeerIdChainIdsRecorder) addPeerChainId(peerId string, chainId string) bool {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	mapList, ok := pcr.records[peerId]
	if !ok {
		pcr.records[peerId] = NewStringMapList()
		mapList = pcr.records[peerId]
	}
	result := mapList.Add(chainId)
	if result && pcr.onAddC != nil {
		pcr.onAddC <- peerId + "<-->" + chainId
	}
	return result
}

func (pcr *PeerIdChainIdsRecorder) removeAllByPeerId(peerId string) bool {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	chains, ok := pcr.records[peerId]
	if ok {
		if pcr.onRemoveC != nil {
			for chainId := range chains.mapList {
				pcr.onRemoveC <- peerId + "<-->" + chainId
			}
		}
		delete(pcr.records, peerId)
		return true
	}
	return false
}

func (pcr *PeerIdChainIdsRecorder) isPeerBelongToChain(peerId string, chainId string) bool {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	m, ok := pcr.records[peerId]
	if ok {
		return m.Contains(chainId)
	}
	return false
}

func (pcr *PeerIdChainIdsRecorder) peerIdsOfChain(chainId string) []string {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	result := make([]string, 0)
	for peerId, mapList := range pcr.records {
		if mapList.Contains(chainId) {
			result = append(result, peerId)
		}
	}
	return result
}

func (pcr *PeerIdChainIdsRecorder) stopHandling() {
	pcr.stopHandleAddSignalC <- struct{}{}
	pcr.stopHandleRemoveSignalC <- struct{}{}
}

func (pcr *PeerIdChainIdsRecorder) handleNewTlsPeerChainIdsNotifyC() error {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	if pcr.newTlsPeerChainIdsNotifyChanHandling {
		return fmt.Errorf("peer chainid recorder is handling")
	}
	go func() {
		for {
			if pcr.addC == nil {
				select {
				case <-pcr.stopHandleAddSignalC:
					logger.Infof("[PeerIdTlsCertStore] stop handling new notify chan.")
					return
				default:
					time.Sleep(5 * time.Second)
					continue
				}
			}

			select {
			case m := <-pcr.addC:
				for peerId, chainIds := range m {
					for _, chainId := range chainIds {
						logger.Debugf("[PeerIdChainIdsRecorder] recv new from notify chan(peer-id:%s, chain-id:%s)", peerId, chainId)
						pcr.addPeerChainId(peerId, chainId)
					}
				}
			case <-pcr.stopHandleAddSignalC:
				logger.Infof("[PeerIdChainIdsRecorder] stop handling new notify chan.")
				return
			}
		}
	}()
	pcr.newTlsPeerChainIdsNotifyChanHandling = true
	return nil
}

func (pcr *PeerIdChainIdsRecorder) handleRemoveTlsPeerNotifyC() error {
	pcr.lock.Lock()
	defer pcr.lock.Unlock()
	if pcr.removeTlsPeerNotifyChanHandling {
		return fmt.Errorf("peer chainid recorder is handling")
	}
	go func() {
		for {
			if pcr.removeC == nil {
				select {
				case <-pcr.stopHandleRemoveSignalC:
					logger.Infof("[PeerIdTlsCertStore] stop handling new notify chan.")
					return
				default:
					time.Sleep(5 * time.Second)
					continue
				}
			}

			select {
			case peerId := <-pcr.removeC:
				logger.Debugf("[PeerIdChainIdsRecorder] recv removed peer-id from notify chan(peer-id:%s)", peerId)
				pcr.removeAllByPeerId(peerId)
			case <-pcr.stopHandleRemoveSignalC:
				logger.Infof("[PeerIdChainIdsRecorder] stop handling remove notify chan.")
				return
			}
		}
	}()
	pcr.removeTlsPeerNotifyChanHandling = true
	return nil
}
