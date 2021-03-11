/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"errors"
	"github.com/libp2p/go-libp2p-core/network"
	"sync"
)

// DefaultStreamPoolCap is the default cap of stream pool.
const DefaultStreamPoolCap int = 100

// StreamPool is a stream pool.
type StreamPool struct {
	cap           int
	currentSize   int
	streams       chan network.Stream
	newStreamFunc func() (network.Stream, error)
	closing       bool
	closingLock   sync.RWMutex

	newStreamSignalC chan struct{}

	lock sync.RWMutex
}

func newStreamPool(cap int, newStreamFunc func() (network.Stream, error)) *StreamPool {
	if cap < 1 {
		cap = DefaultStreamPoolCap
	}
	sp := &StreamPool{
		cap:              cap,
		streams:          make(chan network.Stream, cap),
		newStreamFunc:    newStreamFunc,
		closing:          false,
		newStreamSignalC: make(chan struct{}, 5),
	}
	go sp.initStreams()
	go sp.addStreamLoop()
	return sp
}

func (sp *StreamPool) isClosing() bool {
	sp.closingLock.RLock()
	defer sp.closingLock.RUnlock()
	return sp.closing
}

func (sp *StreamPool) borrowStream() (network.Stream, error) {
	if sp.isClosing() {
		return nil, errors.New("stream pool disabled")
	}
	var stream network.Stream
	select {
	case stream = <-sp.streams:
		if stream == nil {
			return nil, errors.New("no stream can be borrowed")
		}
		go sp.expansionCheck()
		return stream, nil
	default:
		return nil, errors.New("no stream can be borrowed")
	}
}

func (sp *StreamPool) initStreams() {
	if sp.isClosing() {
		return
	}
	sp.lock.Lock()
	defer sp.lock.Unlock()
	initSize := sp.cap / 2
	if initSize > 10 {
		initSize = 10
	}
	for i := 0; i < initSize; i++ {
		stream, err := sp.newStreamFunc()
		if err != nil {
			logger.Warnf("[StreamPool.initStreams] try to create and add new stream failed, %s", err.Error())
			continue
		}
		if sp.isClosing() {
			return
		}
		sp.streams <- stream
		sp.currentSize++
		logger.Infof("[StreamPool.initStreams] try to create and add new stream success(pid:%s)", stream.Conn().RemotePeer())
	}
}

func (sp *StreamPool) expansionCheck() {
	if sp.isClosing() {
		return
	}
	sp.lock.RLock()
	defer sp.lock.RUnlock()
	if sp.currentSize*25/100 >= len(sp.streams) {
		select {
		case sp.newStreamSignalC <- struct{}{}:
		default:
		}
	}
}

func (sp *StreamPool) addStreamLoop() {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("[StreamPool.loop] recover err, %s", err)
		}
	}()
	for range sp.newStreamSignalC {
		if sp.isClosing() {
			return
		}
		sp.lock.Lock()
		if sp.currentSize < sp.cap {
			stream, err := sp.newStreamFunc()
			if err != nil {
				logger.Errorf("[StreamPool.loop] try to create and add new stream failed, %s", err.Error())
				sp.lock.Unlock()
				continue
			}
			if sp.isClosing() {
				return
			}
			select {
			case sp.streams <- stream:
				sp.currentSize++
				logger.Infof("[StreamPool.loop] try to create and add new stream success(pid:%s)", stream.Conn().RemotePeer())
				sp.lock.Unlock()
			default:
				logger.Infof("[StreamPool.loop] add stream failed[pool full], dropped(pid:%s)", stream.Conn().RemotePeer())
				sp.lock.Unlock()
				sp.dropStream(stream)
			}
			continue
		}
		sp.lock.Unlock()
	}
}

func (sp *StreamPool) addStream(stream network.Stream) {
	if sp.isClosing() {
		return
	}
	if stream == nil {
		return
	}
	sp.lock.Lock()
	select {
	case sp.streams <- stream:
		sp.currentSize++
		logger.Infof("[StreamPool] add stream success(pid:%s)", stream.Conn().RemotePeer())
		sp.lock.Unlock()
	default:
		logger.Infof("[StreamPool] add stream failed[pool full], dropped(pid:%s)", stream.Conn().RemotePeer())
		sp.lock.Unlock()
		sp.dropStream(stream)
	}
}

func (sp *StreamPool) returnStream(stream network.Stream) {
	if sp.isClosing() {
		return
	}
	if stream == nil {
		return
	}
	select {
	case sp.streams <- stream:
	default:
		logger.Infof("[StreamPool] return stream failed[pool full], dropped(pid:%s)", stream.Conn().RemotePeer())
		sp.dropStream(stream)
	}
}

func (sp *StreamPool) dropStream(stream network.Stream) {
	logger.Debugf("[StreamPool] before drop stream (currentSize:%d , pid:%s)", sp.currentSize, stream.Conn().RemotePeer())
	if sp.isClosing() {
		return
	}
	if stream == nil {
		return
	}
	sp.lock.Lock()
	defer sp.lock.Unlock()
	sp.currentSize--
	_ = stream.Reset()
}

func (sp *StreamPool) getCurrentSize() int {
	sp.lock.RLock()
	defer sp.lock.RUnlock()
	return int(sp.currentSize)
}

func (sp *StreamPool) cleanAndDisable() {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	sp.closingLock.Lock()
	defer sp.closingLock.Unlock()
	sp.closing = true
	sp.cap = 0
	sp.currentSize = 0
	close(sp.streams)
	close(sp.newStreamSignalC)
}
