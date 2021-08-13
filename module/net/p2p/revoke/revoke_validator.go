/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package revoke

import (
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"sync"

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

// ValidateMemberStatus check the status of members.
func (rv *RevokedValidator) ValidateMemberStatus(members []*pbac.Member) (bool, error) {
	bl := true
	var err error
	rv.accessControls.Range(func(key, value interface{}) bool {
		ac, _ := value.(protocol.AccessControlProvider)
		if ac == nil {
			return false
		}
		allOk := true
		for _, member := range members {
			var s pbac.MemberStatus
			s, err = ac.GetMemberStatus(member)
			if err != nil {
				return false
			}
			if s == pbac.MemberStatus_INVALID || s == pbac.MemberStatus_FROZEN || s == pbac.MemberStatus_REVOKED {
				allOk = false
				break
			}
		}
		if allOk {
			bl = false
			return false
		}
		return true
	})
	if err != nil {
		return false, err
	}
	return bl, nil
}
