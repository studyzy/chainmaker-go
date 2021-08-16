/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package libp2pgmtls

import (
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
)

import (
	cmtls "chainmaker.org/chainmaker/common/crypto/tls"
)

type conn struct {
	*cmtls.Conn

	localPeer peer.ID
	privKey   ci.PrivKey

	remotePeer   peer.ID
	remotePubKey ci.PubKey
}

var _ sec.SecureConn = &conn{}

func (c *conn) LocalPeer() peer.ID {
	return c.localPeer
}

func (c *conn) LocalPrivateKey() ci.PrivKey {
	return c.privKey
}

func (c *conn) RemotePeer() peer.ID {
	return c.remotePeer
}

func (c *conn) RemotePublicKey() ci.PubKey {
	return c.remotePubKey
}
