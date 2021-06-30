/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker/common/concurrentlru"
	bccrypto "chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/asym"
	"chainmaker.org/chainmaker/common/crypto/pkcs11"
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	"chainmaker.org/chainmaker-go/localconf"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/config"
	"chainmaker.org/chainmaker/protocol"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"io/ioutil"
	"strings"
	"sync"
	"sync/atomic"
)

const unsupportedRuleErrorTemplate = "bad configuration: unsupported rule [%s]"

// authentication mode: white list, identity, certificate chain, etc.
type AuthMode string

// type used to specify the public information type: certificate or public key
type IdentityType string

const (
	ModuleNameAccessControl = "Access Control"

	MemberMode   AuthMode = "white list" // white list mode
	IdentityMode AuthMode = "identity"   // attribute-authorization mode

	IdentityTypeCert      IdentityType = "certificate"
	IdentityTypePublicKey IdentityType = "public key"
)

var _ protocol.AccessControlProvider = (*accessControl)(nil)

type accessControl struct {
	authMode              AuthMode
	orgList               *sync.Map // map[string]*organization , orgId -> *organization
	orgNum                int32
	resourceNamePolicyMap *sync.Map // map[string]*policy , resourceName -> *policy
	// hash algorithm configured for this chain
	hashType string
	// authentication type: x509 certificate or plain public key
	identityType IdentityType

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

	localOrg           *organization
	localSigningMember protocol.SigningMember

	log protocol.Logger
}

func NewAccessControlWithChainConfig(localPrivKeyFile, localPrivKeyPwd, localCertFile string, chainConfig protocol.ChainConf,
	localOrgId string, store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	conf := chainConfig.ChainConfig()
	acp, err := newAccessControlWithChainConfigPb(localPrivKeyFile, localPrivKeyPwd, localCertFile, conf, localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConfig.AddWatch(acp)
	chainConfig.AddVmWatch(acp)
	return acp, err
}

func newAccessControlWithChainConfigPb(localPrivKeyFile, localPrivKeyPwd, localCertFile string, chainConfig *config.ChainConfig,
	localOrgId string, store protocol.BlockchainStore, log protocol.Logger) (*accessControl, error) {
	ac := &accessControl{
		authMode:              AuthMode(chainConfig.AuthType),
		orgList:               &sync.Map{},
		orgNum:                0,
		resourceNamePolicyMap: &sync.Map{},
		hashType:              chainConfig.GetCrypto().GetHash(),
		identityType:          "",
		dataStore:             store,
		memberCache:           concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.SignerCacheSize),
		certCache:             concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.CertCacheSize),
		crl:                   sync.Map{},
		frozenList:            sync.Map{},
		opts: bcx509.VerifyOptions{
			Intermediates: bcx509.NewCertPool(),
			Roots:         bcx509.NewCertPool(),
		},
		localOrg: nil,
		log:      log,
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

	if err := ac.initLocalSigningMember(localOrgId, localPrivKeyFile, localPrivKeyPwd, localCertFile); err != nil {
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

	if ac.authMode == MemberMode || localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return true, nil
	}

	p, err := ac.lookUpPolicyByResourceName(principal.GetResourceName())
	if err != nil {
		return false, fmt.Errorf("authentication fail: [%v]", err)
	}

	return ac.verifyPrincipalPolicy(principal, refinedPrincipal, p)
}

// LookUpResourceNameByTxType returns resource name corresponding to the tx type
func (ac *accessControl) LookUpResourceNameByTxType(txType common.TxType) (string, error) {
	id, ok := txTypeToResourceNameMap[txType]
	if !ok {
		return protocol.ResourceNameUnknown, fmt.Errorf("invalid transaction type")
	} else {
		return id, nil
	}
}

// GetValidEndorsements filters all endorsement entries and returns all valid ones
func (ac *accessControl) GetValidEndorsements(principal protocol.Principal) ([]*common.EndorsementEntry, error) {
	if atomic.LoadInt32(&ac.orgNum) <= 0 {
		return nil, fmt.Errorf("authentication fail: empty organization list or trusted node list on this chain")
	}
	refinedPolicy, err := ac.refinePrincipal(principal)
	if err != nil {
		return nil, fmt.Errorf("authentication fail, not a member on this chain: [%v]", err)
	}
	endorsements := refinedPolicy.GetEndorsement()
	if ac.authMode == MemberMode || localconf.ChainMakerConfig.DebugConfig.IsSkipAccessControl {
		return endorsements, nil
	}
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

// ValidateCRL validates whether the CRL is issued by a trusted CA
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

// IsCertRevoked verify whether cert chain is revoked by a trusted CA.
func (ac *accessControl) IsCertRevoked(certChain []*bcx509.Certificate) bool {
	err := ac.checkCRL(certChain)
	if err != nil && err.Error() == "certificate is revoked" {
		return true
	}
	return false
}

// DeserializeMember converts bytes to Member
func (ac *accessControl) DeserializeMember(serializedMember []byte) (protocol.Member, error) {
	memberPb := &pbac.SerializedMember{}
	err := proto.Unmarshal(serializedMember, memberPb)
	if err != nil {
		return nil, err
	}

	if !memberPb.IsFullCert {
		memInfoBytes, ok := ac.lookUpCertCache(string(memberPb.MemberInfo))
		if !ok {
			return nil, fmt.Errorf("deserialize Member failed, unrecognized compressed certificate")
		}
		memberPb.MemberInfo = memInfoBytes
		memberPb.IsFullCert = true
	}
	return ac.NewMemberFromCertPem(memberPb.OrgId, string(memberPb.MemberInfo))
}

// GetLocalOrgId returns local organization id
func (ac *accessControl) GetLocalOrgId() string {
	return ac.localOrg.id
}

// GetLocalSigningMember returns local SigningMember
func (ac *accessControl) GetLocalSigningMember() protocol.SigningMember {
	return ac.localSigningMember
}

// NewMemberFromCertPem creates a member from cert pem
func (ac *accessControl) NewMemberFromCertPem(orgId, certPEM string) (protocol.Member, error) {
	var err error

	memberCached, ok := ac.lookUpSignerInCache(certPEM)
	if ok && memberCached.signer.GetOrgId() == orgId {
		ac.log.Debugf("member found in local cache")
		return memberCached.signer, nil
	}

	var newMember member
	newMember.orgId = orgId
	newMember.hashType = ac.hashType

	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, fmt.Errorf("setup member failed, none public key or certificate given")
	}

	pk, err := asym.PublicKeyFromPEM([]byte(certPEM))
	if err == nil {
		certificate := &bcx509.Certificate{
			SubjectKeyId: nil,
			Signature:    nil,
			Raw:          certBlock.Bytes,
			PublicKey:    pk,
		}
		newMember.id = certPEM
		newMember.cert = certificate
		newMember.pk = pk
		newMember.identityType = IdentityTypePublicKey
		return &newMember, nil
	}

	cert, err := bcx509.ParseCertificate(certBlock.Bytes)
	if err == nil {
		orgIdFromCert := cert.Subject.Organization[0]
		if orgIdFromCert != orgId {
			return nil, fmt.Errorf("setup member failed, organization information in certificate and in input parameter do not match [certificate: %s, parameter: %s]", orgIdFromCert, orgId)
		}
		id, err := bcx509.GetExtByOid(bcx509.OidNodeId, cert.Extensions)
		if err != nil {
			id = []byte(cert.Subject.CommonName)
		}
		newMember.id = string(id)
		newMember.cert = cert
		newMember.pk = cert.PublicKey
		/*
			newMember.pk, err = asym.PublicKeyFromDER(cert.RawSubjectPublicKeyInfo)
			if err != nil {
				return nil, fmt.Errorf("fail to parse member public key: %v", err)
			}
		*/

		ou := ""
		if len(cert.Subject.OrganizationalUnit) > 0 {
			ou = cert.Subject.OrganizationalUnit[0]
		}
		ou = strings.ToUpper(ou)

		newMember.role = append(newMember.role, protocol.Role(ou))

		newMember.identityType = IdentityTypeCert
		return &newMember, nil
	}

	return nil, fmt.Errorf("setup member failed, invalid public key or certificate")
}

// NewMemberFromProto creates a member from SerializedMember
func (ac *accessControl) NewMemberFromProto(serializedMember *pbac.SerializedMember) (protocol.Member, error) {
	if serializedMember.IsFullCert {
		return ac.NewMemberFromCertPem(serializedMember.OrgId, string(serializedMember.MemberInfo))
	} else {
		certPEM, ok := ac.lookUpCertCache(string(serializedMember.MemberInfo))
		if !ok {
			return nil, fmt.Errorf("setup member failed, fail to look up certificate ID")
		}
		if certPEM == nil {
			return nil, fmt.Errorf("setup member failed, unknown certificate ID")
		}
		return ac.NewMemberFromCertPem(serializedMember.OrgId, string(certPEM))
	}
}

// NewSigningMemberFromCertFile creates a signing member from private key and cert files
func (ac *accessControl) NewSigningMemberFromCertFile(orgId string, prvKeyFile, password, certFile string) (protocol.SigningMember, error) {
	memberInst, err := ac.newMemberFromCert(orgId, certFile)
	if err != nil {
		return nil, err
	}

	skPEM, err := ioutil.ReadFile(prvKeyFile)

	return ac.NewSigningMember(memberInst, string(skPEM), password)
}

// NewSigningMember creates a signing member from existing member
func (ac *accessControl) NewSigningMember(mem protocol.Member, privateKeyPem string, password string) (protocol.SigningMember, error) {
	var err error
	var sk bccrypto.PrivateKey
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	if p11Config.Enabled {
		p11Handle, err := getP11Handle()
		if err != nil {
			return nil, err
		}
		mem, ok := mem.(*member)
		if !ok {
			return nil, fmt.Errorf("setup member failed, invalid member type")
		}
		sk, err = pkcs11.NewPrivateKey(p11Handle, mem.pk)
		if err != nil {
			return nil, err
		}
	} else {
		sk, err = asym.PrivateKeyFromPEM([]byte(privateKeyPem), []byte(password))
		if err != nil {
			return nil, err
		}
	}

	return &signingMember{
		member: *mem.(*member),
		sk:     sk,
	}, nil
}

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

	return nil
}

func (ac *accessControl) ContractNames() []string {
	return []string{common.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String()}
}

func (ac *accessControl) Callback(contractName string, payloadBytes []byte) error {
	switch contractName {
	case common.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String():
		return ac.systemContractCallbackCertManagementCase(payloadBytes)
	default:
		ac.log.Debugf("unwatched smart contract [%s]", contractName)
		return nil
	}
}
