/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package libp2pgmtls

import (
	"context"
	gocrypto "crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"net"

	"chainmaker.org/chainmaker-go/net/p2p/revoke"
	cmx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmtls"
)

// ID is the protocol ID (used when negotiating with multistream)
const ID = "/gmtls/1.0.0"

// Transport constructs secure communication sessions for a peer.
type Transport struct {
	config *gmtls.Config

	privKey   crypto.PrivKey
	localPeer peer.ID

	revokeValidator *revoke.RevokedValidator
}

func getAllCertsBytes(source []byte) [][]byte {
	result := make([][]byte, 0)
	if source == nil {
		return nil
	}
	for len(source) > 0 {
		var block *pem.Block
		block, source = pem.Decode(source)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}
		result = append(result, block.Bytes)
	}
	return result
}

// BuildTlsTrustRoots build the cert pool with cert bytes of chain.
func BuildTlsTrustRoots(chainTrustRoots map[string][][]byte) (*ChainTrustRoots, error) {
	tlsTrustRoots := NewChainTrustRoots()
	for chainId, trustRootCertBytes := range chainTrustRoots {
		for _, certByte := range trustRootCertBytes {
			ok, err := loadAllCertsFromCertBytes(certByte, chainId, tlsTrustRoots)
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
		}
	}
	return tlsTrustRoots, nil
}

// AppendNewCertsToTrustRoots add a new cert to ChainTrustRoots.
func AppendNewCertsToTrustRoots(tlsTrustRoots *ChainTrustRoots, chainId string, certPemBytes []byte) (bool, error) {
	return loadAllCertsFromCertBytes(certPemBytes, chainId, tlsTrustRoots)
}

func loadAllCertsFromCertBytes(certByte []byte, chainId string, tlsTrustRoots *ChainTrustRoots) (ok bool, err error) {
	// 1. read all certs from bytes
	allCertsBytes := getAllCertsBytes(certByte)
	// 2. add certs to pool
	if allCertsBytes == nil || len(allCertsBytes) == 0 {
		return false, nil
	}
	for _, cert := range allCertsBytes {
		c, err := sm2.ParseCertificate(cert)
		if err != nil {
			return false, err
		}
		if c.IsCA {
			tlsTrustRoots.AddRoot(chainId, c)
		} else {
			tlsTrustRoots.AddIntermediates(chainId, c)
		}
	}
	return true, nil
}

var _ sec.SecureTransport = &Transport{}

// New return a function can create a new Transport instance.
func New(
	keyBytes,
	certBytes []byte,
	tlsTrustRoots *ChainTrustRoots,
	revokeValidator *revoke.RevokedValidator,
	newTlsPeerChainIdsNotifyC chan<- map[string][]string,
	newTlsCertIdPeerIdNotifyC chan<- string,
	addPeerIdTlsCertNotifyC chan<- map[string][]byte,
) func(key crypto.PrivKey) (*Transport, error) {
	return func(key crypto.PrivKey) (*Transport, error) {
		certificate, err := gmtls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, err
		}

		id, err := peer.IDFromPrivateKey(key)
		if err != nil {
			return nil, err
		}
		addPeerIdTlsCertNotifyC <- map[string][]byte{id.Pretty(): certificate.Certificate[0]}
		return &Transport{
			config: &gmtls.Config{
				Certificates:          []gmtls.Certificate{certificate},
				InsecureSkipVerify:    true,
				ClientAuth:            gmtls.RequireAnyClientCert,
				VerifyPeerCertificate: createVerifyPeerCertificateFunc(tlsTrustRoots, revokeValidator, newTlsPeerChainIdsNotifyC, newTlsCertIdPeerIdNotifyC, addPeerIdTlsCertNotifyC),
			},
			privKey:         key,
			localPeer:       id,
			revokeValidator: revokeValidator,
		}, nil
	}
}

func createVerifyPeerCertificateFunc(
	tlsTrustRoots *ChainTrustRoots,
	revokeValidator *revoke.RevokedValidator,
	newTlsPeerChainIdsNotifyC chan<- map[string][]string,
	newTlsCertIdPeerIdNotifyC chan<- string,
	addPeerIdTlsCertNotifyC chan<- map[string][]byte,
) func(rawCerts [][]byte, _ [][]*sm2.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*sm2.Certificate) error {
		revoked, err := isRevoked(revokeValidator, rawCerts)
		if err != nil {
			return err
		}
		if revoked {
			return fmt.Errorf("certificate revoked")
		}
		tlsCertBytes := rawCerts[0]
		cert, err := sm2.ParseCertificate(tlsCertBytes)
		if err != nil {
			return fmt.Errorf("parse certificate failed: %s", err.Error())
		}
		chainIds, err := tlsTrustRoots.VerifyCert(cert)
		if err != nil {
			return fmt.Errorf("verify certificate failed: %s", err.Error())
		}
		pubKey, err := parsePublicKeyToPubKey(cert.PublicKey)
		if err != nil {
			return fmt.Errorf("parse pubkey failed: %s", err.Error())
		}
		pid, err := peer.IDFromPublicKey(pubKey)
		if err != nil {
			return fmt.Errorf("parse pid from pubkey failed: %s", err.Error())
		}
		peerId := pid.Pretty()
		certId, err := cmx509.GetNodeIdFromSm2Certificate(cmx509.OidNodeId, *cert)
		if err != nil {
			return fmt.Errorf("get certid failed: %s", err.Error())
		}

		newTlsPeerChainIdsNotifyC <- map[string][]string{peerId: chainIds}
		newTlsCertIdPeerIdNotifyC <- string(certId) + "<-->" + peerId
		addPeerIdTlsCertNotifyC <- map[string][]byte{peerId: tlsCertBytes}
		return nil
	}
}

func isRevoked(revokeValidator *revoke.RevokedValidator, rawCerts [][]byte) (bool, error) {
	//certs := make([]*cmx509.Certificate, 0)
	//for idx := range rawCerts {
	//	cert, err := cmx509.ParseCertificate(rawCerts[idx])
	//	if err != nil {
	//		return false, err
	//	}
	//	certs = append(certs, cert)
	//}
	//return revokeValidator.ValidateCertsIsRevoked(certs), nil
	members := make([]*pbac.Member, 0)
	for idx := range rawCerts {
		m := &pbac.Member{
			OrgId:      "",
			MemberType: pbac.MemberType_CERT,
			MemberInfo: rawCerts[idx],
		}
		members = append(members, m)
	}
	ok, err := revokeValidator.ValidateMemberStatus(members)
	return !ok, err
}

// SecureInbound runs the TLS handshake as a server.
func (t *Transport) SecureInbound(ctx context.Context, insecure net.Conn) (sec.SecureConn, error) {
	conn := gmtls.Server(insecure, t.config.Clone())
	if err := conn.Handshake(); err != nil {
		insecure.Close()
		return nil, err
	}

	remotePubKey, err := t.getPeerPubKey(conn)
	if err != nil {
		return nil, err
	}

	return t.setupConn(conn, remotePubKey)
}

// SecureOutbound runs the TLS handshake as a client.
func (t *Transport) SecureOutbound(ctx context.Context, insecure net.Conn, p peer.ID) (sec.SecureConn, error) {
	conn := gmtls.Client(insecure, t.config.Clone())
	if err := conn.Handshake(); err != nil {
		insecure.Close()
		return nil, err
	}

	remotePubKey, err := t.getPeerPubKey(conn)
	if err != nil {
		return nil, err
	}

	return t.setupConn(conn, remotePubKey)
}

func (t *Transport) getPeerPubKey(conn *gmtls.Conn) (crypto.PubKey, error) {
	state := conn.ConnectionState()
	if len(state.PeerCertificates) <= 0 {
		return nil, errors.New("expected one certificates in the chain")
	}

	pubKey, err := parsePublicKeyToPubKey(state.PeerCertificates[0].PublicKey)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling public key failed: %s", err)
	}
	return pubKey, err
}

func (t *Transport) setupConn(tlsConn *gmtls.Conn, remotePubKey crypto.PubKey) (sec.SecureConn, error) {
	remotePeerID, err := peer.IDFromPublicKey(remotePubKey)
	if err != nil {
		return nil, err
	}

	return &conn{
		Conn:         tlsConn,
		localPeer:    t.localPeer,
		privKey:      t.privKey,
		remotePeer:   remotePeerID,
		remotePubKey: remotePubKey,
	}, nil
}

func parsePublicKeyToPubKey(publicKey gocrypto.PublicKey) (crypto.PubKey, error) {
	switch p := publicKey.(type) {
	case *ecdsa.PublicKey:
		if p.Curve == sm2.P256Sm2() {
			b, err := sm2.MarshalPKIXPublicKey(p)
			if err != nil {
				return nil, err
			}
			pub, err := sm2.ParseSm2PublicKey(b)
			if err != nil {
				return nil, err
			}
			return crypto.NewSM2PublicKey(pub), nil
		}
		return crypto.NewECDSAPublicKey(p), nil
	case *sm2.PublicKey:
		return crypto.NewSM2PublicKey(p), nil
	case *rsa.PublicKey:
		return crypto.NewRsaPublicKey(*p), nil
	}
	return nil, errors.New("unsupported public key type")
}
