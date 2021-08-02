/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/pb-go/syscontract"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/common/crypto/asym"
	"chainmaker.org/chainmaker/common/crypto/pkcs11"
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	"chainmaker.org/chainmaker/common/json"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/protocol"
	"github.com/gogo/protobuf/proto"
)

// Special characters allowed to define customized access rules
const (
	LIMIT_DELIMITER = "/"
	PARAM_CERTS     = "certs"

	authenticationFailedErrorTemplate        = "authentication failed, %v"
	failToGetRoleInfoFromCertWarningTemplate = "authentication warning: fail to get role information from " +
		"certificate [%v], certificate information: \n%s\n"
)

var notEnoughParticipantsSupportError = "authentication fail: not enough participants support this action"

var p11HandleMap = map[string]*pkcs11.P11Handle{}

// List of access principals which should not be customized
var restrainedResourceList = map[string]bool{
	protocol.ResourceNameAllTest:          true,
	protocol.ResourceNameReadData:         true,
	protocol.ResourceNameWriteData:        true,
	protocol.ResourceNameUpdateSelfConfig: true,
	protocol.ResourceNameUpdateConfig:     true,
	protocol.ResourceNameP2p:              true,
	protocol.ResourceNameConsensusNode:    true,
	protocol.ResourceNameSubscribe:        true,
}

// Default access principals for predefined operation categories
var txTypeToResourceNameMap = map[common.TxType]string{
	common.TxType_QUERY_CONTRACT:  protocol.ResourceNameReadData,
	common.TxType_INVOKE_CONTRACT: protocol.ResourceNameWriteData,
	//common.TxType_INVOKE_CONTRACT:  protocol.ResourceNameWriteData,
	common.TxType_SUBSCRIBE: protocol.ResourceNameSubscribe,
	//common.TxType_SUBSCRIBE:    protocol.ResourceNameReadData,
	//common.TxType_MANAGE_USER_CONTRACT:          protocol.ResourceNameWriteData,
	//common.TxType_SUBSCRIBE: protocol.ResourceNameReadData,

	common.TxType_ARCHIVE: protocol.ResourceNameArchive,
	//common.TxType_ARCHIVE: protocol.ResourceNameArchive,
}

var (
	policyRead = newPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleConsensusNode,
		protocol.RoleCommonNode, protocol.RoleClient, protocol.RoleAdmin})
	policyWrite     = newPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleClient, protocol.RoleAdmin})
	policyConsensus = newPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleConsensusNode})
	policyP2P       = newPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleConsensusNode,
		protocol.RoleCommonNode})
	policyAdmin     = newPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleAdmin})
	policySubscribe = newPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleLight, protocol.RoleClient,
		protocol.RoleAdmin})

	policyConfig     = newPolicy(protocol.RuleMajority, nil, []protocol.Role{protocol.RoleAdmin})
	policySelfConfig = newPolicy(protocol.RuleSelf, nil, []protocol.Role{protocol.RoleAdmin})

	//policyForbidden = newPolicy(protocol.RuleForbidden, nil, nil)

	policyAllTest = newPolicy(protocol.RuleAll, nil, []protocol.Role{protocol.RoleAdmin})

	policyLimitTestAny        = newPolicy("2", nil, nil)
	policyLimitTestAdmin      = newPolicy("2", nil, []protocol.Role{protocol.RoleAdmin})
	policyPortionTestAny      = newPolicy("3/4", nil, nil)
	policyPortionTestAnyAdmin = newPolicy("3/4", nil, []protocol.Role{protocol.RoleAdmin})
)

func (ac *accessControl) initTrustRoots(roots []*config.TrustRootConfig, localOrgId string) error {
	ac.orgNum = 0
	ac.orgList = &sync.Map{}
	for _, root := range roots {
		org := &organization{
			id:                       root.OrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}

		certificateChain, err := ac.buildCertificateChain(root, org)
		if err != nil {
			return err
		}
		if certificateChain == nil || !certificateChain[len(certificateChain)-1].IsCA {
			return fmt.Errorf("the certificate configured as root for organization %s is not a CA certificate", root.OrgId)
		}
		org.trustedRootCerts[string(certificateChain[len(certificateChain)-1].Raw)] =
			certificateChain[len(certificateChain)-1]
		ac.opts.Roots.AddCert(certificateChain[len(certificateChain)-1])
		for i := 0; i < len(certificateChain); i++ {
			org.trustedIntermediateCerts[string(certificateChain[i].Raw)] = certificateChain[i]
			ac.opts.Intermediates.AddCert(certificateChain[i])
		}

		/*for _, certificate := range certificateChain {
			if certificate.IsCA {
				org.trustedRootCerts[string(certificate.Raw)] = certificate
				ac.opts.Roots.AddCert(certificate)
			} else {
				org.trustedIntermediateCerts[string(certificate.Raw)] = certificate
				ac.opts.Intermediates.AddCert(certificate)
			}
		}*/

		if len(org.trustedRootCerts) <= 0 {
			return fmt.Errorf("setup organization failed, no trusted root (for %s): please configure "+
				"trusted root certificate or trusted public key whitelist", root.OrgId)
		}

		ac.addOrg(org)
	}
	localOrg := ac.getOrgByOrgId(localOrgId)
	if localOrg == nil {
		localOrg = &organization{
			id:                       localOrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
	}
	ac.localOrg = localOrg
	return nil
}

func (ac *accessControl) initLocalSigningMember(localOrgId, localPrivKeyFile, localPrivKeyPwd,
	localCertFile string) error {
	if localPrivKeyFile != "" && localCertFile != "" {
		var err error
		ac.localSigningMember, err = ac.NewSigningMemberFromCertFile(localOrgId, localPrivKeyFile, localPrivKeyPwd,
			localCertFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ac *accessControl) buildCertificateChain(root *config.TrustRootConfig, org *organization) (
	[]*bcx509.Certificate, error) {
	isUsePk := false

	pk, errPubKey := asym.PublicKeyFromPEM([]byte(root.Root))
	if errPubKey == nil {
		isUsePk = true
		if ac.getOrgByOrgId(root.OrgId) != nil {
			return nil, fmt.Errorf("multiple public key for member %s", root.OrgId)
		}
		org.trustedRootCerts[root.Root] = &bcx509.Certificate{Raw: []byte(root.Root), PublicKey: pk, Signature: nil,
			SubjectKeyId: nil}
		ac.identityType = pbac.MemberType_PUBLIC_KEY
	}

	var certificates, certificateChain []*bcx509.Certificate

	pemBlock, rest := pem.Decode([]byte(root.Root))
	for pemBlock != nil {
		cert, errCert := bcx509.ParseCertificate(pemBlock.Bytes)
		if (errCert != nil || cert == nil) && errPubKey != nil {
			return nil, fmt.Errorf("invalid entry in whitelist or trusted root cert list")
		}
		if isUsePk {
			return nil, fmt.Errorf("mixed authentication type: both public key and certificate exist at the same time")
		}
		if len(cert.Signature) == 0 {
			return nil, fmt.Errorf("invalid certificate [SN: %s]", cert.SerialNumber)
		}
		ac.identityType = pbac.MemberType_CERT
		certificates = append(certificates, cert)

		pemBlock, rest = pem.Decode(rest)
	}

	certificateChain = bcx509.BuildCertificateChain(certificates)
	return certificateChain, nil
}

func (ac *accessControl) initTrustRootsForUpdatingChainConfig(roots []*config.TrustRootConfig,
	localOrgId string) error {
	var orgNum int32
	orgList := &sync.Map{}

	opts := bcx509.VerifyOptions{
		Intermediates: bcx509.NewCertPool(),
		Roots:         bcx509.NewCertPool(),
	}
	for _, root := range roots {
		org := &organization{
			id:                       root.OrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}

		certificateChain, err := ac.buildCertificateChainForUpdatingChainConfig(root, org)
		if err != nil {
			return err
		}
		for _, certificate := range certificateChain {
			if certificate.IsCA {
				org.trustedRootCerts[string(certificate.Raw)] = certificate
				opts.Roots.AddCert(certificate)
			} else {
				org.trustedIntermediateCerts[string(certificate.Raw)] = certificate
				opts.Intermediates.AddCert(certificate)
			}
		}

		if len(org.trustedRootCerts) <= 0 {
			return fmt.Errorf("update configuration failed, no trusted root (for %s): please configure "+
				"trusted root certificate or trusted public key whitelist", root.OrgId)
		}

		orgList.Store(org.id, org)
		orgNum++
	}
	atomic.StoreInt32(&ac.orgNum, orgNum)
	ac.orgList = orgList
	ac.opts = opts

	localOrg := ac.getOrgByOrgId(localOrgId)
	if localOrg == nil {
		localOrg = &organization{
			id:                       localOrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
	}
	ac.localOrg = localOrg

	return nil
}

func (ac *accessControl) buildCertificateChainForUpdatingChainConfig(root *config.TrustRootConfig, org *organization) (
	[]*bcx509.Certificate, error) {
	var certificates, certificateChain []*bcx509.Certificate

	if ac.identityType == pbac.MemberType_PUBLIC_KEY {
		pk, errPubKey := asym.PublicKeyFromPEM([]byte(root.Root))
		if errPubKey != nil {
			return nil, fmt.Errorf("update configuration failed, invalid public key for organization %s", root.OrgId)
		}
		if ac.getOrgByOrgId(root.OrgId) != nil {
			return nil, fmt.Errorf("update configuration failed, multiple public key for member %s", root.OrgId)
		}

		org.trustedRootCerts[root.Root] = &bcx509.Certificate{Raw: []byte(root.Root), PublicKey: pk, Signature: nil,
			SubjectKeyId: nil}
	}
	if ac.identityType == pbac.MemberType_CERT {
		pemBlock, rest := pem.Decode([]byte(root.Root))
		for pemBlock != nil {
			cert, errCert := bcx509.ParseCertificate(pemBlock.Bytes)
			if errCert != nil {
				return nil, fmt.Errorf("update configuration failed, invalid certificate for organization %s", root.OrgId)
			}
			if len(cert.Signature) == 0 {
				return nil, fmt.Errorf("update configuration failed, invalid certificate [SN: %s]", cert.SerialNumber)
			}

			certificates = append(certificates, cert)
			pemBlock, rest = pem.Decode(rest)
		}
	}
	certificateChain = bcx509.BuildCertificateChain(certificates)
	return certificateChain, nil
}

func (ac *accessControl) createDefaultResourcePolicy() *sync.Map {
	resourceNamePolicyMap := &sync.Map{}

	resourceNamePolicyMap.Store(protocol.ResourceNameReadData, policyRead)
	resourceNamePolicyMap.Store(protocol.ResourceNameWriteData, policyWrite)
	resourceNamePolicyMap.Store(protocol.ResourceNameConsensusNode, policyConsensus)
	resourceNamePolicyMap.Store(protocol.ResourceNameP2p, policyP2P)
	resourceNamePolicyMap.Store(protocol.ResourceNameAdmin, policyAdmin)
	resourceNamePolicyMap.Store(protocol.ResourceNameSubscribe, policySubscribe)

	resourceNamePolicyMap.Store(protocol.ResourceNameUpdateConfig, policyConfig)
	resourceNamePolicyMap.Store(protocol.ResourceNameUpdateSelfConfig, policySelfConfig)

	// only used for test
	resourceNamePolicyMap.Store(protocol.ResourceNameAllTest, policyAllTest)
	resourceNamePolicyMap.Store("test_2", policyLimitTestAny)
	resourceNamePolicyMap.Store("test_2_admin", policyLimitTestAdmin)
	resourceNamePolicyMap.Store("test_3/4", policyPortionTestAny)
	resourceNamePolicyMap.Store("test_3/4_admin", policyPortionTestAnyAdmin)

	// transaction resource definitions
	resourceNamePolicyMap.Store(protocol.ResourceNameTxQuery, policyRead)
	resourceNamePolicyMap.Store(protocol.ResourceNameTxTransact, policyWrite)

	//for private compute
	resourceNamePolicyMap.Store(protocol.ResourceNamePrivateCompute, policyWrite)
	resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeContractFunction_SAVE_CA_CERT.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
		syscontract.PrivateComputeContractFunction_SAVE_ENCLAVE_REPORT.String(), policyConfig)

	// system contract interface resource definitions
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(), policyRead)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CORE_UPDATE.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), policyConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), policyConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), policyConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), policyConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), policyConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), policyConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), policySelfConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), policySelfConfig)

	resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_INIT_CONTRACT.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(), policyConfig)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(), policyConfig)

	// certificate management
	resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_FREEZE.String(), policyAdmin)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(), policyAdmin)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_DELETE.String(), policyAdmin)
	resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_REVOKE.String(), policyAdmin)

	// Archive
	resourceNamePolicyMap.Store(protocol.ResourceNameArchive, newPolicy(protocol.RuleAny,
		[]string{localconf.ChainMakerConfig.NodeConfig.OrgId}, []protocol.Role{protocol.RoleAdmin}))

	return resourceNamePolicyMap
}

func (ac *accessControl) initResourcePolicy(resourcePolicies []*config.ResourcePolicy) {
	resourceNamePolicyMap := ac.createDefaultResourcePolicy()
	for _, resourcePolicy := range resourcePolicies {
		if ac.ValidateResourcePolicy(resourcePolicy) {
			policy := newPolicyFromPb(resourcePolicy.Policy)
			resourceNamePolicyMap.Store(resourcePolicy.ResourceName, policy)
		}
	}
	ac.resourceNamePolicyMap = resourceNamePolicyMap
}

func (ac *accessControl) checkResourcePolicyOrgList(policy *pbac.Policy) bool {
	orgCheckList := map[string]bool{}
	for _, org := range policy.OrgList {
		if _, ok := ac.orgList.Load(org); !ok {
			ac.log.Errorf("bad configuration: configured organization list contains unknown organization [%s]", org)
			return false
		} else if _, alreadyIn := orgCheckList[org]; alreadyIn {
			ac.log.Errorf("bad configuration: duplicated entries [%s] in organization list", org)
			return false
		} else {
			orgCheckList[org] = true
		}
	}
	return true
}

func (ac *accessControl) checkResourcePolicyRule(resourcePolicy *config.ResourcePolicy) bool {
	switch resourcePolicy.Policy.Rule {
	case string(protocol.RuleAny), string(protocol.RuleAll), string(protocol.RuleForbidden):
		return true
	case string(protocol.RuleSelf):
		return ac.checkResourcePolicyRuleSelfCase(resourcePolicy)
	case string(protocol.RuleMajority):
		return ac.checkResourcePolicyRuleMajorityCase(resourcePolicy.Policy)
	case string(protocol.RuleDelete):
		ac.log.Debugf("delete policy configuration of %s", resourcePolicy.ResourceName)
		return true
	default:
		return ac.checkResourcePolicyRuleDefaultCase(resourcePolicy.Policy)
	}
}

func (ac *accessControl) checkResourcePolicyRuleSelfCase(resourcePolicy *config.ResourcePolicy) bool {
	switch resourcePolicy.ResourceName {
	case syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(),
		syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
			syscontract.ChainConfigFunction_NODE_ID_UPDATE.String():
		return true
	default:
		ac.log.Errorf("bad configuration: the access rule of [%s] should not be [%s]", resourcePolicy.ResourceName,
			resourcePolicy.Policy.Rule)
		return false
	}
}

func (ac *accessControl) checkResourcePolicyRuleMajorityCase(policy *pbac.Policy) bool {
	if len(policy.OrgList) != int(atomic.LoadInt32(&ac.orgNum)) {
		ac.log.Warnf("[%s] rule considers all the organizations on the chain, any customized configuration for "+
			"organization list will be overridden, should use [Portion] rule for customized organization list",
			protocol.RuleMajority)
	}
	switch len(policy.RoleList) {
	case 0:
		ac.log.Warnf("role allowed in [%s] is [%s]", protocol.RuleMajority, protocol.RoleAdmin)
		return true
	case 1:
		if policy.RoleList[0] != string(protocol.RoleAdmin) {
			ac.log.Warnf("role allowed in [%s] is only [%s], [%s] will be overridden", protocol.RuleMajority,
				protocol.RoleAdmin, policy.RoleList[0])
		}
		return true
	default:
		ac.log.Warnf("role allowed in [%s] is only [%s], the other roles in the list will be ignored",
			protocol.RuleMajority, protocol.RoleAdmin)
		return true
	}
}

func (ac *accessControl) checkResourcePolicyRuleDefaultCase(policy *pbac.Policy) bool {
	nums := strings.Split(policy.Rule, LIMIT_DELIMITER)
	switch len(nums) {
	case 1:
		_, err := strconv.Atoi(nums[0])
		if err != nil {
			ac.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		return true
	case 2:
		numerator, err := strconv.Atoi(nums[0])
		if err != nil {
			ac.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		denominator, err := strconv.Atoi(nums[1])
		if err != nil {
			ac.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		if numerator <= 0 || denominator <= 0 {
			ac.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		return true
	default:
		ac.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
		return false
	}
}

func (ac *accessControl) verifyPrincipalPolicy(principal, refinedPrincipal protocol.Principal, p *policy) (
	bool, error) {
	endorsements := refinedPrincipal.GetEndorsement()
	rule := p.GetRule()

	switch rule {
	case protocol.RuleForbidden:
		return false, fmt.Errorf("authentication fail: [%s] is forbidden to access", refinedPrincipal.GetResourceName())
	case protocol.RuleMajority:
		return ac.verifyPrincipalPolicyRuleMajorityCase(p, endorsements)
	case protocol.RuleSelf:
		return ac.verifyPrincipalPolicyRuleSelfCase(principal.GetTargetOrgId(), endorsements)
	case protocol.RuleAny:
		return ac.verifyPrincipalPolicyRuleAnyCase(p, endorsements, principal.GetResourceName())
	case protocol.RuleAll:
		return ac.verifyPrincipalPolicyRuleAllCase(p, endorsements)
	default:
		return ac.verifyPrincipalPolicyRuleDefaultCase(p, endorsements)
	}
}

func buildOrgListRoleListOfPolicyForVerifyPrincipal(p *policy) (map[string]bool, map[protocol.Role]bool) {
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
	return orgList, roleList
}

func (ac *accessControl) verifyPrincipalPolicyRuleMajorityCase(p *policy, endorsements []*common.EndorsementEntry) (
	bool, error) {
	// notice: accept admin role only, and require majority of all the organizations on the chain
	role := protocol.RoleAdmin
	// orgList, _ := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)

	// warning: majority keywork with non admin constraints
	/*
		if roleList[0] != protocol.protocol.RoleAdmin {
			return false, fmt.Errorf("authentication fail: MAJORITY keyword only allows admin role")
		}
	*/

	numOfValid := ac.countValidEndorsements(map[string]bool{}, map[protocol.Role]bool{role: true}, endorsements)

	if float64(numOfValid) > float64(ac.orgNum)/2.0 {
		return true, nil
	}
	return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
		notEnoughParticipantsSupportError, int(float64(ac.orgNum)/2.0+1), numOfValid)
}

func (ac *accessControl) verifyPrincipalPolicyRuleSelfCase(targetOrg string, endorsements []*common.EndorsementEntry) (
	bool, error) {
	role := protocol.RoleAdmin
	if targetOrg == "" {
		return false, fmt.Errorf("authentication fail: SELF keyword requires the owner of the affected target")
	}
	for _, entry := range endorsements {
		if entry.Signer.OrgId != targetOrg {
			continue
		}
		ouList, err := ac.getSignerRoleList(entry.Signer.MemberInfo)
		if err != nil {
			var info string
			if entry.Signer.MemberType == pbac.MemberType_CERT {
				info = string(entry.Signer.MemberInfo)
			} else {
				info = hex.EncodeToString(entry.Signer.MemberInfo)
			}
			ac.log.Debugf(failToGetRoleInfoFromCertWarningTemplate, err, info)
			continue
		}
		for _, ou := range ouList {
			if ou == role {
				return true, nil
			}
		}
	}
	return false, fmt.Errorf("authentication fail: target [%s] does not belong to the signer", targetOrg)
}

func (ac *accessControl) verifyPrincipalPolicyRuleAnyCase(p *policy, endorsements []*common.EndorsementEntry,
	resourceName string) (bool, error) {
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	for _, endorsement := range endorsements {
		if len(orgList) > 0 {
			if _, ok := orgList[endorsement.Signer.OrgId]; !ok {
				ac.log.Debugf("authentication warning: signer's organization [%s] is not permitted, requires [%v]",
					endorsement.Signer.OrgId, p.GetOrgList())
				continue
			}
		}

		if len(roleList) == 0 {
			return true, nil
		}
		var signerRoleList []protocol.Role
		var err error
		signerRoleList, err = ac.getSignerRoleList(endorsement.Signer.MemberInfo)
		if err != nil {
			ac.log.Debugf(failToGetRoleInfoFromCertWarningTemplate, err,
				ac.getEndorsementSignerMemberInfoString(endorsement.Signer))
			continue
		}
		for _, ou := range signerRoleList {
			if _, ok := roleList[ou]; ok {
				return true, nil
			}
		}
		ac.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]",
			signerRoleList, p.GetRoleList())
	}

	return false, fmt.Errorf("authentication fail: signers do not meet the requirement (%s)", resourceName)
}

func (ac *accessControl) getEndorsementSignerMemberInfoString(signer *pbac.Member) string {
	if signer.MemberType == pbac.MemberType_CERT {
		return string(signer.MemberInfo)
	}
	return hex.EncodeToString(signer.MemberInfo)
}

func (ac *accessControl) verifyPrincipalPolicyRuleAllCase(p *policy, endorsements []*common.EndorsementEntry) (
	bool, error) {
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	numOfValid := ac.countValidEndorsements(orgList, roleList, endorsements)
	if len(orgList) <= 0 && numOfValid == int(atomic.LoadInt32(&ac.orgNum)) {
		return true, nil
	}
	if len(orgList) > 0 && numOfValid == len(orgList) {
		return true, nil
	}
	return false, fmt.Errorf("authentication fail: not all of the listed organtizations consend to this action")
}

func (ac *accessControl) verifyPrincipalPolicyRuleDefaultCase(p *policy, endorsements []*common.EndorsementEntry) (
	bool, error) {
	rule := p.GetRule()
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	nums := strings.Split(string(rule), LIMIT_DELIMITER)
	switch len(nums) {
	case 1:
		threshold, err := strconv.Atoi(nums[0])
		if err != nil {
			return false, fmt.Errorf("authentication fail: unrecognized rule, should be ANY, MAJORITY, ALL, " +
				"SELF, ac threshold (integer), or ac portion (fraction)")
		}

		numOfValid := ac.countValidEndorsements(orgList, roleList, endorsements)
		if numOfValid >= threshold {
			return true, nil
		}
		return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
			notEnoughParticipantsSupportError, threshold, numOfValid)

	case 2:
		numerator, err := strconv.Atoi(nums[0])
		denominator, err2 := strconv.Atoi(nums[1])
		if err != nil || err2 != nil {
			return false, fmt.Errorf("authentication fail: unrecognized rule, should be ANY, MAJORITY, ALL, " +
				"SELF, an integer, or ac fraction")
		}

		if denominator <= 0 {
			denominator = int(atomic.LoadInt32(&ac.orgNum))
		}

		numOfValid := ac.countValidEndorsements(orgList, roleList, endorsements)

		var numRequired float64
		if len(orgList) <= 0 {
			numRequired = float64(atomic.LoadInt32(&ac.orgNum)) * float64(numerator) / float64(denominator)
		} else {
			numRequired = float64(len(orgList)) * float64(numerator) / float64(denominator)
		}
		if float64(numOfValid) >= numRequired {
			return true, nil
		}
		return false, fmt.Errorf("%s: %f valid endorsements required, %d valid endorsements received",
			notEnoughParticipantsSupportError, numRequired, numOfValid)
	default:
		return false, fmt.Errorf("authentication fail: unrecognized principle type, should be ANY, MAJORITY, " +
			"ALL, SELF, an integer (Threshold), or ac fraction (Portion)")
	}
}

func (ac *accessControl) validateCrlVersion(crlPemBytes []byte, crl *pkix.CertificateList) error {
	if ac.dataStore != nil {
		aki, isASN1Encoded, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
		if err != nil {
			return fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPemBytes))
		}
		ac.log.Debugf("AKI is ASN1 encoded: %v", isASN1Encoded)
		crlOldBytes, err := ac.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), aki)
		if err != nil {
			return fmt.Errorf("lookup CRL [%s] failed: %v", hex.EncodeToString(aki), err)
		}
		if crlOldBytes != nil {
			crlOldBlock, _ := pem.Decode(crlOldBytes)
			crlOld, err := x509.ParseCRL(crlOldBlock.Bytes)
			if err != nil {
				return fmt.Errorf("parse old CRL failed: %v", err)
			}
			if crlOld.TBSCertList.Version > crl.TBSCertList.Version {
				return fmt.Errorf("validate CRL failed: version of new CRL should be greater than the old one")
			}
		}
	}
	return nil
}

func (ac *accessControl) systemContractCallbackCertManagementCase(payloadBytes []byte) error {
	var payload common.Payload
	err := proto.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return fmt.Errorf("resolve payload failed: %v", err)
	}
	switch payload.Method {
	case syscontract.CertManageFunction_CERTS_FREEZE.String():
		return ac.systemContractCallbackCertManagementCertFreezeCase(&payload)
	case syscontract.CertManageFunction_CERTS_UNFREEZE.String():
		return ac.systemContractCallbackCertManagementCertUnfreezeCase(&payload)
	case syscontract.CertManageFunction_CERTS_REVOKE.String():
		return ac.systemContractCallbackCertManagementCertRevokeCase(&payload)
	default:
		ac.log.Debugf("unwatched method [%s]", payload.Method)
		return nil
	}
}

func (ac *accessControl) systemContractCallbackCertManagementCertFreezeCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == PARAM_CERTS {
			certList := strings.Replace(string(param.Value), ",", "\n", -1)
			certBlock, rest := pem.Decode([]byte(certList))
			for certBlock != nil {
				ac.frozenList.Store(string(certBlock.Bytes), true)

				certBlock, rest = pem.Decode(rest)
			}
			return nil
		}
	}
	return nil
}

func (ac *accessControl) systemContractCallbackCertManagementCertUnfreezeCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == PARAM_CERTS {
			certList := strings.Replace(string(param.Value), ",", "\n", -1)
			certBlock, rest := pem.Decode([]byte(certList))
			for certBlock != nil {
				_, ok := ac.frozenList.Load(string(certBlock.Bytes))
				if ok {
					ac.frozenList.Delete(string(certBlock.Bytes))
				}

				certBlock, rest = pem.Decode(rest)
			}
			return nil
		}
	}
	return nil
}

func (ac *accessControl) systemContractCallbackCertManagementCertRevokeCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == "cert_crl" {
			crl := strings.Replace(string(param.Value), ",", "\n", -1)
			crls, err := ac.ValidateCRL([]byte(crl))
			if err != nil {
				return fmt.Errorf("update CRL failed, invalid CRLS: %v", err)
			}
			for _, crl := range crls {
				aki, _, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
				if err != nil {
					return fmt.Errorf("update CRL failed: %v", err)
				}
				ac.crl.Store(string(aki), crl)
			}
			return nil
		}
	}
	return nil
}

func (ac *accessControl) getOrgByOrgId(orgId string) *organization {
	org, ok := ac.orgList.Load(orgId)
	if !ok {
		return nil
	}
	return org.(*organization)
}

func (ac *accessControl) getAllOrgs() []*organization {
	orgs := make([]*organization, 0)
	ac.orgList.Range(func(_, value interface{}) bool {
		orgs = append(orgs, value.(*organization))
		return true
	})
	return orgs
}

func (ac *accessControl) addOrg(org *organization) {
	_, loaded := ac.orgList.LoadOrStore(org.id, org)
	if loaded {
		ac.orgList.Store(org.id, org)
	} else {
		ac.orgNum++
	}
}

// Cache for compressed certificate
func (ac *accessControl) lookUpCertCache(certId string) ([]byte, bool) {
	ret, ok := ac.certCache.Get(certId)
	if !ok {
		ac.log.Debugf("looking up the full certificate for the compressed one [%v]", []byte(certId))
		if ac.dataStore == nil {
			ac.log.Debugf("local data storage is not set up")
			return nil, false
		}
		certIdHex := hex.EncodeToString([]byte(certId))
		cert, err := ac.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certIdHex))
		if err != nil {
			ac.log.Debugf("fail to load compressed certificate from local storage [%s]", certIdHex)
			return nil, false
		}
		if cert == nil {
			ac.log.Debugf("cert id [%s] does not exist in local storage", certIdHex)
			return nil, false
		}
		ac.addCertCache(certId, cert)
		ac.log.Debugf("compressed certificate [%s] found and stored in cache", certIdHex)
		return cert, true
	} else if ret != nil {
		ac.log.Debugf("compressed certificate [%v] found in cache", []byte(certId))
		return ret.([]byte), true
	} else {
		ac.log.Debugf("fail to look up compressed certificate [%v] due to an internal error of local cache",
			[]byte(certId))
		return nil, false
	}
}

func (ac *accessControl) addCertCache(signer string, cert []byte) {
	ac.certCache.Add(signer, cert)
}

// Check certificate chain against CRL and frozen list
func (ac *accessControl) checkCRLAgainstTrustedCerts(crl *pkix.CertificateList, orgList []*organization,
	isIntermediate bool) error {
	aki, isASN1Encoded, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
	if err != nil {
		return fmt.Errorf("fail to get AKI of CRL [%s]: %v", crl.TBSCertList.Issuer.String(), err)
	}
	ac.log.Debugf("AKI is ASN1 encoded: %v", isASN1Encoded)
	for _, org := range orgList {
		var targetCerts map[string]*bcx509.Certificate
		if !isIntermediate {
			targetCerts = org.trustedRootCerts
		} else {
			targetCerts = org.trustedIntermediateCerts
		}
		for _, cert := range targetCerts {
			if bytes.Equal(aki, cert.SubjectKeyId) {
				if err := cert.CheckCRLSignature(crl); err != nil {
					return fmt.Errorf("CRL [AKI: %s] is not signed by CA it claims: %v", hex.EncodeToString(aki), err)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("CRL [AKI: %s] is not signed by ac trusted CA", hex.EncodeToString(aki))
}

func (ac *accessControl) loadCRL() error {
	ac.crl = sync.Map{}

	if ac.dataStore == nil {
		return nil
	}

	crlAKIList, err := ac.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(),
		[]byte(protocol.CertRevokeKey))
	if err != nil {
		return fmt.Errorf("fail to update CRL list: %v", err)
	}
	if crlAKIList == nil {
		ac.log.Debugf("empty CRL")
		return nil
	}

	var crlAKIs []string
	err = json.Unmarshal(crlAKIList, &crlAKIs)
	if err != nil {
		return fmt.Errorf("fail to update CRL list: %v", err)
	}

	err = ac.storeCrls(crlAKIs)
	return err
}

func (ac *accessControl) storeCrls(crlAKIs []string) error {
	for _, crlAKI := range crlAKIs {
		crlbytes, err := ac.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(crlAKI))
		if err != nil {
			return fmt.Errorf("fail to load CRL [%s]: %v", hex.EncodeToString([]byte(crlAKI)), err)
		}
		if crlbytes == nil {
			return fmt.Errorf("fail to load CRL [%s]: CRL is nil", hex.EncodeToString([]byte(crlAKI)))
		}
		crls, err := ac.ValidateCRL(crlbytes)
		if err != nil {
			return err
		}
		if crls == nil {
			return fmt.Errorf("empty CRL")
		}

		for _, crl := range crls {
			aki, _, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
			if err != nil {
				return fmt.Errorf("fail to load CRL, fail to get AKI from CRL: %v", err)
			}
			ac.crl.Store(string(aki), crl)
		}
	}
	return nil
}

func (ac *accessControl) checkCRL(certChain []*bcx509.Certificate) error {
	if len(certChain) < 1 {
		return fmt.Errorf("given certificate chain is empty")
	}

	for _, cert := range certChain {
		akiCert := cert.AuthorityKeyId

		crl, ok := ac.crl.Load(string(akiCert))
		if ok {
			// we have ac CRL, check whether the serial number is revoked
			for _, rc := range crl.(*pkix.CertificateList).TBSCertList.RevokedCertificates {
				if rc.SerialNumber.Cmp(cert.SerialNumber) == 0 {
					return errors.New("certificate is revoked")
				}
			}
		}
	}

	return nil
}

func (ac *accessControl) loadCertFrozenList() error {
	ac.frozenList = sync.Map{}

	if ac.dataStore == nil {
		return nil
	}

	certList, err := ac.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(),
		[]byte(protocol.CertFreezeKey))
	if err != nil {
		return fmt.Errorf("update frozen certificate list failed: %v", err)
	}
	if certList == nil {
		return nil
	}

	var certIDs []string
	err = json.Unmarshal(certList, &certIDs)
	if err != nil {
		return fmt.Errorf("update frozen certificate list failed: %v", err)
	}

	for _, certID := range certIDs {
		certBytes, err := ac.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certID))
		if err != nil {
			return fmt.Errorf("load frozen certificate failed: %s", certID)
		}
		if certBytes == nil {
			return fmt.Errorf("load frozen certificate failed: empty certificate [%s]", certID)
		}

		certBlock, _ := pem.Decode(certBytes)
		ac.frozenList.Store(string(certBlock.Bytes), true)
	}

	return nil
}

func (ac *accessControl) checkCertFrozenList(certChain []*bcx509.Certificate) error {
	if len(certChain) < 1 {
		return fmt.Errorf("given certificate chain is empty")
	}

	_, ok := ac.frozenList.Load(string(certChain[0].Raw))
	if ok {
		return fmt.Errorf("certificate is frozen")
	}

	return nil
}

// Local cache for signer verification
func (ac *accessControl) lookUpSignerInCache(signer string) (*cachedSigner, bool) {
	ret, ok := ac.memberCache.Get(signer)
	if ok {
		return ret.(*cachedSigner), true
	}
	return nil, false
}

func (ac *accessControl) addSignerToCache(signer string, info *cachedSigner) {
	ac.memberCache.Add(signer, info)
}

func getP11HandleId() string {
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	return p11Config.Library + p11Config.Label
}

func getP11Handle() (*pkcs11.P11Handle, error) {
	var err error
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	p11Key := getP11HandleId()
	p11Handle, ok := p11HandleMap[p11Key]
	if !ok {
		p11Handle, err = pkcs11.New(p11Config.Library, p11Config.Label, p11Config.Password, p11Config.SessionCacheSize,
			p11Config.Hash)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize organization with HSM: [%v]", err)
		}
		p11HandleMap[p11Key] = p11Handle
	}
	return p11Handle, nil
}

func (ac *accessControl) newMemberFromCert(orgId string, certFile string) (protocol.Member, error) {
	//certPEM, err := ioutil.ReadFile(filepath.Join(localconf.ConfPath, localconf.ChainMakerConfig.NodeConfig.CertFile))
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize identity management service: [%v]", err)
	}
	return ac.NewMemberFromCertPem(orgId, string(certPEM))
}

func (ac *accessControl) countValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) int {
	refinedEndorsements := ac.getValidEndorsements(orgList, roleList, endorsements)
	return ac.countOrgsFromEndorsements(refinedEndorsements)
}

func (ac *accessControl) getValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) []*common.EndorsementEntry {
	var refinedEndorsements []*common.EndorsementEntry
	var err error
	for _, endorsement := range endorsements {
		if len(orgList) > 0 {
			if _, ok := orgList[endorsement.Signer.OrgId]; !ok {
				ac.log.Debugf("authentication warning: signer's organization [%s] is not permitted, requires",
					endorsement.Signer.OrgId, orgList)
				continue
			}
		}

		var signerRoleList []protocol.Role
		if len(roleList) == 0 {
			refinedEndorsements = append(refinedEndorsements, endorsement)
			continue
		}
		signerRoleList, err = ac.getSignerRoleList(endorsement.Signer.MemberInfo)
		if err != nil {
			ac.log.Debugf(failToGetRoleInfoFromCertWarningTemplate, err,
				ac.getEndorsementSignerMemberInfoString(endorsement.Signer))
			continue
		}
		isRoleMatching := ac.isRoleMatching(signerRoleList, roleList, &refinedEndorsements, endorsement)
		if !isRoleMatching {
			ac.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]",
				signerRoleList, roleList)
		}
	}

	return refinedEndorsements
}

func (ac *accessControl) isRoleMatching(signerRoleList []protocol.Role, roleList map[protocol.Role]bool,
	refinedEndorsements *[]*common.EndorsementEntry, endorsement *common.EndorsementEntry) bool {
	isRoleMatching := false
	for _, sr := range signerRoleList {
		if _, ok := roleList[sr]; ok {
			*refinedEndorsements = append(*refinedEndorsements, endorsement)
			isRoleMatching = true
			break
		}
	}
	return isRoleMatching
}

func (ac *accessControl) countOrgsFromEndorsements(endorsements []*common.EndorsementEntry) int {
	mapOrg := map[string]int{}
	for _, endorsement := range endorsements {
		mapOrg[endorsement.Signer.OrgId]++
	}
	return len(mapOrg)
}

// Check whether the provided member is a valid member of this group
func (ac *accessControl) verifyMember(mem protocol.Member) ([]*bcx509.Certificate, error) {
	if mem == nil {
		return nil, fmt.Errorf("authentication failed, invalid member: member should not be nil")
	}
	cert, err := mem.GetCertificate()
	if err != nil {
		return nil, err
	}
	if ac.authMode == MemberMode || ac.identityType == pbac.MemberType_PUBLIC_KEY { // white list mode or public key mode
		return []*bcx509.Certificate{cert}, nil
	}

	certChains, err := cert.Verify(ac.opts)
	if err != nil {
		return nil, fmt.Errorf("authentication failed, not ac valid certificate from trusted CAs: %v", err)
	}
	orgIdFromCert := cert.Subject.Organization[0]
	if mem.GetOrgId() != orgIdFromCert {
		return nil, fmt.Errorf("authentication failed, signer does not belong to the organization it claims "+
			"[claim: %s, certificate: %s]", mem.GetOrgId(), orgIdFromCert)
	}
	org := ac.getOrgByOrgId(orgIdFromCert)
	if org == nil {
		return nil, fmt.Errorf("authentication failed, no orgnization found")
	}
	if len(org.trustedRootCerts) <= 0 {
		return nil, fmt.Errorf("authentication failed, no trusted root: please configure " +
			"trusted root certificate or trusted public key whitelist")
	}

	certChain := ac.findCertChain(org, certChains)
	if certChain != nil {
		return certChain, nil
	}

	return nil, fmt.Errorf("authentication failed, signer does not belong to the organization it claims"+
		" [claim: %s]", mem.GetOrgId())
}

func (ac *accessControl) findCertChain(org *organization, certChains [][]*bcx509.Certificate) []*bcx509.Certificate {
	for _, chain := range certChains {
		rootCert := chain[len(chain)-1]
		_, ok := org.trustedRootCerts[string(rootCert.Raw)]
		if ok {
			var err error
			// check CRL and frozen list
			err = ac.checkCRL(chain)
			if err != nil {
				ac.log.Debugf("authentication failed, CRL: %v", err)
				continue
			}
			err = ac.checkCertFrozenList(chain)
			if err != nil {
				ac.log.Debugf("authentication failed, certificate frozen list: %v", err)
				continue
			}
			return chain
		}
	}
	return nil
}

// Check whether the provided member's role matches the description supplied in PrincipleWhiteList
func (ac *accessControl) satisfyPolicy(mem protocol.Member, policy *policyWhiteList) error {
	return mem.(*member).satisfyPolicy(policy)
}

// all-in-one validation for signing members: certificate chain/whitelist, signature, policies
func (ac *accessControl) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
	endorsements := principal.GetEndorsement()
	msg := principal.GetMessage()
	refinedEndorsement, resultMsg := ac.refineEndorsements(endorsements, msg)
	if len(refinedEndorsement) <= 0 {
		return nil, fmt.Errorf("authentication failed: message not signed by ac member on this chain %s", resultMsg)
	}

	refinedPrincipal, err := ac.CreatePrincipal(principal.GetResourceName(), refinedEndorsement, msg)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: message not signed by ac member on this chain [%v]", err)
	}

	return refinedPrincipal, nil
}

func (ac *accessControl) refineEndorsements(endorsements []*common.EndorsementEntry, msg []byte) (
	[]*common.EndorsementEntry, string) {
	refinedSigners := map[string]bool{}
	var refinedEndorsement []*common.EndorsementEntry

	var memInfo string
	resultMsg := ""
	for _, endorsementEntry := range endorsements {
		endorsement := &common.EndorsementEntry{
			Signer: &pbac.Member{
				OrgId:      endorsementEntry.Signer.OrgId,
				MemberInfo: endorsementEntry.Signer.MemberInfo,
				MemberType: endorsementEntry.Signer.MemberType,
			},
			Signature: endorsementEntry.Signature,
		}
		if endorsement.Signer.MemberType == pbac.MemberType_CERT {
			ac.log.Debugf("target endorser uses full certificate")
			memInfo = string(endorsement.Signer.MemberInfo)
		} else {
			ac.log.Debugf("target endorser uses compressed certificate")
			memInfoBytes, ok := ac.lookUpCertCache(string(endorsement.Signer.MemberInfo))
			if !ok {
				ac.log.Errorf("authentication failed, unknown signer, the provided certificate ID is not registered")
				continue
			}
			memInfo = string(memInfoBytes)
			endorsement.Signer.MemberType = pbac.MemberType_CERT
			endorsement.Signer.MemberInfo = memInfoBytes
		}

		signerInfo, ok := ac.lookUpSignerInCache(memInfo)
		if !ok {
			ac.log.Debugf("certificate not in local cache, should verify it against the trusted root certificates: "+
				"\n%s", memInfo)
			remoteMember, certChain, ok, msgTmp := ac.verifyPrincipalSignerNotInCache(endorsement, msg, memInfo)
			if !ok {
				resultMsg += msgTmp
				continue
			}

			signerInfo = &cachedSigner{
				signer:    remoteMember,
				certChain: certChain,
			}

			ac.addSignerToCache(memInfo, signerInfo)
		} else {
			flat, msgTmp := ac.verifyPrincipalSignerInCache(signerInfo, endorsement, msg, memInfo)
			resultMsg += msgTmp
			if !flat {
				continue
			}
		}

		if _, ok := refinedSigners[memInfo]; !ok {
			refinedSigners[memInfo] = true
			refinedEndorsement = append(refinedEndorsement, endorsement)
		}
	}
	return refinedEndorsement, resultMsg
}

func (ac *accessControl) verifyPrincipalSignerNotInCache(endorsement *common.EndorsementEntry, msg []byte,
	memInfo string) (remoteMember protocol.Member, certChain []*bcx509.Certificate, ok bool, resultMsg string) {
	var err error
	remoteMember, err = ac.NewMemberFromCertPem(endorsement.Signer.OrgId, memInfo)
	if err != nil {
		resultMsg = fmt.Sprintf(authenticationFailedErrorTemplate, err)
		ac.log.Warn(resultMsg)
		ok = false
		return
	}
	certChain, err = ac.verifyMember(remoteMember)
	if err != nil {
		resultMsg = fmt.Sprintf(authenticationFailedErrorTemplate, err)
		ac.log.Warn(resultMsg)
		ok = false
		return
	}
	if err = ac.satisfyPolicy(remoteMember, &policyWhiteList{
		policyType: ac.authMode,
		policyList: ac.localOrg.trustedRootCerts,
	}); err != nil {
		resultMsg = fmt.Sprintf(authenticationFailedErrorTemplate, err)
		ac.log.Warn(resultMsg)
		ok = false
		return
	}

	if err = remoteMember.Verify(ac.GetHashAlg(), msg, endorsement.Signature); err != nil {
		resultMsg = fmt.Sprintf(authenticationFailedErrorTemplate, err)
		ac.log.Debugf("information for invalid signature:\norganization: %s\ncertificate: %s\nmessage: %s\n"+
			"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
		ac.log.Warn(resultMsg)
		ok = false
		return
	}
	ok = true
	return
}

func (ac *accessControl) verifyPrincipalSignerInCache(signerInfo *cachedSigner, endorsement *common.EndorsementEntry,
	msg []byte, memInfo string) (bool, string) {
	// check CRL and certificate frozen list
	err := ac.checkCRL(signerInfo.certChain)
	if err != nil {
		resultMsg := fmt.Sprintf("authentication failed, checking CRL returns error: %v", err)
		ac.log.Warn(resultMsg)
		return false, resultMsg
	}
	err = ac.checkCertFrozenList(signerInfo.certChain)
	if err != nil {
		resultMsg := fmt.Sprintf("authentication failed, checking certificate frozen list returns error: %v", err)
		ac.log.Warn(resultMsg)
		return false, resultMsg
	}

	ac.log.Debugf("certificate is already seen, no need to verify against the trusted root certificates")
	if endorsement.Signer.OrgId != signerInfo.signer.GetOrgId() {
		resultMsg := fmt.Sprintf("authentication failed, signer does not belong to the organization it claims "+
			"[claim: %s, root cert: %s]", endorsement.Signer.OrgId, signerInfo.signer.GetOrgId())
		ac.log.Warn(resultMsg)
		return false, resultMsg
	}
	if err := signerInfo.signer.Verify(ac.GetHashAlg(), msg, endorsement.Signature); err != nil {
		resultMsg := fmt.Sprintf(authenticationFailedErrorTemplate, err)
		ac.log.Debugf("information for invalid signature:\norganization: %s\ncertificate: %s\nmessage: %s\n"+
			"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
		ac.log.Warn(resultMsg)
		return false, resultMsg
	}
	return true, ""
}

func (ac *accessControl) lookUpPolicyByResourceName(resourceName string) (*policy, error) {
	p, ok := ac.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		return nil, fmt.Errorf("look up access policy failed, did not configure access policy "+
			"for resource %s", resourceName)
	}
	return p.(*policy), nil
}

func (ac *accessControl) getSignerRoleList(signerInfo []byte) ([]protocol.Role, error) {
	var ouList []protocol.Role
	memberInfo, ok := ac.lookUpSignerInCache(string(signerInfo))
	if ok {
		ouList = memberInfo.signer.GetRole()
	} else {
		ouListString, err := bcx509.GetOUFromPEM(signerInfo)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: fail to get role list")
		}
		for _, ouString := range ouListString {
			ouString = strings.ToUpper(ouString)
			ouList = append(ouList, protocol.Role(ouString))
		}
	}
	return ouList, nil
}
