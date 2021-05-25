/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	"errors"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorHostNotRunning
var ErrorHostNotRunning = errors.New("libp2p libP2pHost is not running")

// ErrorPubSubNotRunning
var ErrorPubSubNotRunning = errors.New("libp2p gossip-sub is not running")

// LibP2pPubSub is a pub-sub service implementation.
type LibP2pPubSub struct {
	topicLock sync.Mutex
	topicMap  map[string]*pubsub.Topic // topicMap mapping topic name to Topic .

	libP2pHost           *LibP2pHost    // libP2pHost is LibP2pHost instance.
	pubsubUid            string         // pubsubUid is the unique id of pubsub.
	pubsub               *pubsub.PubSub // pubsub is a pubsub.PubSub instance.
	pubSubMaxMessageSize int            // pubSubMaxMessageSize is the value for MaxMessageSize option.
	startUp              int32          // startUp is the flag of the state of LibP2pPubSub. 0 not start, 1 starting, 2 started
}

//NewPubsub create a new LibP2pPubSub instance.
func NewPubsub(pubsubUid string, host *LibP2pHost, maxMessageSize int) (*LibP2pPubSub, error) {
	ps := &LibP2pPubSub{
		topicMap: make(map[string]*pubsub.Topic),

		libP2pHost:           host,
		pubsubUid:            pubsubUid,
		pubSubMaxMessageSize: maxMessageSize,
		startUp:              0,
	}
	return ps, nil
}

// isSubscribed 是否已订阅某个topic
func (ps *LibP2pPubSub) isSubscribed(topic string) bool {
	_, ok := ps.topicMap[topic]
	return ok
}

// GetTopic get a topic with the name given.
func (ps *LibP2pPubSub) GetTopic(name string) (*pubsub.Topic, error) {
	if atomic.LoadInt32(&ps.startUp) < 2 {
		return nil, ErrorPubSubNotRunning
	}
	ps.topicLock.Lock()
	t, ok := ps.topicMap[name]
	ps.topicLock.Unlock()
	if !ok || t == nil {
		topic, err := ps.pubsub.Join(name)
		if err != nil {
			return nil, err
		}
		ps.topicMap[name] = topic
		t = topic
	}
	return t, nil
}

// Subscribe a topic.
func (ps *LibP2pPubSub) Subscribe(topic string) (*pubsub.Subscription, error) {
	t, err := ps.GetTopic(topic)
	if err != nil {
		return nil, err
	}
	logger.Infof("[PubSub] gossip-sub subscribe topic[%s].", topic)
	return t.Subscribe()
}

// Publish a msg to the topic.
func (ps *LibP2pPubSub) Publish(topic string, data []byte) error {
	logger.Debugf("[PubSub] publish msg to topic[%s]", topic)
	t, err := ps.GetTopic(topic)
	if err != nil {
		return err
	}
	return t.Publish(ps.libP2pHost.ctx, data)
}

// Start
func (ps *LibP2pPubSub) Start() error {
	if !ps.libP2pHost.IsRunning() {
		logger.Errorf("[PubSub] gossip-sub service can not start. start host first pls.")
		return ErrorHostNotRunning
	}
	if atomic.LoadInt32(&ps.startUp) > 0 {
		logger.Warnf("[PubSub] gossip-sub service[%s] is running.", ps.pubsubUid)
		return nil
	}
	atomic.StoreInt32(&ps.startUp, 1)
	logger.Infof("[PubSub] gossip-sub service[%s] starting... ", ps.pubsubUid)
	pss, err := pubsub.NewGossipSub(
		ps.libP2pHost.ctx,
		ps.libP2pHost.host,
		pubsub.WithUid(ps.pubsubUid),
		pubsub.WithMaxMessageSize(ps.pubSubMaxMessageSize),
	)
	if err != nil {
		return err
	}
	ps.pubsub = pss
	atomic.StoreInt32(&ps.startUp, 2)
	logger.Infof("[PubSub] gossip-sub service[%s] started. ", ps.pubsubUid)
	return nil
}

// AddWhitelistPeer add a peer.ID to pubsub white list.
func (ps *LibP2pPubSub) AddWhitelistPeer(pid peer.ID) error {
	switch atomic.LoadInt32(&ps.startUp) {
	case 0:
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if atomic.LoadInt32(&ps.startUp) != 1 {
				ps.pubsub.AddWhitelistPeer(pid)
				return nil
			}
		}
		return ErrorPubSubNotRunning
	case 1:
		for {
			time.Sleep(500 * time.Millisecond)
			if atomic.LoadInt32(&ps.startUp) != 1 {
				ps.pubsub.AddWhitelistPeer(pid)
				return nil
			}
		}
	case 2:
		ps.pubsub.AddWhitelistPeer(pid)
	default:

	}
	return nil
}

// TryToReloadPeer try to reload peer as new peer.
func (ps *LibP2pPubSub) TryToReloadPeer(pid peer.ID) error {
	switch atomic.LoadInt32(&ps.startUp) {
	case 0:
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if atomic.LoadInt32(&ps.startUp) != 1 {
				ps.pubsub.TryToReloadPeer(pid)
				return nil
			}
		}
		return ErrorPubSubNotRunning
	case 1:
		for {
			time.Sleep(500 * time.Millisecond)
			if atomic.LoadInt32(&ps.startUp) != 1 {
				ps.pubsub.TryToReloadPeer(pid)
				return nil
			}
		}
	case 2:
		ps.pubsub.TryToReloadPeer(pid)
	default:

	}
	return nil
}

// RemoveWhitelistPeer remove a peer.ID to pubsub white list.
func (ps *LibP2pPubSub) RemoveWhitelistPeer(pid peer.ID) error {
	switch atomic.LoadInt32(&ps.startUp) {
	case 0:
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if atomic.LoadInt32(&ps.startUp) == 2 {
				ps.pubsub.RemoveWhitelistPeer(pid)
				return nil
			}
		}
		return ErrorPubSubNotRunning
	case 1:
		for {
			time.Sleep(500 * time.Millisecond)
			if atomic.LoadInt32(&ps.startUp) != 1 {
				ps.pubsub.RemoveWhitelistPeer(pid)
				return nil
			}
		}
	case 2:
		ps.pubsub.RemoveWhitelistPeer(pid)
	default:

	}
	return nil
}
