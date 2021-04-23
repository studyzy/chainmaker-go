/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"chainmaker.org/chainmaker-go/protocol"
)

// NetServiceFactory is a net service instance factory.
type NetServiceFactory struct {
}

// NewNetService create a new net service instance.
func (nsf *NetServiceFactory) NewNetService(net Net, chainId string, ac protocol.AccessControlProvider, chainConf protocol.ChainConf, opts ...NetServiceOption) (protocol.NetService, error) {
	//初始化工厂实例
	ns := NewNetService(chainId, net, ac)
	if err := ns.Apply(opts...); err != nil {
		return nil, err
	}
	if chainConf != nil {
		if err := nsf.setAllConsensusNodeIds(ns, chainConf); err != nil {
			return nil, err
		}
		if err := nsf.setAllTlsTrustRoots(ns, chainConf); err != nil {
			return nil, err
		}
		// set config watcher
		chainConf.AddWatch(ns.ConfigWatcher())
		// set vm watcher
		chainConf.AddVmWatch(ns.VmWatcher())
	}
	return ns, nil
}

func (nsf *NetServiceFactory) setAllConsensusNodeIds(ns *NetService, chainConf protocol.ChainConf) error {
	consensusNodeUidList := make([]string, 0)
	// add all the seeds
	for _, node := range chainConf.ChainConfig().Consensus.Nodes {
		for _, nodeUid := range node.NodeId {
			consensusNodeUidList = append(consensusNodeUidList, nodeUid)
		}
	}
	// set all consensus node id for net service
	err := ns.Apply(WithConsensusNodeUid(consensusNodeUidList...))
	if err != nil {
		return err
	}
	ns.logger.Infof("[NetServiceFactory] set consensus node uid list ok(chain-id:%s)", ns.chainId)
	return nil
}

func (nsf *NetServiceFactory) setAllTlsTrustRoots(ns *NetService, chainConf protocol.ChainConf) error {
	// set all tls trust root certs
	for _, root := range chainConf.ChainConfig().TrustRoots {
		if err := ns.localNet.AddTrustRoot(ns.chainId, []byte(root.Root)); err != nil {
			return err
		}
	}
	ns.logger.Infof("[NetServiceFactory] add trust root certs ok(chain-id:%s)", ns.chainId)
	// check whether peers already connected contains to this chain
	ns.localNet.ReVerifyTrustRoots(chainConf.ChainConfig().ChainId)
	return nil
}
