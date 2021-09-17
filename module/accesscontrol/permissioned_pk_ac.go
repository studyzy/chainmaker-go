/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker-go/localconf"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"encoding/hex"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"fmt"
	"sync"
	"sync/atomic"
)

var _ protocol.AccessControlProvider = (*permissionedPkACProvider)(nil)

var NilPermissionedPkACProvider ACProvider = (*permissionedPkACProvider)(nil)

type permissionedPkACProvider struct {
	//chainconfig authType
	authType AuthType

	acService *accessControlService

	hashType string

	log protocol.Logger

	localOrg string

	adminMember *sync.Map

	consensusMember *sync.Map
}

// all-in-one validation for signing members: certificate chain/whitelist, signature, policies
func (cp *permissionedPkACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
	endorsements := principal.GetEndorsement()
	msg := principal.GetMessage()
	refinedEndorsement := cp.refineEndorsements(endorsements, msg)
	if len(refinedEndorsement) <= 0 {
		return nil, fmt.Errorf("refine endorsements failed, all endorsers have failed verification")
	}

	refinedPrincipal, err := cp.CreatePrincipal(principal.GetResourceName(), refinedEndorsement, msg)
	if err != nil {
		return nil, fmt.Errorf("create principal failed: [%s]", err.Error())
	}

	return refinedPrincipal, nil
}

func (cp *permissionedPkACProvider) refineEndorsements(endorsements []*common.EndorsementEntry,
	msg []byte) []*common.EndorsementEntry {

	refinedSigners := map[string]bool{}
	var refinedEndorsement []*common.EndorsementEntry
	var memInfo string

	for _, endorsementEntry := range endorsements {
		endorsement := &common.EndorsementEntry{
			Signer: &pbac.Member{
				OrgId:      endorsementEntry.Signer.OrgId,
				MemberInfo: endorsementEntry.Signer.MemberInfo,
				MemberType: endorsementEntry.Signer.MemberType,
			},
			Signature: endorsementEntry.Signature,
		}
		if endorsement.Signer.MemberType == pbac.MemberType_PUBLIC_KEY {
			cp.log.Debugf("target endorser uses public key")
			memInfo = string(endorsement.Signer.MemberInfo)
		} else {
			cp.log.Errorf("member type error")
			continue
		}

		remoteMember, err := cp.acService.newMember(endorsement.Signer)
		if err != nil {
			err = fmt.Errorf("new member failed: [%s]", err.Error())
			continue
		}

		if err := remoteMember.Verify(cp.hashType, msg, endorsement.Signature); err != nil {
			err = fmt.Errorf("signer member verify signature failed: [%s]", err.Error())
			cp.log.Debugf("information for invalid signature:\norganization: %s\npubkey: %s\nmessage: %s\n"+
				"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
			continue
		}

		if _, ok := refinedSigners[memInfo]; !ok {
			refinedSigners[memInfo] = true
			refinedEndorsement = append(refinedEndorsement, endorsement)
		}
	}
	return refinedEndorsement
}

// GetHashAlg return hash algorithm the access control provider uses
func (cp *permissionedPkACProvider) GetHashAlg() string {
	return cp.hashType
}

func (cp *permissionedPkACProvider) NewMember(member *pbac.Member) (protocol.Member, error) {
	if member.MemberType == pbac.MemberType_PUBLIC_KEY {
		return newMemberFromPkPem(member.GetOrgId(),  "", string(member.MemberInfo), cp.hashType)
	}
	return nil, fmt.Errorf("new member for permissionedPk failed, member type error")
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (cp *permissionedPkACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return cp.acService.validateResourcePolicy(resourcePolicy)
}

// CreatePrincipalForTargetOrg creates a principal for "SELF" type principal,
// which needs to convert SELF to a sepecific organization id in one authentication
func (cp *permissionedPkACProvider) CreatePrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte,
	targetOrgId string) (protocol.Principal, error) {
	return cp.acService.createPrincipalForTargetOrg(resourceName, endorsements, message, targetOrgId)
}

// CreatePrincipal creates a principal for one time authentication
func (cp *permissionedPkACProvider) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (
	protocol.Principal, error) {
	return cp.acService.createPrincipal(resourceName, endorsements, message)
}

func (cp *permissionedPkACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	return cp.acService.lookUpPolicy(resourceName)
}

func (cp *permissionedPkACProvider) LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	return cp.acService.lookUpExceptionalPolicy(resourceName)
}

func (cp *permissionedPkACProvider) GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error) {
	return pbac.MemberStatus_NORMAL, nil
}

func (cp *permissionedPkACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {
	return true, nil
}

// VerifyPrincipal verifies if the principal for the resource is met
func (cp *permissionedPkACProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {

	if atomic.LoadInt32(&cp.acService.orgNum) <= 0 {
		return false, fmt.Errorf("authentication failed: empty organization list or trusted node list on this chain")
	}

	refinedPrincipal, err := cp.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := cp.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return cp.acService.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (cp *permissionedPkACProvider) GetValidEndorsements(principal protocol.Principal) ([]*common.EndorsementEntry, error) {
	if atomic.LoadInt32(&cp.acService.orgNum) <= 0 {
		return nil, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}
	refinedPolicy, err := cp.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("authentication fail, not a member on this chain: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()

	p, err := cp.acService.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return nil, fmt.Errorf("authentication fail: [%v]", err)
	}
	orgListRaw := p.GetOrgList()
	roleListRaw := p.GetRoleList()
	orgList := map[string]bool{}
	roleList := map[protocol.Role]bool{}
	for _, orgRaw := range orgListRaw {
		orgList[orgRaw] = true
	}
	for _, roleRaw := range roleListRaw {
		roleList[roleRaw] = true
	}
	return cp.acService.getValidEndorsements(orgList, roleList, endorsements), nil
}