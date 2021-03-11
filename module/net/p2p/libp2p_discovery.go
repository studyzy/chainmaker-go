/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package p2p

import (
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

// SetupDiscovery setup a discovery service.
func SetupDiscovery(host *LibP2pHost, enableDHTBootstrapProvider bool, bootstraps []string) error {
	logger.Info("[Discovery] discovery setting...")
	bootstrapAddrInfos, err := ParseAddrInfo(bootstraps)
	if err != nil {
		return err
	}
	var mode dht.ModeOpt
	// is enable bootstrap mode
	if enableDHTBootstrapProvider {
		logger.Info("[Discovery] dht will be created with server-mode.")
		mode = dht.ModeServer
	} else {
		logger.Info("[Discovery] dht will be created with client-mode.")
		mode = dht.ModeClient
	}

	options := []dht.Option{dht.Mode(mode)}
	//if len(bootstraps) > 0 {
	//	options = append(options, dht.BootstrapPeers(bootstraps...))
	//}
	ctx := host.Context()
	h := host.Host()
	// new kademlia DHT
	kademliaDHT, err := dht.New(
		ctx,
		h,
		options...)
	if err != nil {
		logger.Infof("[Discovery] create dht failed,%s", err.Error())
		return err
	}
	// set as bootstrap
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return err
	}
	// new ConnSupervisor
	host.connSupervisor = newConnSupervisor(host, bootstrapAddrInfos)
	// start supervising.
	host.connSupervisor.startSupervising()
	// announce self
	logger.Info("[Discovery] announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, DefaultLibp2pServiceTag)
	logger.Info("[Discovery] successfully announced!")
	// start to find other peers
	logger.Info("[Discovery] searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, DefaultLibp2pServiceTag)
	if err != nil {
		return err
	}
	// find new peer and make connection
	host.connSupervisor.handleChanNewPeerFound(peerChan)
	logger.Info("[Discovery] discovery set up.")
	return nil
}
