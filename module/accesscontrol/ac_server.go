package accesscontrol

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker/common/concurrentlru"
	"chainmaker.org/chainmaker/common/crypto/pkcs11"
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/pb-go/syscontract"
	"chainmaker.org/chainmaker/protocol"
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
	protocol.ResourceNameAllTest:       true,
	protocol.ResourceNameP2p:           true,
	protocol.ResourceNameConsensusNode: true,

	common.TxType_QUERY_CONTRACT.String():  true,
	common.TxType_INVOKE_CONTRACT.String(): true,
	common.TxType_SUBSCRIBE.String():       true,
	common.TxType_ARCHIVE.String():         true,
}

// Default access principals for predefined operation categories
var txTypeToResourceNameMap = map[common.TxType]string{
	common.TxType_QUERY_CONTRACT:  protocol.ResourceNameReadData,
	common.TxType_INVOKE_CONTRACT: protocol.ResourceNameWriteData,
	common.TxType_SUBSCRIBE:       protocol.ResourceNameSubscribe,
	common.TxType_ARCHIVE:         protocol.ResourceNameArchive,
}

var (
	policyRead      = NewPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleConsensusNode, protocol.RoleCommonNode, protocol.RoleClient, protocol.RoleAdmin})
	policyWrite     = NewPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleClient, protocol.RoleAdmin})
	policyConsensus = NewPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleConsensusNode})
	policyP2P       = NewPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleConsensusNode, protocol.RoleCommonNode})
	policyAdmin     = NewPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleAdmin})
	policySubscribe = NewPolicy(protocol.RuleAny, nil, []protocol.Role{protocol.RoleLight, protocol.RoleClient, protocol.RoleAdmin})
	policyArchive   = NewPolicy(protocol.RuleAny, []string{localconf.ChainMakerConfig.NodeConfig.OrgId}, []protocol.Role{protocol.RoleAdmin})

	policyConfig = NewPolicy(protocol.RuleMajority, nil, []protocol.Role{protocol.RoleAdmin})

	policySelfConfig = NewPolicy(protocol.RuleSelf, nil, []protocol.Role{protocol.RoleAdmin})

	policyForbidden = NewPolicy(protocol.RuleForbidden, nil, nil)

	policyAllTest = NewPolicy(protocol.RuleAll, nil, []protocol.Role{protocol.RoleAdmin})

	policyLimitTestAny        = NewPolicy("2", nil, nil)
	policyLimitTestAdmin      = NewPolicy("2", nil, []protocol.Role{protocol.RoleAdmin})
	policyPortionTestAny      = NewPolicy("3/4", nil, nil)
	policyPortionTestAnyAdmin = NewPolicy("3/4", nil, []protocol.Role{protocol.RoleAdmin})
)

type accessControlService struct {
	orgNum                int32
	orgList               sync.Map                    // map[string]interface{} , orgId -> interface{}
	resourceNamePolicyMap sync.Map                    // map[string]*policy , resourceName -> *policy
	localTrustMembers     []*config.TrustMemberConfig //local trust members
	memberCache           *concurrentlru.Cache
	dataStore             protocol.BlockchainStore
	log                   protocol.Logger
	hashType              string
}

type memberCache struct {
	member    protocol.Member
	certChain []*bcx509.Certificate
}

func initAccessControlService(hashType string, chainConf *config.ChainConfig,
	store protocol.BlockchainStore, log protocol.Logger) *accessControlService {
	acService := &accessControlService{
		orgNum:                0,
		orgList:               sync.Map{},
		resourceNamePolicyMap: sync.Map{},
		localTrustMembers:     chainConf.TrustMembers,
		memberCache:           concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.SignerCacheSize),
		dataStore:             store,
		log:                   log,
		hashType:              hashType,
	}
	acService.initResourcePolicy(chainConf.ResourcePolicies)
	return acService
}

func (acs *accessControlService) createDefaultResourcePolicy() {

	acs.resourceNamePolicyMap.Store(protocol.ResourceNameReadData, policyRead)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameWriteData, policyWrite)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameUpdateSelfConfig, policySelfConfig)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameUpdateConfig, policyConfig)

	acs.resourceNamePolicyMap.Store(protocol.ResourceNameConsensusNode, policyConsensus)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameP2p, policyP2P)

	// only used for test
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameAllTest, policyAllTest)
	acs.resourceNamePolicyMap.Store("test_2", policyLimitTestAny)
	acs.resourceNamePolicyMap.Store("test_2_admin", policyLimitTestAdmin)
	acs.resourceNamePolicyMap.Store("test_3/4", policyPortionTestAny)
	acs.resourceNamePolicyMap.Store("test_3/4_admin", policyPortionTestAnyAdmin)

	// for txtype
	acs.resourceNamePolicyMap.Store(common.TxType_QUERY_CONTRACT.String(), policyRead)
	acs.resourceNamePolicyMap.Store(common.TxType_INVOKE_CONTRACT.String(), policyWrite)
	acs.resourceNamePolicyMap.Store(common.TxType_SUBSCRIBE.String(), policySubscribe)
	acs.resourceNamePolicyMap.Store(common.TxType_ARCHIVE.String(), policyArchive)

	// transaction resource definitions
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameTxQuery, policyRead)
	acs.resourceNamePolicyMap.Store(protocol.ResourceNameTxTransact, policyWrite)

	//for private compute
	acs.resourceNamePolicyMap.Store(protocol.ResourceNamePrivateCompute, policyWrite)
	//resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
	//syscontract.PrivateComputeContractFunction_SAVE_CA_CERT.String(), policyConfig)
	//resourceNamePolicyMap.Store(syscontract.SystemContract_PRIVATE_COMPUTE.String() + "-" +
	//syscontract.PrivateComputeContractFunction_SAVE_ENCLAVE_REPORT.String(), policyConfig)

	// system contract interface resource definitions
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_GET_CHAIN_CONFIG.String(), policyRead)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CORE_UPDATE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_BLOCK_UPDATE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(), policySelfConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_TRUST_MEMBER_UPDATE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_DELETE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ID_UPDATE.String(), policySelfConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_NODE_ORG_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_CONSENSUS_EXT_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_ADD.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_UPDATE.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CHAIN_CONFIG.String()+"-"+
		syscontract.ChainConfigFunction_PERMISSION_DELETE.String(), policyConfig)

	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_INIT_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UPGRADE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_FREEZE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_UNFREEZE_CONTRACT.String(), policyConfig)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CONTRACT_MANAGE.String()+"-"+
		syscontract.ContractManageFunction_REVOKE_CONTRACT.String(), policyConfig)

	// certificate management
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_FREEZE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_UNFREEZE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_DELETE.String(), policyAdmin)
	acs.resourceNamePolicyMap.Store(syscontract.SystemContract_CERT_MANAGE.String()+"-"+
		syscontract.CertManageFunction_CERTS_REVOKE.String(), policyAdmin)
}

func (acs *accessControlService) initResourcePolicy(resourcePolicies []*config.ResourcePolicy) {
	acs.createDefaultResourcePolicy()
	for _, resourcePolicy := range resourcePolicies {
		if acs.validateResourcePolicy(resourcePolicy) {
			policy := NewPolicyFromPb(resourcePolicy.Policy)
			acs.resourceNamePolicyMap.Store(resourcePolicy.ResourceName, policy)
		}
	}
}

func (acs *accessControlService) checkResourcePolicyOrgList(policy *pbac.Policy) bool {
	orgCheckList := map[string]bool{}
	for _, org := range policy.OrgList {
		if _, ok := acs.orgList.Load(org); !ok {
			acs.log.Errorf("bad configuration: configured organization list contains unknown organization [%s]", org)
			return false
		} else if _, alreadyIn := orgCheckList[org]; alreadyIn {
			acs.log.Errorf("bad configuration: duplicated entries [%s] in organization list", org)
			return false
		} else {
			orgCheckList[org] = true
		}
	}
	return true
}

func (acs *accessControlService) checkResourcePolicyRule(resourcePolicy *config.ResourcePolicy) bool {
	switch resourcePolicy.Policy.Rule {
	case string(protocol.RuleAny), string(protocol.RuleAll), string(protocol.RuleForbidden):
		return true
	case string(protocol.RuleSelf):
		return acs.checkResourcePolicyRuleSelfCase(resourcePolicy)
	case string(protocol.RuleMajority):
		return acs.checkResourcePolicyRuleMajorityCase(resourcePolicy.Policy)
	case string(protocol.RuleDelete):
		acs.log.Debugf("delete policy configuration of %s", resourcePolicy.ResourceName)
		return true
	default:
		return acs.checkResourcePolicyRuleDefaultCase(resourcePolicy.Policy)
	}
}

func (acs *accessControlService) checkResourcePolicyRuleSelfCase(resourcePolicy *config.ResourcePolicy) bool {
	switch resourcePolicy.ResourceName {
	case syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
		syscontract.ChainConfigFunction_TRUST_ROOT_UPDATE.String(),
		syscontract.SystemContract_CHAIN_CONFIG.String() + "-" +
			syscontract.ChainConfigFunction_NODE_ID_UPDATE.String():
		return true
	default:
		acs.log.Errorf("bad configuration: the access rule of [%s] should not be [%s]", resourcePolicy.ResourceName,
			resourcePolicy.Policy.Rule)
		return false
	}
}

func (acs *accessControlService) checkResourcePolicyRuleMajorityCase(policy *pbac.Policy) bool {
	if len(policy.OrgList) != int(atomic.LoadInt32(&acs.orgNum)) {
		acs.log.Warnf("[%s] rule considers all the organizations on the chain, any customized configuration for "+
			"organization list will be overridden, should use [Portion] rule for customized organization list",
			protocol.RuleMajority)
	}
	switch len(policy.RoleList) {
	case 0:
		acs.log.Warnf("role allowed in [%s] is [%s]", protocol.RuleMajority, protocol.RoleAdmin)
		return true
	case 1:
		if policy.RoleList[0] != string(protocol.RoleAdmin) {
			acs.log.Warnf("role allowed in [%s] is only [%s], [%s] will be overridden", protocol.RuleMajority,
				protocol.RoleAdmin, policy.RoleList[0])
		}
		return true
	default:
		acs.log.Warnf("role allowed in [%s] is only [%s], the other roles in the list will be ignored",
			protocol.RuleMajority, protocol.RoleAdmin)
		return true
	}
}

func (acs *accessControlService) checkResourcePolicyRuleDefaultCase(policy *pbac.Policy) bool {
	nums := strings.Split(policy.Rule, LIMIT_DELIMITER)
	switch len(nums) {
	case 1:
		_, err := strconv.Atoi(nums[0])
		if err != nil {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		return true
	case 2:
		numerator, err := strconv.Atoi(nums[0])
		if err != nil {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		denominator, err := strconv.Atoi(nums[1])
		if err != nil {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		if numerator <= 0 || denominator <= 0 {
			acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
			return false
		}
		return true
	default:
		acs.log.Errorf(unsupportedRuleErrorTemplate, policy.Rule)
		return false
	}
}

func (acs *accessControlService) addOrg(orgId string, orgInfo interface{}) {
	_, loaded := acs.orgList.LoadOrStore(orgId, orgInfo)
	if loaded {
		acs.orgList.Store(orgId, orgInfo)
	} else {
		acs.orgNum++
	}
}

func (acs *accessControlService) getOrgInfoByOrgId(orgId string) interface{} {
	orgInfo, ok := acs.orgList.Load(orgId)
	if !ok {
		return nil
	}
	return orgInfo
}

func (acs *accessControlService) getAllOrgInfos() []interface{} {
	orgInfos := make([]interface{}, 0)
	acs.orgList.Range(func(_, value interface{}) bool {
		orgInfos = append(orgInfos, value)
		return true
	})
	return orgInfos
}

func (acs *accessControlService) validateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	if _, ok := restrainedResourceList[resourcePolicy.ResourceName]; ok {
		acs.log.Errorf("bad configuration: should not modify the access policy of the resource: %s",
			resourcePolicy.ResourceName)
		return false
	}

	if resourcePolicy.Policy == nil {
		acs.log.Errorf("bad configuration: access principle should not be nil when modifying access control configurations")
		return false
	}

	if !acs.checkResourcePolicyOrgList(resourcePolicy.Policy) {
		return false
	}

	return acs.checkResourcePolicyRule(resourcePolicy)
}

func (acs *accessControlService) createPrincipal(resourceName string, endorsements []*common.EndorsementEntry, message []byte) (
	protocol.Principal, error) {
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

func (acs *accessControlService) createPrincipalForTargetOrg(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte, targetOrgId string) (protocol.Principal, error) {
	p, err := acs.createPrincipal(resourceName, endorsements, message)
	if err != nil {
		return nil, err
	}
	p.(*principal).targetOrg = targetOrgId
	return p, nil
}

func (acs *accessControlService) lookUpPolicyByResourceName(resourceName string) (*policy, error) {
	p, ok := acs.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		return nil, fmt.Errorf("look up access policy failed, did not configure access policy "+
			"for resource %s", resourceName)
	}
	return p.(*policy), nil
}

func (acs *accessControlService) lookUpMemberInCache(memberInfo string) (*memberCache, bool) {
	ret, ok := acs.memberCache.Get(memberInfo)
	if ok {
		return ret.(*memberCache), true
	}
	return nil, false
}

func (acs *accessControlService) addMemberToCache(memberInfo string, member *memberCache) {
	acs.memberCache.Add(memberInfo, member)
}

func (acs *accessControlService) newMember(member *pbac.Member) (protocol.Member, error) {
	memberCached, ok := acs.lookUpMemberInCache(string(member.MemberInfo))
	if ok && memberCached.member.GetOrgId() == member.OrgId {
		acs.log.Debugf("member found in local cache")
		return memberCached.member, nil
	}
	memberFactory := MemberFactory()
	return memberFactory.NewMember(member, acs)
}

func (acs *accessControlService) verifyPrincipalPolicy(principal, refinedPrincipal protocol.Principal, p *policy) (
	bool, error) {
	endorsements := refinedPrincipal.GetEndorsement()
	rule := p.GetRule()

	switch rule {
	case protocol.RuleForbidden:
		return false, fmt.Errorf("authentication fail: [%s] is forbidden to access", refinedPrincipal.GetResourceName())
	case protocol.RuleMajority:
		return acs.verifyPrincipalPolicyRuleMajorityCase(p, endorsements)
	case protocol.RuleSelf:
		return acs.verifyPrincipalPolicyRuleSelfCase(principal.GetTargetOrgId(), endorsements)
	case protocol.RuleAny:
		return acs.verifyPrincipalPolicyRuleAnyCase(p, endorsements, principal.GetResourceName())
	case protocol.RuleAll:
		return acs.verifyPrincipalPolicyRuleAllCase(p, endorsements)
	default:
		return acs.verifyPrincipalPolicyRuleDefaultCase(p, endorsements)
	}
}

func (acs *accessControlService) verifyPrincipalPolicyRuleMajorityCase(p *policy, endorsements []*common.EndorsementEntry) (
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

	numOfValid := acs.countValidEndorsements(map[string]bool{}, map[protocol.Role]bool{role: true}, endorsements)

	if float64(numOfValid) > float64(acs.orgNum)/2.0 {
		return true, nil
	}
	return false, fmt.Errorf("%s: %d valid endorsements required, %d valid endorsements received",
		notEnoughParticipantsSupportError, int(float64(acs.orgNum)/2.0+1), numOfValid)
}

func (acs *accessControlService) verifyPrincipalPolicyRuleSelfCase(targetOrg string, endorsements []*common.EndorsementEntry) (
	bool, error) {
	role := protocol.RoleAdmin
	if targetOrg == "" {
		return false, fmt.Errorf("authentication fail: SELF keyword requires the owner of the affected target")
	}
	for _, entry := range endorsements {
		if entry.Signer.OrgId != targetOrg {
			continue
		}

		member, err := acs.newMember(entry.Signer)
		if err != nil {
			acs.log.Debugf("failed to convert endorsement to member: %s", err.Error())
			continue
		}
		if member.GetRole() == role {
			return true, nil
		}
	}
	return false, fmt.Errorf("authentication fail: target [%s] does not belong to the signer", targetOrg)
}

func (acs *accessControlService) verifyPrincipalPolicyRuleAnyCase(p *policy, endorsements []*common.EndorsementEntry,
	resourceName string) (bool, error) {
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	for _, endorsement := range endorsements {
		if len(orgList) > 0 {
			if _, ok := orgList[endorsement.Signer.OrgId]; !ok {
				acs.log.Debugf("authentication warning: signer's organization [%s] is not permitted, requires [%v]",
					endorsement.Signer.OrgId, p.GetOrgList())
				continue
			}
		}

		if len(roleList) == 0 {
			return true, nil
		}

		member, err := acs.newMember(endorsement.Signer)
		if err != nil {
			acs.log.Debugf("failed to convert endorsement to member: %s", err.Error())
			continue
		}

		if _, ok := roleList[member.GetRole()]; ok {
			return true, nil
		}
		acs.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]",
			member.GetRole(), p.GetRoleList())
	}

	return false, fmt.Errorf("authentication fail: signers do not meet the requirement (%s)", resourceName)
}

func (acs *accessControlService) verifyPrincipalPolicyRuleAllCase(p *policy, endorsements []*common.EndorsementEntry) (
	bool, error) {
	orgList, roleList := buildOrgListRoleListOfPolicyForVerifyPrincipal(p)
	numOfValid := acs.countValidEndorsements(orgList, roleList, endorsements)
	if len(orgList) <= 0 && numOfValid == int(atomic.LoadInt32(&acs.orgNum)) {
		return true, nil
	}
	if len(orgList) > 0 && numOfValid == len(orgList) {
		return true, nil
	}
	return false, fmt.Errorf("authentication fail: not all of the listed organtizations consend to this action")
}

func (acs *accessControlService) verifyPrincipalPolicyRuleDefaultCase(p *policy, endorsements []*common.EndorsementEntry) (
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

		numOfValid := acs.countValidEndorsements(orgList, roleList, endorsements)
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
			denominator = int(atomic.LoadInt32(&acs.orgNum))
		}

		numOfValid := acs.countValidEndorsements(orgList, roleList, endorsements)

		var numRequired float64
		if len(orgList) <= 0 {
			numRequired = float64(atomic.LoadInt32(&acs.orgNum)) * float64(numerator) / float64(denominator)
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

func (acs *accessControlService) countValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool, endorsements []*common.EndorsementEntry) int {
	refinedEndorsements := acs.getValidEndorsements(orgList, roleList, endorsements)
	return countOrgsFromEndorsements(refinedEndorsements)
}

func (acs *accessControlService) getValidEndorsements(orgList map[string]bool, roleList map[protocol.Role]bool,
	endorsements []*common.EndorsementEntry) []*common.EndorsementEntry {
	var refinedEndorsements []*common.EndorsementEntry
	for _, endorsement := range endorsements {
		if len(orgList) > 0 {
			if _, ok := orgList[endorsement.Signer.OrgId]; !ok {
				acs.log.Debugf("authentication warning: signer's organization [%s] is not permitted, requires",
					endorsement.Signer.OrgId, orgList)
				continue
			}
		}

		if len(roleList) == 0 {
			refinedEndorsements = append(refinedEndorsements, endorsement)
			continue
		}

		member, err := acs.newMember(endorsement.Signer)
		if err != nil {
			acs.log.Debugf("failed to convert endorsement to member: %s", err.Error())
			continue
		}

		isRoleMatching := isRoleMatching(member.GetRole(), roleList, &refinedEndorsements, endorsement)
		if !isRoleMatching {
			acs.log.Debugf("authentication warning: signer's role [%v] is not permitted, requires [%v]", member.GetRole(), roleList)
		}
	}

	return refinedEndorsements
}

func isRoleMatching(signerRole protocol.Role, roleList map[protocol.Role]bool, refinedEndorsements *[]*common.EndorsementEntry, endorsement *common.EndorsementEntry) bool {
	isRoleMatching := false
	if _, ok := roleList[signerRole]; ok {
		*refinedEndorsements = append(*refinedEndorsements, endorsement)
		isRoleMatching = true
	}
	return isRoleMatching
}

func countOrgsFromEndorsements(endorsements []*common.EndorsementEntry) int {
	mapOrg := map[string]int{}
	for _, endorsement := range endorsements {
		mapOrg[endorsement.Signer.OrgId]++
	}
	return len(mapOrg)
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

func (acs *accessControlService) lookUpPolicy(resourceName string) (*pbac.Policy, error) {
	p, ok := acs.resourceNamePolicyMap.Load(resourceName)
	if !ok {
		return nil, fmt.Errorf("policy not found for resource %s", resourceName)
	}
	pbPolicy := p.(*policy).GetPbPolicy()
	return pbPolicy, nil
}
