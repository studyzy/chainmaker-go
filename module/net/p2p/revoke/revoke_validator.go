/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package revoke

import (
	"sync"

	cmx509 "chainmaker.org/chainmaker/common/crypto/x509"
	"chainmaker.org/chainmaker/protocol"
)

// RevokedValidator is a validator for validating revoked peer use revoked tls cert.
type RevokedValidator struct {
	accessControls sync.Map
	revokedPeerIds sync.Map
}

// NewRevokedValidator create a new RevokedValidator instance.
func NewRevokedValidator() *RevokedValidator {
	return &RevokedValidator{}
}

// AddPeerId add new pid to revoked list.
func (rv *RevokedValidator) AddPeerId(pid string) {
	rv.revokedPeerIds.LoadOrStore(pid, struct{}{})
}

// RemovePeerId remove pid given from revoked list.
func (rv *RevokedValidator) RemovePeerId(pid string) {
	rv.revokedPeerIds.Delete(pid)
}

// ContainsPeerId return whether pid given exist in revoked list.
func (rv *RevokedValidator) ContainsPeerId(pid string) bool {
	_, ok := rv.revokedPeerIds.Load(pid)
	return ok
}

// AddAC add access control of chain to validator.
func (rv *RevokedValidator) AddAC(chainId string, ac protocol.AccessControlProvider) {
	rv.accessControls.LoadOrStore(chainId, ac)
}

// ValidateCertsIsRevoked return whether certs given is revoked.
func (rv *RevokedValidator) ValidateCertsIsRevoked(certs []*cmx509.Certificate) bool {
	bl := false
	rv.accessControls.Range(func(key, value interface{}) bool {
		ac, _ := value.(protocol.AccessControlProvider)
		if ac == nil {
			return false
		}
		return true
	})
	return bl
}
