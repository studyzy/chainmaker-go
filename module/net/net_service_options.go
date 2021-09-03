/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import "chainmaker.org/chainmaker/common/v2/msgbus"

// NetServiceOption is a net service option.
type NetServiceOption func(ns *NetService) error

// Apply the net service options given.
func (ns *NetService) Apply(opts ...NetServiceOption) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(ns); err != nil {
			return err
		}
	}
	return nil
}

// WithMsgBus set msg-bus.
func WithMsgBus(msgBus msgbus.MessageBus) NetServiceOption {
	return func(ns *NetService) error {
		return ns.bindMsgBus(msgBus)
	}
}

// WithConsensusNodeUid set the consensus node id list for net service.
// This list will be used for broadcast consensus msg to consensus nodes.
func WithConsensusNodeUid(consensusNodeUid ...string) NetServiceOption {
	return func(ns *NetService) error {
		ns.consensusNodeIdsLock.Lock()
		defer ns.consensusNodeIdsLock.Unlock()
		for _, nodeUid := range consensusNodeUid {
			ns.consensusNodeIds[nodeUid] = struct{}{}
		}
		return nil
	}
}
