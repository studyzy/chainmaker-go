/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	tlsCertIdFlag     = ".tls."
	signCertIdFlag    = ".sign."
	signCertIdPattern = ".+\\.sign\\..+"
)

// CertIdPeerIdMapper mapped cert id with peer id.
type CertIdPeerIdMapper struct {
	lock                    sync.RWMutex
	mapper                  map[string]string
	addC                    <-chan string
	removeC                 <-chan string
	addCHandling            bool
	removeCHandling         bool
	stopHandleAddSignalC    chan struct{}
	stopHandleRemoveSignalC chan struct{}
}

// newCertIdPeerIdMapper create a new CertIdPeerIdMapper instance.
func newCertIdPeerIdMapper(addC <-chan string, removeC <-chan string) *CertIdPeerIdMapper {
	return &CertIdPeerIdMapper{mapper: make(map[string]string), addC: addC, removeC: removeC, stopHandleAddSignalC: make(chan struct{}), stopHandleRemoveSignalC: make(chan struct{})}
}

// add a record mapping cert id with peer id.
func (c *CertIdPeerIdMapper) add(certId string, peerId string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if _, ok := c.mapper[certId]; ok {
		return
	}
	c.mapper[certId] = peerId
}

// removeByPeerId remove all records mapped with given peerId.
func (c *CertIdPeerIdMapper) removeByPeerId(peerId string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for certId, pid := range c.mapper {
		if pid == peerId {
			delete(c.mapper, certId)
		}
	}
}

// findPeerIdByCertId will return a peer id if the given cert id has mapped with a peer id .
func (c *CertIdPeerIdMapper) findPeerIdByCertId(certId string) (string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	certId = parseSignCertIdToTlsCertId(certId)
	peerId, ok := c.mapper[certId]
	if !ok {
		logger.Debugf("cert id not mapping(certId:%s)", certId)
		return "", errors.New("cert id not mapping")
	}
	return peerId, nil
}

func parseSignCertIdToTlsCertId(certId string) string {
	ok, err := regexp.Match(signCertIdPattern, []byte(certId))
	if err != nil {
		return certId
	}
	if ok {
		return strings.ReplaceAll(certId, signCertIdFlag, tlsCertIdFlag)
	}
	return certId
}

// stopHandling stop all the handling work.
func (c *CertIdPeerIdMapper) stopHandling() {
	c.stopHandleAddSignalC <- struct{}{}
	c.stopHandleRemoveSignalC <- struct{}{}
}

// handleNewTlsCertIdPeerIdNotifyC start to handle addC chan.
func (c *CertIdPeerIdMapper) handleNewTlsCertIdPeerIdNotifyC() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.addCHandling {
		return fmt.Errorf("certid-peerid mapper is handling")
	}
	go func() {
		for {
			if c.addC == nil {
				select {
				case <-c.stopHandleAddSignalC:
					logger.Infof("[PeerIdTlsCertStore] stop handling new peer notify chan.")
					return
				default:
					time.Sleep(5 * time.Second)
					continue
				}
			}

			select {
			case notifyStr := <-c.addC:
				s := strings.Split(notifyStr, "<-->")
				certId := s[0]
				peerId := s[1]
				logger.Debugf("[CertIdPeerIdMapper] recv new from notify chan(certId:%s, peerId:%s)", certId, peerId)
				c.add(certId, peerId)
			case <-c.stopHandleAddSignalC:
				logger.Infof("[CertIdPeerIdMapper] stop handling new peer notify chan.")
				return
			}
		}
	}()
	c.addCHandling = true
	return nil
}

// handleRemoveTlsPeerNotifyC start to handle removeC chan.
func (c *CertIdPeerIdMapper) handleRemoveTlsPeerNotifyC() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.removeCHandling {
		return fmt.Errorf("peer chainid recorder is handling")
	}
	go func() {
		for {
			if c.removeC == nil {
				select {
				case <-c.stopHandleRemoveSignalC:
					logger.Infof("[PeerIdTlsCertStore] stop handling removed peer notify chan.")
					return
				default:
					time.Sleep(5 * time.Second)
					continue
				}
			}

			select {
			case peerId := <-c.removeC:
				logger.Debugf("[CertIdPeerIdMapper] recv removed peer-id from notify chan(peer-id:%s)", peerId)
				c.removeByPeerId(peerId)
			case <-c.stopHandleRemoveSignalC:
				logger.Infof("[CertIdPeerIdMapper] stop handling removed peer notify chan.")
				return
			}
		}
	}()
	c.removeCHandling = true
	return nil
}
