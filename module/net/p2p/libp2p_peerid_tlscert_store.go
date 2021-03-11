/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// PeerIdTlsCertStore record the tls cert bytes of peer .
type PeerIdTlsCertStore struct {
	lock                    sync.Mutex
	store                   map[string][]byte
	addC                    <-chan map[string][]byte
	removeC                 <-chan string
	handlingAddCState       bool
	handlingRemoveCState    bool
	stopHandleAddSignalC    chan struct{}
	stopHandleRemoveSignalC chan struct{}
}

func newPeerIdTlsCertStore(addNotifyC <-chan map[string][]byte, removeNotifyC <-chan string) *PeerIdTlsCertStore {
	return &PeerIdTlsCertStore{store: make(map[string][]byte), addC: addNotifyC, removeC: removeNotifyC, stopHandleAddSignalC: make(chan struct{}), stopHandleRemoveSignalC: make(chan struct{})}
}

func (p *PeerIdTlsCertStore) setPeerTlsCert(peerId string, tlsCert []byte) {
	p.lock.Lock()
	defer p.lock.Unlock()
	c, ok := p.store[peerId]
	if ok {
		if bytes.Compare(c, tlsCert) != 0 {
			p.store[peerId] = tlsCert
		}
	} else {
		p.store[peerId] = tlsCert
	}
}

func (p *PeerIdTlsCertStore) removeByPeerId(peerId string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.store[peerId]; ok {
		delete(p.store, peerId)
	}
}

func (p *PeerIdTlsCertStore) getCertByPeerId(peerId string) []byte {
	p.lock.Lock()
	defer p.lock.Unlock()
	if cert, ok := p.store[peerId]; ok {
		return cert
	}
	return nil
}

func (p *PeerIdTlsCertStore) startHandlingNotifyC() error {
	if err := p.handleAddNotifyC(); err != nil {
		return err
	}
	if err := p.handleRemoveNotifyC(); err != nil {
		return err
	}
	return nil
}

func (p *PeerIdTlsCertStore) stopHandling() {
	p.stopHandleAddSignalC <- struct{}{}
	p.stopHandleRemoveSignalC <- struct{}{}
}

func (p *PeerIdTlsCertStore) handleAddNotifyC() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.handlingAddCState {
		return fmt.Errorf("peerId-tlsCert store is handling")
	}
	go func() {
		for {
			if p.addC == nil {
				select {
				case <-p.stopHandleAddSignalC:
					logger.Infof("[PeerIdTlsCertStore] stop handling new notify chan.")
					return
				default:
					time.Sleep(5 * time.Second)
					continue
				}
			}

			select {
			case m := <-p.addC:
				for peerId, tlsCert := range m {
					logger.Debugf("[PeerIdTlsCertStore] recv new from notify chan(peerId:%s)", peerId)
					p.setPeerTlsCert(peerId, tlsCert)
				}
			case <-p.stopHandleAddSignalC:
				logger.Infof("[PeerIdTlsCertStore] stop handling new notify chan.")
				return
			}
		}
	}()
	p.handlingAddCState = true
	return nil
}

func (p *PeerIdTlsCertStore) handleRemoveNotifyC() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.handlingRemoveCState {
		return fmt.Errorf("peerId-tlsCert store is handling")
	}
	go func() {
		for {
			if p.removeC == nil {
				select {
				case <-p.stopHandleRemoveSignalC:
					logger.Infof("[PeerIdTlsCertStore] stop handling removed notify chan.")
					return
				default:
					time.Sleep(5 * time.Second)
					continue
				}
			}
			select {
			case peerId := <-p.removeC:
				logger.Debugf("[PeerIdTlsCertStore] recv removed notify from chan(peer-id:%s)", peerId)
				p.removeByPeerId(peerId)
			case <-p.stopHandleRemoveSignalC:
				logger.Infof("[PeerIdTlsCertStore] stop handling removed notify chan.")
				return
			}
		}
	}()
	p.handlingRemoveCState = true
	return nil
}

func (p *PeerIdTlsCertStore) storeCopy() map[string][]byte {
	p.lock.Lock()
	defer p.lock.Unlock()
	newMap := make(map[string][]byte)
	for pid := range p.store {
		bytes := p.store[pid]
		newBytes := make([]byte, len(bytes))
		copy(newBytes, bytes)
		newMap[pid] = newBytes
	}
	return newMap
}
