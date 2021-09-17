package accesscontrol

import (
	"chainmaker.org/chainmaker-go/localconf"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"encoding/hex"
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
)

var _ protocol.AccessControlProvider = (*pkACProvider)(nil)

var NilPkACProvider ACProvider = (*pkACProvider)(nil)

type pkACProvider struct {

	//chainconfig authType
	authType AuthType

	hashType string

	adminNum int32

	log protocol.Logger

	localOrg string

	adminMember *sync.Map

	consensusMember *sync.Map

	resourceNamePolicyMap *sync.Map

	exceptionalPolicyMap *sync.Map
}

var (
	pubPolicyConsensus = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role {
			protocol.RoleConsensusNode,
		},
	)
	pubPolicyManage = newPolicy(
		protocol.RuleAny,
		nil,
		[]protocol.Role {
			protocol.RoleAdmin,
		},
	)
	pubPolicyMajorityAdmin = newPolicy(
		protocol.RuleMajority,
		nil,
		[]protocol.Role {
			protocol.RoleAdmin,
		},
	)
	pubPolicyTransaction = newPolicy(
		protocol.RuleAny,
		nil,
		nil,
	)
	pubPolicyForbidden = newPolicy(
		protocol.RuleForbidden,
		nil,
		nil,
	)
)

func (cp *pkACProvider) createDefaultResourcePolicy(localOrgId string) {

	policyArchive.orgList = []string{localOrgId}

	cp.resourceNamePolicyMap.Store(protocol.ResourceNameConsensusNode, pubPolicyConsensus)
	// for txtype
	cp.resourceNamePolicyMap.Store(common.TxType_QUERY_CONTRACT.String(), pubPolicyTransaction)
	cp.resourceNamePolicyMap.Store(common.TxType_INVOKE_CONTRACT.String(), pubPolicyTransaction)
	cp.resourceNamePolicyMap.Store(common.TxType_SUBSCRIBE.String(), pubPolicyTransaction)
	cp.resourceNamePolicyMap.Store(common.TxType_ARCHIVE.String(), pubPolicyTransaction)

	// exceptional resourceName
	cp.exceptionalPolicyMap.Store(protocol.ResourceNamePrivateCompute, pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String()+"-"+
		syscontract.PrivateComputeFunction_SAVE_CA_CERT.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String()+"-"+
		syscontract.PrivateComputeFunction_SAVE_ENCLAVE_REPORT.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERT_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_FREEZE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_DELETE.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_REVOKE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_PUBKEY_MANAGEMENT.String()+"-"+
		syscontract.PubkeyManageFunction_PUBKEY_ADD.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_PUBKEY_MANAGEMENT.String()+"-"+
		syscontract.PubkeyManageFunction_PUBKEY_DELETE.String(), pubPolicyForbidden)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CORE_UPDATE.String(), pubPolicyManage)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), pubPolicyManage)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), pubPolicyManage)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(), pubPolicyManage)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(), pubPolicyManage)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(), pubPolicyManage)

	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_GRANT_CONTRACT_ACCESS.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT_ACCESS.String(), pubPolicyForbidden)
	cp.exceptionalPolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_VERIFY_CONTRACT_ACCESS.String(), pubPolicyForbidden)

	// for admin management
	cp.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), pubPolicyMajorityAdmin)
	cp.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), pubPolicyMajorityAdmin)
	cp.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), pubPolicyMajorityAdmin)

}

func (cp *pkACProvider) verifyPrincipalPolicy(principal, refinedPrincipal protocol.Principal, p *policy) (bool, error) {
	endorsements := refinedPrincipal.GetEndorsement()
	rule := p.GetRule()
	switch rule {
	case protocol.RuleForbidden:
		return false, fmt.Errorf("public authentication fail: [%s] is forbidden to access",
			refinedPrincipal.GetResourceName())
	case protocol.RuleAny:
		return cp.verifyRuleAnyCase(p, endorsements)
	case protocol.RuleMajority:
		return cp.verifyRuleMajorityCase(p, endorsements)
	default:
		return false, fmt.Errorf("public authentication fail: [%s] is not supported", rule)
	}
}

func (cp *pkACProvider) verifyRuleAnyCase(p *policy, endorsements []*common.EndorsementEntry) (bool, error) {
	roleList := cp.buildRoleListForVerifyPrincipal(p)
	for _, endorsement := range endorsements {
		if len(roleList) == 0 {
			return true, nil
		}
		member, err := cp.NewMember(endorsement.Signer)
		if err != nil {
			cp.log.Debugf("failed to convert endorsement to member: %s,member info: [%v]",
				err.Error(), string(endorsement.Signer.MemberInfo))
			continue
		}
		if _, ok := roleList[member.GetRole()]; ok {
			return true, nil
		}
	}
	return false, fmt.Errorf("authentication fail for any rule")
}

func (cp *pkACProvider) verifyRuleMajorityCase(p *policy, endorsements []*common.EndorsementEntry) (bool, error) {
	role := protocol.RoleAdmin
	refinedEndorsements := cp.getValidEndorsements(map[string]bool{}, map[protocol.Role]bool{role: true}, endorsements)
	numOfValid := len(refinedEndorsements)
	cp.log.Debugf("verifyRuleMajorityAdminCase: numOfValid=[%d], cp.adminNum=[%d]", numOfValid, cp.adminNum)
	if float64(numOfValid) > float64(cp.adminNum)/2.0 {
		return true, nil
	}
	return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
		notEnoughParticipantsSupportError, int(float64(cp.adminNum)/2.0+1), numOfValid)
}

func (cp *pkACProvider) buildRoleListForVerifyPrincipal(p *policy) (map[protocol.Role]bool) {
	roleListRaw := p.GetRoleList()
	roleList := map[protocol.Role]bool{}
	for _, roleRaw := range roleListRaw {
		roleList[roleRaw] = true
	}
	return roleList
}

func (cp *pkACProvider) lookUpPolicyByResourceName(resourceName string) (*policy, error) {
	p, ok := cp.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		if p, ok = cp.exceptionalPolicyMap.Load(resourceName); !ok {
			return nil, fmt.Errorf("look up access policy failed, did not configure access policy "+
				"for resource %s", resourceName)
		}
	}
	return p.(*policy), nil
}

// all-in-one validation for signing members: signature, policies
func (cp *pkACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
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

func (cp *pkACProvider) refineEndorsements(endorsements []*common.EndorsementEntry,
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
			cp.log.Debugf("target endorser uses public key in pkACProvider")
			memInfo = string(endorsement.Signer.MemberInfo)
		} else {
			cp.log.Errorf("member type error in pkACProvider")
			continue
		}

		remoteMember, err := cp.NewMember(endorsement.Signer)
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

func (cp *pkACProvider) getValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) []*common.EndorsementEntry {
	var refinedEndorsements []*common.EndorsementEntry
	for _, endorsement := range endorsements {
		if len(roleList) == 0 {
			refinedEndorsements = append(refinedEndorsements, endorsement)
			continue
		}

		member, err := cp.NewMember(endorsement.Signer)
		if err != nil {
			cp.log.Debugf("failed to convert endorsement to member: %s,member info: [%v]",
				err.Error(), string(endorsement.Signer.MemberInfo))
			continue
		}
		cp.log.Debugf("getValidEndorsements: signer's role [%v]", member.GetRole())

		if _, ok := roleList[member.GetRole()]; ok {
			refinedEndorsements = append(refinedEndorsements, endorsement)
		} else {
			cp.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]",
				member.GetRole(), roleList)
		}
	}

	return refinedEndorsements
}

// GetHashAlg return hash algorithm the access control provider uses
func (cp *pkACProvider) GetHashAlg() string {
	return cp.hashType
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (cp *pkACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return true
}

func (cp *pkACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	p, ok := cp.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		return nil, fmt.Errorf("policy not found for resource %s", resourceName)
	}
	pbPolicy := p.(*policy).GetPbPolicy()
	return pbPolicy, nil
}

func (cp *pkACProvider) LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	p, ok := cp.exceptionalPolicyMap.Load(resourceName)
	if !ok {
		return nil, fmt.Errorf("exceptional policy not found for resource %s", resourceName)
	}
	pbPolicy := p.(*policy).GetPbPolicy()
	return pbPolicy, nil
}

// CreatePrincipal creates a principal for one time authentication
func (cp *pkACProvider) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (protocol.Principal, error) {

	if len(endorsements) == 0 || message == nil {
		return nil, fmt.Errorf("setup access control principal failed, a principal should contain valid (non-empty)" +
			" signer information, signature, and message")
	}
	if endorsements[0] == nil {
		return nil, fmt.Errorf("setup access control principal failed, signer-signature pair should not be nil")
	}
	return &principal{
		resourceName: resourceName,
		endorsement:  endorsements,
		message:      message,
		targetOrg:    "",
	}, nil
}

func (cp *pkACProvider) CreatePrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte, 	targetOrgId string) (protocol.Principal, error) {

	return nil, fmt.Errorf("setup access control principal failed, CreatePrincipalForTargetOrg is not supported")
}

// VerifyPrincipal verifies if the principal for the resource is met
func (cp *pkACProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {

	refinedPrincipal, err := cp.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := cp.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication failed, [%s]", err.Error())
	}

	return cp.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

func (cp *pkACProvider) NewMember(member *pbac.Member) (protocol.Member, error) {
	if member.MemberType == pbac.MemberType_PUBLIC_KEY {
		return newMemberFromPkPem(member.GetOrgId(), "", string(member.MemberInfo), cp.hashType)
	}
	return nil, fmt.Errorf("new member for pk failed, member type error")
}

func (cp *pkACProvider) GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error) {
	return pbac.MemberStatus_NORMAL, nil
}

func (cp *pkACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {
	return true, nil
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (cp *pkACProvider) GetValidEndorsements(principal protocol.Principal) ([]*common.EndorsementEntry, error) {
	refinedPolicy, err := cp.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("refinePrincipal fail in GetValidEndorsements: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()

	p, err := cp.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return nil, fmt.Errorf("lookUpPolicyByResourceName fail in GetValidEndorsements: [%v]", err)
	}
	roleListRaw := p.GetRoleList()
	orgList := map[string]bool{}
	roleList := map[protocol.Role]bool{}
	for _, roleRaw := range roleListRaw {
		roleList[roleRaw] = true
	}
	return cp.getValidEndorsements(orgList, roleList, endorsements), nil
}