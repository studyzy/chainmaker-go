/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package libp2pgmtls

import (
	cmx509 "chainmaker.org/chainmaker/common/crypto/x509"
	"fmt"
	"sync"
)

// ChainTrustRoots keep the trust root cert pools and the trust intermediates cert pools of all chains.
type ChainTrustRoots struct {
	lock               sync.Mutex
	trustRoots         map[string]*cmx509.CertPool
	trustIntermediates map[string]*cmx509.CertPool
}

// NewChainTrustRoots create a new ChainTrustRoots instance.
func NewChainTrustRoots() *ChainTrustRoots {
	return &ChainTrustRoots{trustRoots: make(map[string]*cmx509.CertPool), trustIntermediates: make(map[string]*cmx509.CertPool)}
}

// RootsPool return the trust root cert pool of the chain which id is the id given.
func (ctr *ChainTrustRoots) RootsPool(chainId string) (*cmx509.CertPool, bool) {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	cp, ok := ctr.trustRoots[chainId]
	return cp, ok
}

// AddRoot add a trust root cert to cert pool.
func (ctr *ChainTrustRoots) AddRoot(chainId string, root *cmx509.Certificate) {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	cp, ok := ctr.trustRoots[chainId]
	if !ok {
		ctr.trustRoots[chainId] = cmx509.NewCertPool()
		cp = ctr.trustRoots[chainId]
	}
	cp.AddCert(root)
}

// AppendRootsFromPem append trust root certs from pem bytes to cert pool.
func (ctr *ChainTrustRoots) AppendRootsFromPem(chainId string, rootPem []byte) bool {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	cp, ok := ctr.trustRoots[chainId]
	if !ok {
		ctr.trustRoots[chainId] = cmx509.NewCertPool()
		cp = ctr.trustRoots[chainId]
	}
	return cp.AppendCertsFromPEM(rootPem)
}

// RefreshRootsFromPem reset all trust root certs from pem bytes array to cert pool.
func (ctr *ChainTrustRoots) RefreshRootsFromPem(chainId string, rootsPem [][]byte) bool {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	newCertPool := cmx509.NewCertPool()
	for _, bytes := range rootsPem {
		if !newCertPool.AppendCertsFromPEM(bytes) {
			return false
		}
	}
	ctr.trustRoots[chainId] = newCertPool
	return true
}

// IntermediatesPool return the trust intermediates cert pool of the chain which id is the id given.
func (ctr *ChainTrustRoots) IntermediatesPool(chainId string) (*cmx509.CertPool, bool) {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	cp, ok := ctr.trustIntermediates[chainId]
	return cp, ok
}

// AddIntermediates add a trust intermediates cert to cert pool.
func (ctr *ChainTrustRoots) AddIntermediates(chainId string, intermediates *cmx509.Certificate) {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	cp, ok := ctr.trustIntermediates[chainId]
	if !ok {
		ctr.trustIntermediates[chainId] = cmx509.NewCertPool()
		cp = ctr.trustIntermediates[chainId]
	}
	cp.AddCert(intermediates)
}

// AppendIntermediatesFromPem append trust intermediates certs from pem bytes to cert pool.
func (ctr *ChainTrustRoots) AppendIntermediatesFromPem(chainId string, intermediatesPem []byte) bool {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	cp, ok := ctr.trustIntermediates[chainId]
	if !ok {
		ctr.trustIntermediates[chainId] = cmx509.NewCertPool()
		cp = ctr.trustIntermediates[chainId]
	}
	return cp.AppendCertsFromPEM(intermediatesPem)
}

// RefreshIntermediatesFromPem reset all trust intermediates certs from pem bytes array to cert pool.
func (ctr *ChainTrustRoots) RefreshIntermediatesFromPem(chainId string, intermediatesPem [][]byte) bool {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	newCertPool := cmx509.NewCertPool()
	for _, bytes := range intermediatesPem {
		if !newCertPool.AppendCertsFromPEM(bytes) {
			return false
		}
	}
	ctr.trustIntermediates[chainId] = newCertPool
	return true
}

// VerifyCert verify the cert given. If ok, return chain id list.
func (ctr *ChainTrustRoots) VerifyCert(cert *cmx509.Certificate) ([]string, error) {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	if cert == nil {
		return nil, fmt.Errorf("cert is nil")
	}
	chainIds := make([]string, 0)
	var err error
	for chainId, certPool := range ctr.trustRoots {
		vo := cmx509.VerifyOptions{Roots: certPool}
		trustIntermediates, ok := ctr.trustIntermediates[chainId]
		if ok {
			vo.Intermediates = trustIntermediates
		}
		if _, e := cert.Verify(vo); e != nil {
			err = e
			continue
		}
		chainIds = append(chainIds, chainId)
	}
	if len(chainIds) == 0 {
		return nil, fmt.Errorf("certificate verification failed: %s", err)
	}
	return chainIds, nil
}

// VerifyCertOfChain verify the cert given with chainId. If ok, return true.
func (ctr *ChainTrustRoots) VerifyCertOfChain(chainId string, cert *cmx509.Certificate) bool {
	ctr.lock.Lock()
	defer ctr.lock.Unlock()
	if cert == nil {
		return false
	}
	certPool, ok := ctr.trustRoots[chainId]
	if !ok {
		return false
	}
	vo := cmx509.VerifyOptions{Roots: certPool}
	trustIntermediates, ok := ctr.trustIntermediates[chainId]
	if ok {
		vo.Intermediates = trustIntermediates
	}
	if _, err := cert.Verify(vo); err != nil {
		return false
	}
	return true
}
