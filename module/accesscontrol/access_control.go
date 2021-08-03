/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/pb-go/syscontract"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/common/concurrentlru"
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/protocol"
)

const unsupportedRuleErrorTemplate = "bad configuration: unsupported rule [%s]"

const (
	ModuleNameAccessControl = "Access Control"
)

var _ protocol.AccessControlProvider = (*accessControl)(nil)

type accessControl struct {
	orgList               *sync.Map // map[string]*organization , orgId -> *organization
	orgNum                int32
	resourceNamePolicyMap *sync.Map // map[string]*policy , resourceName -> *policy
	// hash algorithm configured for this chain
	hashType string
	// authentication type: x509 certificate or plain public key
	identityType pbac.MemberType

	// data store for chain data
	dataStore protocol.BlockchainStore

	// local cache for the other members on the chain
	memberCache *concurrentlru.Cache

	// local cache for certificates (reduce the size of block)
	certCache *concurrentlru.Cache

	// local cache for certificate revocation list and frozen list
	crl        sync.Map
	frozenList sync.Map

	// verification options for organization members
	opts bcx509.VerifyOptions

	localOrg *organization

	//local trust members
	localTrustMembers []*config.TrustMemberConfig
	log               protocol.Logger
}

func NewAccessControlWithChainConfig(chainConfig protocol.ChainConf,
	localOrgId string, store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	conf := chainConfig.ChainConfig()
	acp, err := newAccessControlWithChainConfigPb(conf, localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConfig.AddWatch(acp)
	chainConfig.AddVmWatch(acp)
	return acp, err
}

func newAccessControlWithChainConfigPb(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*accessControl, error) {
	ac := &accessControl{
		orgList:               &sync.Map{},
		orgNum:                0,
		resourceNamePolicyMap: &sync.Map{},
		hashType:              chainConfig.GetCrypto().GetHash(),
		dataStore:             store,
		memberCache:           concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.SignerCacheSize),
		certCache:             concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.CertCacheSize),
		crl:                   sync.Map{},
		frozenList:            sync.Map{},
		opts: bcx509.VerifyOptions{
			Intermediates: bcx509.NewCertPool(),
			Roots:         bcx509.NewCertPool(),
		},
		localOrg:          nil,
		localTrustMembers: chainConfig.TrustMembers,
		log:               log,
	}
	err := ac.initTrustRoots(chainConfig.TrustRoots, localOrgId)
	if err != nil {
		return nil, err
	}
	ac.initResourcePolicy(chainConfig.ResourcePolicies)

	ac.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	ac.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	if err := ac.loadCRL(); err != nil {
		return nil, err
	}
	if err := ac.loadCertFrozenList(); err != nil {
		return nil, err
	}
	return ac, nil
}

// GetHashAlg return hash algorithm the access control provider uses
func (ac *accessControl) GetHashAlg() string {
	return ac.hashType
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (ac *accessControl) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	if _, ok := restrainedResourceList[resourcePolicy.ResourceName]; ok {
		ac.log.Errorf("bad configuration: should not modify the access policy of the resource: %s", resourcePolicy.ResourceName)
		return false
	}

	if resourcePolicy.Policy == nil {
		ac.log.Errorf("bad configuration: access principle should not be nil when modifying access control configurations")
		return false
	}

	if !ac.checkResourcePolicyOrgList(resourcePolicy.Policy) {
		return false
	}

	return ac.checkResourcePolicyRule(resourcePolicy)
}

// CreatePrincipalForTargetOrg creates a principal for "SELF" type principal,
// which needs to convert SELF to a sepecific organization id in one authentication
func (ac *accessControl) CreatePrincipalForTargetOrg(resourceName string, endorsements []*common.EndorsementEntry, message []byte, targetOrgId string) (protocol.Principal, error) {
	p, err := ac.CreatePrincipal(resourceName, endorsements, message)
	if err != nil {
		return nil, err
	}
	p.(*principal).targetOrg = targetOrgId
	return p, nil
}

// CreatePrincipal creates a principal for one time authentication
func (ac *accessControl) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry, message []byte) (protocol.Principal, error) {
	if len(endorsements) == 0 || message == nil {
		return nil, fmt.Errorf("setup access control principal failed, a principal should contain valid (non-empty) signer information, signature, and message")
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

// VerifyPrincipal verifies if the principal for the resource is met
func (ac *accessControl) VerifyPrincipal(principal protocol.Principal) (bool, error) {
	if atomic.LoadInt32(&ac.orgNum) <= 0 {
		return false, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}

	refinedPrincipal, err := ac.refinePrincipal(principal)
	if err != nil {
		return false, fmt.Errorf("authentication fail, not ac member on this chain: [%v]", err)
	}

	// if ac.authMode == MemberMode || localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
	// 	return true, nil
	// }
	if localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := ac.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication fail: [%v]", err)
	}

	return ac.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

/*
// LookUpResourceNameByTxType returns resource name corresponding to the tx type
func (ac *accessControl) LookUpResourceNameByTxType(txType common.TxType) (string, error) {
	id, ok := txTypeToResourceNameMap[txType]
	if !ok {
		return protocol.ResourceNameUnknown, fmt.Errorf("invalid transaction type")
	} else {
		return id, nil
	}
}

// ResourcePolicyExists checks whether there is corresponding policy configured for the given resource name
func (ac *accessControl) ResourcePolicyExists(resourceName string) bool {
	_, ok := ac.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		ac.log.Debugf("policy not found for resource %s", resourceName)
		return false
	}
	return true
}
*/

// LookUpPolicy returns corresponding policy configured for the given resource name
func (ac *accessControl) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	p, ok := ac.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		return nil, fmt.Errorf("policy not found for resource %s", resourceName)
	}
	pbPolicy := p.(*policy).GetPbPolicy()
	return pbPolicy, nil
}

func (ac *accessControl) NewMember(member *pbac.Member) (protocol.Member, error) {
	return nil, nil
}

func (ac *accessControl) GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error) {
	switch member.MemberType {
	case pbac.MemberType_CERT | pbac.MemberType_CERT_HASH:
		certBlock, _ := pem.Decode(member.MemberInfo)
		if certBlock == nil {
			return pbac.MemberStatus_INVALID, fmt.Errorf("member info decode failed")
		}
		cert, err := bcx509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return pbac.MemberStatus_INVALID, fmt.Errorf("parsing member info failed: %s", err.Error())
		}
		var certChain []*bcx509.Certificate
		certChain = append(certChain, cert)
		err = ac.checkCRL(certChain)
		if err != nil && err.Error() == "certificate is revoked" {
			return pbac.MemberStatus_REVOKED, nil
		}
		return pbac.MemberStatus_NORMAL, nil
	case pbac.MemberType_PUBLIC_KEY:
		return pbac.MemberStatus_NORMAL, nil
	}
	return pbac.MemberStatus_INVALID, fmt.Errorf("get member status failed: unsupport member type")
}

func (ac *accessControl) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {
	switch verifyType {
	case pbac.VerifyType_CRL:
		crlPEM, _ := pem.Decode(data)
		if crlPEM == nil {
			return false, fmt.Errorf("empty CRL")
		}
		var orgs = ac.getAllOrgs()
		for crlPEM != nil {
			crl, err := x509.ParseCRL(crlPEM.Bytes)
			if err != nil {
				return false, fmt.Errorf("invalid CRL: %v\n[%s]\n", err, hex.EncodeToString(crlPEM.Bytes))
			}

			err = ac.validateCrlVersion(crlPEM.Bytes, crl)
			if err != nil {
				return false, err
			}

			err1 := ac.checkCRLAgainstTrustedCerts(crl, orgs, false)
			err2 := ac.checkCRLAgainstTrustedCerts(crl, orgs, true)
			if err1 != nil && err2 != nil {
				return false, fmt.Errorf("invalid CRL: \n\t[verification against trusted root certs: %v], \n\t[verification against trusted intermediate certs: %v]", err1, err2)
			}
		}
		return true, nil
	}
	return false, fmt.Errorf("verify member's related material failed: unsupport verify type")
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (ac *accessControl) GetValidEndorsements(principal protocol.Principal) ([]*common.EndorsementEntry, error) {
	if atomic.LoadInt32(&ac.orgNum) <= 0 {
		return nil, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}
	refinedPolicy, err := ac.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("authentication fail, not a member on this chain: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()
	// if ac.authMode == MemberMode || localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
	// 	return endorsements, nil
	// }
	p, err := ac.lookUpPolicyByResourceName(principal.GetResourceName())
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
	return ac.getValidEndorsements(orgList, roleList, endorsements), nil
}

//ValidateCRL validates whether the CRL is issued by a trusted CA
func (ac *accessControl) ValidateCRL(crlBytes []byte) ([]*pkix.CertificateList, error) {
	crlPEM, rest := pem.Decode(crlBytes)
	if crlPEM == nil {
		return nil, fmt.Errorf("empty CRL")
	}
	var crls []*pkix.CertificateList
	var orgs = ac.getAllOrgs()
	for crlPEM != nil {
		crl, err := x509.ParseCRL(crlPEM.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid CRL: %v\n[%s]\n", err, hex.EncodeToString(crlPEM.Bytes))
		}

		err = ac.validateCrlVersion(crlPEM.Bytes, crl)
		if err != nil {
			return nil, err
		}

		err1 := ac.checkCRLAgainstTrustedCerts(crl, orgs, false)
		err2 := ac.checkCRLAgainstTrustedCerts(crl, orgs, true)
		if err1 != nil && err2 != nil {
			return nil, fmt.Errorf("invalid CRL: \n\t[verification against trusted root certs: %v], \n\t[verification against trusted intermediate certs: %v]", err1, err2)
		}

		crls = append(crls, crl)

		crlPEM, rest = pem.Decode(rest)
	}
	return crls, nil
}

//IsCertRevoked verify whether cert chain is revoked by a trusted CA.
// func (ac *accessControl) IsCertRevoked(certChain []*bcx509.Certificate) bool {
// 	err := ac.checkCRL(certChain)
// 	if err != nil && err.Error() == "certificate is revoked" {
// 		return true
// 	}
// 	return false
// }

func (ac *accessControl) Module() string {
	return ModuleNameAccessControl
}

func (ac *accessControl) Watch(chainConfig *config.ChainConfig) error {
	ac.hashType = chainConfig.GetCrypto().GetHash()
	err := ac.initTrustRootsForUpdatingChainConfig(chainConfig.TrustRoots, ac.localOrg.id)
	if err != nil {
		return err
	}

	ac.initResourcePolicy(chainConfig.ResourcePolicies)

	ac.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	ac.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	ac.memberCache.Clear()
	ac.certCache.Clear()

	ac.localTrustMembers = chainConfig.TrustMembers
	return nil
}

func (ac *accessControl) ContractNames() []string {
	return []string{syscontract.SystemContract_CERT_MANAGE.String()}
}

func (ac *accessControl) Callback(contractName string, payloadBytes []byte) error {
	switch contractName {
	case syscontract.SystemContract_CERT_MANAGE.String():
		return ac.systemContractCallbackCertManagementCase(payloadBytes)
	default:
		ac.log.Debugf("unwatched smart contract [%s]", contractName)
		return nil
	}
}
