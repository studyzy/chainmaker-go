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
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v2/concurrentlru"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/json"
	"chainmaker.org/chainmaker/localconf/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

const ModuleNameAccessControl = "Access Control"

type certACProvider struct {
	acService *accessControlService

	// local cache for certificates (reduce the size of block)
	certCache *concurrentlru.Cache

	// local cache for certificate revocation list and frozen list
	crl        sync.Map
	frozenList sync.Map

	// verification options for organization members
	opts bcx509.VerifyOptions

	localOrg *organization

	//third-party trusted members
	trustMembers *sync.Map
}

type trustMemberCached struct {
	trustMember *config.TrustMemberConfig
	cert        *bcx509.Certificate
}

var _ protocol.AccessControlProvider = (*certACProvider)(nil)

var NilCertACProvider ACProvider = (*certACProvider)(nil)

func (cp *certACProvider) NewACProvider(chainConf protocol.ChainConf, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (protocol.AccessControlProvider, error) {
	certACProvider, err := newCertACProvider(chainConf.ChainConfig(), localOrgId, store, log)
	if err != nil {
		return nil, err
	}
	chainConf.AddWatch(certACProvider)
	chainConf.AddVmWatch(certACProvider)
	return certACProvider, nil
}

func newCertACProvider(chainConfig *config.ChainConfig, localOrgId string,
	store protocol.BlockchainStore, log protocol.Logger) (*certACProvider, error) {
	certACProvider := &certACProvider{
		certCache:  concurrentlru.New(localconf.ChainMakerConfig.NodeConfig.CertCacheSize),
		crl:        sync.Map{},
		frozenList: sync.Map{},
		opts: bcx509.VerifyOptions{
			Intermediates: bcx509.NewCertPool(),
			Roots:         bcx509.NewCertPool(),
		},
		localOrg:     nil,
		trustMembers: &sync.Map{},
	}

	err := certACProvider.initTrustMembers(chainConfig.TrustMembers)
	if err != nil {
		return nil, err
	}

	certACProvider.acService = initAccessControlService(chainConfig.GetCrypto().Hash,
		chainConfig.AuthType, store, log)

	err = certACProvider.initTrustRoots(chainConfig.TrustRoots, localOrgId)
	if err != nil {
		return nil, err
	}

	certACProvider.acService.initResourcePolicy(chainConfig.ResourcePolicies, localOrgId)

	certACProvider.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	certACProvider.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	if err := certACProvider.loadCRL(); err != nil {
		return nil, err
	}

	if err := certACProvider.loadCertFrozenList(); err != nil {
		return nil, err
	}
	return certACProvider, nil
}

func (cp *certACProvider) initTrustRoots(roots []*config.TrustRootConfig, localOrgId string) error {

	for _, orgRoot := range roots {
		org := &organization{
			id:                       orgRoot.OrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
		for _, root := range orgRoot.Root {
			certificateChain, err := cp.buildCertificateChain(root, orgRoot.OrgId, org)
			if err != nil {
				return err
			}
			if certificateChain == nil || !certificateChain[len(certificateChain)-1].IsCA {
				return fmt.Errorf("the certificate configured as root for organization %s is not a CA certificate", orgRoot.OrgId)
			}
			org.trustedRootCerts[string(certificateChain[len(certificateChain)-1].Raw)] =
				certificateChain[len(certificateChain)-1]
			cp.opts.Roots.AddCert(certificateChain[len(certificateChain)-1])
			for i := 0; i < len(certificateChain); i++ {
				org.trustedIntermediateCerts[string(certificateChain[i].Raw)] = certificateChain[i]
				cp.opts.Intermediates.AddCert(certificateChain[i])
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
				return fmt.Errorf(
					"setup organization failed, no trusted root (for %s): "+
						"please configure trusted root certificate or trusted public key whitelist",
					orgRoot.OrgId,
				)
			}
		}
		cp.acService.addOrg(orgRoot.OrgId, org)
	}

	localOrg := cp.acService.getOrgInfoByOrgId(localOrgId)
	if localOrg == nil {
		localOrg = &organization{
			id:                       localOrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
	}
	cp.localOrg, _ = localOrg.(*organization)
	return nil
}

func (cp *certACProvider) buildCertificateChain(root, orgId string, org *organization) ([]*bcx509.Certificate, error) {

	var certificates, certificateChain []*bcx509.Certificate
	pemBlock, rest := pem.Decode([]byte(root))
	for pemBlock != nil {
		cert, errCert := bcx509.ParseCertificate(pemBlock.Bytes)
		if errCert != nil || cert == nil {
			return nil, fmt.Errorf("invalid entry int trusted root cert list")
		}
		if len(cert.Signature) == 0 {
			return nil, fmt.Errorf("invalid certificate [SN: %s]", cert.SerialNumber)
		}
		certificates = append(certificates, cert)
		pemBlock, rest = pem.Decode(rest)
	}
	certificateChain = bcx509.BuildCertificateChain(certificates)
	return certificateChain, nil
}

func (cp *certACProvider) initTrustMembers(trustMembers []*config.TrustMemberConfig) error {
	var syncMap sync.Map
	for _, member := range trustMembers {
		certBlock, _ := pem.Decode([]byte(member.MemberInfo))
		if certBlock == nil {
			return fmt.Errorf("init trust members failed, none certificate given, memberInfo:[%s]",
				member.MemberInfo)
		}
		trustMemberCert, err := bcx509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return fmt.Errorf("init trust members failed, parse certificate failed, memberInfo:[%s]",
				member.MemberInfo)
		}
		cached := &trustMemberCached{
			trustMember: member,
			cert:        trustMemberCert,
		}
		syncMap.Store(member.MemberInfo, cached)
	}
	cp.trustMembers = &syncMap

	return nil
}

func (cp *certACProvider) loadTrustMembers(memberInfo string) (*trustMemberCached, bool) {
	cached, ok := cp.trustMembers.Load(string(memberInfo))
	if ok {
		return cached.(*trustMemberCached), ok
	}
	return nil, ok
}

func (cp *certACProvider) loadCRL() error {
	if cp.acService.dataStore == nil {
		return nil
	}

	crlAKIList, err := cp.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(),
		[]byte(protocol.CertRevokeKey))
	if err != nil {
		return fmt.Errorf("fail to update CRL list: %v", err)
	}
	if crlAKIList == nil {
		cp.acService.log.Debugf("empty CRL")
		return nil
	}

	var crlAKIs []string
	err = json.Unmarshal(crlAKIList, &crlAKIs)
	if err != nil {
		return fmt.Errorf("fail to update CRL list: %v", err)
	}

	err = cp.storeCrls(crlAKIs)
	return err
}

func (cp *certACProvider) storeCrls(crlAKIs []string) error {
	for _, crlAKI := range crlAKIs {
		crlbytes, err := cp.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(crlAKI))
		if err != nil {
			return fmt.Errorf("fail to load CRL [%s]: %v", hex.EncodeToString([]byte(crlAKI)), err)
		}
		if crlbytes == nil {
			return fmt.Errorf("fail to load CRL [%s]: CRL is nil", hex.EncodeToString([]byte(crlAKI)))
		}
		crls, err := cp.ValidateCRL(crlbytes)
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
			cp.crl.Store(string(aki), crl)
		}
	}
	return nil
}

//ValidateCRL validates whether the CRL is issued by a trusted CA
func (cp *certACProvider) ValidateCRL(crlBytes []byte) ([]*pkix.CertificateList, error) {
	crlPEM, rest := pem.Decode(crlBytes)
	if crlPEM == nil {
		return nil, fmt.Errorf("empty CRL")
	}
	var crls []*pkix.CertificateList
	orgInfos := cp.acService.getAllOrgInfos()
	for crlPEM != nil {
		crl, err := x509.ParseCRL(crlPEM.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPEM.Bytes))
		}

		err = cp.validateCrlVersion(crlPEM.Bytes, crl)
		if err != nil {
			return nil, err
		}
		orgs := make([]*organization, 0)
		for _, org := range orgInfos {
			orgs = append(orgs, org.(*organization))
		}
		err1 := cp.checkCRLAgainstTrustedCerts(crl, orgs, false)
		err2 := cp.checkCRLAgainstTrustedCerts(crl, orgs, true)
		if err1 != nil && err2 != nil {
			return nil, fmt.Errorf("invalid CRL: \n\t[verification against trusted root certs: %v], \n\t["+
				"verification against trusted intermediate certs: %v]", err1, err2)
		}

		crls = append(crls, crl)
		crlPEM, rest = pem.Decode(rest)
	}
	return crls, nil
}

func (cp *certACProvider) validateCrlVersion(crlPemBytes []byte, crl *pkix.CertificateList) error {
	if cp.acService.dataStore != nil {
		aki, isASN1Encoded, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
		if err != nil {
			return fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPemBytes))
		}
		cp.acService.log.Debugf("AKI is ASN1 encoded: %v", isASN1Encoded)
		crlOldBytes, err := cp.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), aki)
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

//check CRL against trusted certs
func (cp *certACProvider) checkCRLAgainstTrustedCerts(crl *pkix.CertificateList,
	orgList []*organization, isIntermediate bool) error {
	aki, isASN1Encoded, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
	if err != nil {
		return fmt.Errorf("fail to get AKI of CRL [%s]: %v", crl.TBSCertList.Issuer.String(), err)
	}
	cp.acService.log.Debugf("AKI is ASN1 encoded: %v", isASN1Encoded)
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

func (cp *certACProvider) checkCRL(certChain []*bcx509.Certificate) error {
	if len(certChain) < 1 {
		return fmt.Errorf("given certificate chain is empty")
	}

	for _, cert := range certChain {
		akiCert := cert.AuthorityKeyId

		crl, ok := cp.crl.Load(string(akiCert))
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

func (cp *certACProvider) loadCertFrozenList() error {
	if cp.acService.dataStore == nil {
		return nil
	}

	certList, err := cp.acService.dataStore.
		ReadObject(syscontract.SystemContract_CERT_MANAGE.String(),
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
		certBytes, err := cp.acService.dataStore.
			ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certID))
		if err != nil {
			return fmt.Errorf("load frozen certificate failed: %s", certID)
		}
		if certBytes == nil {
			return fmt.Errorf("load frozen certificate failed: empty certificate [%s]", certID)
		}

		certBlock, _ := pem.Decode(certBytes)
		cp.frozenList.Store(string(certBlock.Bytes), true)
	}
	return nil
}

func (cp *certACProvider) checkCertFrozenList(certChain []*bcx509.Certificate) error {
	if len(certChain) < 1 {
		return fmt.Errorf("given certificate chain is empty")
	}
	_, ok := cp.frozenList.Load(string(certChain[0].Raw))
	if ok {
		return fmt.Errorf("certificate is frozen")
	}
	return nil
}

func (cp *certACProvider) systemContractCallbackCertManagementCertFreezeCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == PARAM_CERTS {
			certList := strings.Replace(string(param.Value), ",", "\n", -1)
			certBlock, rest := pem.Decode([]byte(certList))
			for certBlock != nil {
				cp.frozenList.Store(string(certBlock.Bytes), true)

				certBlock, rest = pem.Decode(rest)
			}
			return nil
		}
	}
	return nil
}

func (cp *certACProvider) systemContractCallbackCertManagementCertUnfreezeCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == PARAM_CERTS {
			certList := strings.Replace(string(param.Value), ",", "\n", -1)
			certBlock, rest := pem.Decode([]byte(certList))
			for certBlock != nil {
				_, ok := cp.frozenList.Load(string(certBlock.Bytes))
				if ok {
					cp.frozenList.Delete(string(certBlock.Bytes))
				}
				certBlock, rest = pem.Decode(rest)
			}
			return nil
		}
	}
	return nil
}

func (cp *certACProvider) systemContractCallbackCertManagementCertRevokeCase(payload *common.Payload) error {
	for _, param := range payload.Parameters {
		if param.Key == "cert_crl" {
			crl := strings.Replace(string(param.Value), ",", "\n", -1)
			crls, err := cp.ValidateCRL([]byte(crl))
			if err != nil {
				return fmt.Errorf("update CRL failed, invalid CRLS: %v", err)
			}
			for _, crl := range crls {
				aki, _, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
				if err != nil {
					return fmt.Errorf("update CRL failed: %v", err)
				}
				cp.crl.Store(string(aki), crl)
			}
			return nil
		}
	}
	return nil
}

// GetHashAlg return hash algorithm the access control provider uses
func (cp *certACProvider) GetHashAlg() string {
	return cp.acService.hashType
}

func (cp *certACProvider) NewMember(pbMember *pbac.Member) (protocol.Member, error) {
	if pbMember.MemberType != pbac.MemberType_CERT &&
		pbMember.MemberType != pbac.MemberType_CERT_HASH {
		return nil, fmt.Errorf("new member failed: the member type does not match")
	}

	if pbMember.MemberType == pbac.MemberType_CERT_HASH {
		memInfoBytes, ok := cp.lookUpCertCache(pbMember.MemberInfo)
		if !ok {
			return nil, fmt.Errorf("new member failed, the provided certificate ID is not registered")
		}
		pbMember.MemberInfo = memInfoBytes
	}

	memberCache, ok := cp.acService.lookUpMemberInCache(string(pbMember.MemberInfo))
	if !ok {
		remoteMember, isTrustMember, err := cp.newNoCacheMember(pbMember)
		if err != nil {
			return nil, fmt.Errorf("new member failed: %s", err.Error())
		}

		var certChain []*bcx509.Certificate
		if !isTrustMember {
			certChain, err = cp.verifyMember(remoteMember)
			if err != nil {
				return nil, fmt.Errorf("new member failed: %s", err.Error())
			}
		}

		cp.acService.memberCache.Add(string(pbMember.MemberInfo), &memberCached{
			member:    remoteMember,
			certChain: certChain,
		})
		return remoteMember, nil
	}
	return memberCache.member, nil
}

func (cp *certACProvider) newNoCacheMember(pbMember *pbac.Member) (member protocol.Member,
	isTrustMember bool, err error) {
	cached, ok := cp.loadTrustMembers(string(pbMember.MemberInfo))
	if ok {
		var isCompressed bool
		if pbMember.MemberType == pbac.MemberType_CERT {
			isCompressed = false
		}
		var certMember *certificateMember
		certMember, err = newCertMemberFromParam(cached.trustMember.OrgId, cached.trustMember.Role,
			cp.acService.hashType, isCompressed, []byte(cached.trustMember.MemberInfo))
		if err != nil {
			return nil, isTrustMember, err
		}
		isTrustMember = true
		return certMember, isTrustMember, nil
	}

	member, err = cp.acService.newCertMember(pbMember)
	if err != nil {
		return nil, isTrustMember, fmt.Errorf("new member failed: %s", err.Error())
	}
	return member, isTrustMember, nil
}

// ValidateResourcePolicy checks whether the given resource principal is valid
func (cp *certACProvider) ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool {
	return cp.acService.validateResourcePolicy(resourcePolicy)
}

// CreatePrincipalForTargetOrg creates a principal for "SELF" type principal,
// which needs to convert SELF to a sepecific organization id in one authentication
func (cp *certACProvider) CreatePrincipalForTargetOrg(resourceName string,
	endorsements []*common.EndorsementEntry, message []byte,
	targetOrgId string) (protocol.Principal, error) {
	return cp.acService.createPrincipalForTargetOrg(resourceName, endorsements, message, targetOrgId)
}

// CreatePrincipal creates a principal for one time authentication
func (cp *certACProvider) CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry,
	message []byte) (
	protocol.Principal, error) {
	return cp.acService.createPrincipal(resourceName, endorsements, message)
}

func (cp *certACProvider) LookUpPolicy(resourceName string) (*pbac.Policy, error) {
	return cp.acService.lookUpPolicy(resourceName)
}

func (cp *certACProvider) LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error) {
	return cp.acService.lookUpExceptionalPolicy(resourceName)
}

func (cp *certACProvider) GetMemberStatus(pbMember *pbac.Member) (pbac.MemberStatus, error) {

	member, err := cp.NewMember(pbMember)
	if err != nil {
		cp.acService.log.Infof("get member status: %s", err.Error())
		return pbac.MemberStatus_INVALID, err
	}

	var certChain []*bcx509.Certificate
	cert := member.(*certificateMember).cert

	certChain = append(certChain, cert)
	err = cp.checkCRL(certChain)
	if err != nil && err.Error() == "certificate is revoked" {
		return pbac.MemberStatus_REVOKED, err
	}
	err = cp.checkCertFrozenList(certChain)
	if err != nil && err.Error() == "certificate is frozen" {
		return pbac.MemberStatus_FROZEN, err
	}
	return pbac.MemberStatus_NORMAL, nil
}

func (cp *certACProvider) VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error) {

	if verifyType != pbac.VerifyType_CRL {
		return false, fmt.Errorf("verify related material failed: only CRL allowed in permissionedWithCert mode")
	}

	crlPEM, rest := pem.Decode(data)
	if crlPEM == nil {
		cp.acService.log.Debug("verify member's related material failed: empty CRL")
		return false, fmt.Errorf("empty CRL")
	}
	orgInfos := cp.acService.getAllOrgInfos()

	var err1, err2 error

	for crlPEM != nil {
		crl, err := x509.ParseCRL(crlPEM.Bytes)
		if err != nil {
			return false, fmt.Errorf("invalid CRL: %v\n[%s]", err, hex.EncodeToString(crlPEM.Bytes))
		}

		err = cp.validateCrlVersion(crlPEM.Bytes, crl)
		if err != nil {
			return false, err
		}
		orgs := make([]*organization, 0)
		for _, org := range orgInfos {
			orgs = append(orgs, org.(*organization))
		}
		err1 = cp.checkCRLAgainstTrustedCerts(crl, orgs, false)
		err2 = cp.checkCRLAgainstTrustedCerts(crl, orgs, true)
		if err1 != nil && err2 != nil {
			return false, fmt.Errorf(
				"invalid CRL: \n\t[verification against trusted root certs: %v], "+
					"\n\t[verification against trusted intermediate certs: %v]",
				err1,
				err2,
			)
		}
		crlPEM, rest = pem.Decode(rest)
	}
	return true, nil
}

// VerifyPrincipal verifies if the principal for the resource is met
func (cp *certACProvider) VerifyPrincipal(principal protocol.Principal) (bool, error) {

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

// all-in-one validation for signing members: certificate chain/whitelist, signature, policies
func (cp *certACProvider) refinePrincipal(principal protocol.Principal) (protocol.Principal, error) {
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

func (cp *certACProvider) refineEndorsements(endorsements []*common.EndorsementEntry,
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
		if endorsement.Signer.MemberType == pbac.MemberType_CERT {
			cp.acService.log.Debugf("target endorser uses full certificate")
			memInfo = string(endorsement.Signer.MemberInfo)
		}
		if endorsement.Signer.MemberType == pbac.MemberType_CERT_HASH {
			cp.acService.log.Debugf("target endorser uses compressed certificate")
			memInfoBytes, ok := cp.lookUpCertCache(endorsement.Signer.MemberInfo)
			if !ok {
				cp.acService.log.Infof("authentication failed, unknown signer, the provided certificate ID is not registered")
				continue
			}
			memInfo = string(memInfoBytes)
			endorsement.Signer.MemberInfo = memInfoBytes
		}

		signerInfo, ok := cp.acService.lookUpMemberInCache(memInfo)
		if !ok {
			cp.acService.log.Debugf("certificate not in local cache, should verify it against the trusted root certificates: "+
				"\n%s", memInfo)
			remoteMember, certChain, ok, err := cp.verifyPrincipalSignerNotInCache(endorsement, msg, memInfo)
			if !ok {
				cp.acService.log.Infof("verify principal signer not in cache failed, [endorsement: %v],[err: %s]",
					endorsement, err.Error())
				continue
			}

			signerInfo = &memberCached{
				member:    remoteMember,
				certChain: certChain,
			}
			cp.acService.addMemberToCache(memInfo, signerInfo)
		} else {
			flat, err := cp.verifyPrincipalSignerInCache(signerInfo, endorsement, msg, memInfo)
			if !flat {
				cp.acService.log.Infof("verify principal signer in cache failed, [endorsement: %v],[err: %s]",
					endorsement, err.Error())
				continue
			}
		}

		if _, ok := refinedSigners[memInfo]; !ok {
			refinedSigners[memInfo] = true
			refinedEndorsement = append(refinedEndorsement, endorsement)
		}
	}
	return refinedEndorsement
}

// Cache for compressed certificate
func (cp *certACProvider) lookUpCertCache(certId []byte) ([]byte, bool) {
	ret, ok := cp.certCache.Get(string(certId))
	if !ok {
		cp.acService.log.Debugf("looking up the full certificate for the compressed one [%v]", certId)
		if cp.acService.dataStore == nil {
			cp.acService.log.Debugf("local data storage is not set up")
			return nil, false
		}
		certIdHex := hex.EncodeToString(certId)
		cert, err := cp.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certIdHex))
		if err != nil {
			cp.acService.log.Debugf("fail to load compressed certificate from local storage [%s]", certIdHex)
			return nil, false
		}
		if cert == nil {
			cp.acService.log.Debugf("cert id [%s] does not exist in local storage", certIdHex)
			return nil, false
		}
		cp.addCertCache(string(certId), cert)
		cp.acService.log.Debugf("compressed certificate [%s] found and stored in cache", certIdHex)
		return cert, true
	} else if ret != nil {
		cp.acService.log.Debugf("compressed certificate [%v] found in cache", []byte(certId))
		return ret.([]byte), true
	} else {
		cp.acService.log.Debugf("fail to look up compressed certificate [%v] due to an internal error of local cache",
			[]byte(certId))
		return nil, false
	}
}

func (cp *certACProvider) addCertCache(certId string, cert []byte) {
	cp.certCache.Add(certId, cert)
}

func (cp *certACProvider) verifyPrincipalSignerNotInCache(endorsement *common.EndorsementEntry, msg []byte,
	memInfo string) (remoteMember protocol.Member, certChain []*bcx509.Certificate, ok bool, err error) {
	var isTrustMember bool
	remoteMember, isTrustMember, err = cp.newNoCacheMember(endorsement.Signer)
	if err != nil {
		err = fmt.Errorf("new member failed: [%s]", err.Error())
		ok = false
		return
	}

	if !isTrustMember {
		certChain, err = cp.verifyMember(remoteMember)
		if err != nil {
			err = fmt.Errorf("verify member failed: [%s]", err.Error())
			ok = false
			return
		}
	}

	if err = remoteMember.Verify(cp.acService.hashType, msg, endorsement.Signature); err != nil {
		err = fmt.Errorf("member verify signature failed: [%s]", err.Error())
		cp.acService.log.Debugf("information for invalid signature:\norganization: %s\ncertificate: %s\nmessage: %s\n"+
			"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
		ok = false
		return
	}
	ok = true
	return
}

func (cp *certACProvider) verifyPrincipalSignerInCache(signerInfo *memberCached, endorsement *common.EndorsementEntry,
	msg []byte, memInfo string) (bool, error) {
	// check CRL and certificate frozen list

	_, isTrustMember := cp.loadTrustMembers(memInfo)

	if !isTrustMember {
		err := cp.checkCRL(signerInfo.certChain)
		if err != nil {
			return false, fmt.Errorf("check CRL, error: [%s]", err.Error())
		}
		err = cp.checkCertFrozenList(signerInfo.certChain)
		if err != nil {
			return false, fmt.Errorf("check cert forzen list, error: [%s]", err.Error())
		}
		cp.acService.log.Debugf("certificate is already seen, no need to verify against the trusted root certificates")

		if endorsement.Signer.OrgId != signerInfo.member.GetOrgId() {
			err := fmt.Errorf("authentication failed, signer does not belong to the organization it claims "+
				"[claim: %s, root cert: %s]", endorsement.Signer.OrgId, signerInfo.member.GetOrgId())
			return false, err
		}
	}
	if err := signerInfo.member.Verify(cp.acService.hashType, msg, endorsement.Signature); err != nil {
		err = fmt.Errorf("signer member verify signature failed: [%s]", err.Error())
		cp.acService.log.Debugf("information for invalid signature:\norganization: %s\ncertificate: %s\nmessage: %s\n"+
			"signature: %s", endorsement.Signer.OrgId, memInfo, hex.Dump(msg), hex.Dump(endorsement.Signature))
		return false, err
	}
	return true, nil
}

// Check whether the provided member is a valid member of this group
func (cp *certACProvider) verifyMember(mem protocol.Member) ([]*bcx509.Certificate, error) {
	if mem == nil {
		return nil, fmt.Errorf("invalid member: member should not be nil")
	}
	certMember, ok := mem.(*certificateMember)
	if !ok {
		return nil, fmt.Errorf("invalid member: member type err")
	}

	certChains, err := certMember.cert.Verify(cp.opts)
	if err != nil {
		return nil, fmt.Errorf("not ac valid certificate from trusted CAs: %v", err)
	}
	orgIdFromCert := certMember.cert.Subject.Organization[0]
	if mem.GetOrgId() != orgIdFromCert {
		return nil, fmt.Errorf(
			"signer does not belong to the organization it claims [claim: %s, certificate: %s]",
			mem.GetOrgId(),
			orgIdFromCert,
		)
	}
	org := cp.acService.getOrgInfoByOrgId(orgIdFromCert)
	if org == nil {
		return nil, fmt.Errorf("no orgnization found")
	}
	if len(org.(*organization).trustedRootCerts) <= 0 {
		return nil, fmt.Errorf("no trusted root: please configure trusted root certificate")
	}

	certChain := cp.findCertChain(org.(*organization), certChains)
	if certChain != nil {
		return certChain, nil
	}
	return nil, fmt.Errorf("authentication failed, signer does not belong to the organization it claims"+
		" [claim: %s]", mem.GetOrgId())
}

func (cp *certACProvider) findCertChain(org *organization, certChains [][]*bcx509.Certificate) []*bcx509.Certificate {
	for _, chain := range certChains {
		rootCert := chain[len(chain)-1]
		_, ok := org.trustedRootCerts[string(rootCert.Raw)]
		if ok {
			var err error
			// check CRL and frozen list
			err = cp.checkCRL(chain)
			if err != nil {
				cp.acService.log.Debugf("authentication failed, CRL: %v", err)
				continue
			}
			err = cp.checkCertFrozenList(chain)
			if err != nil {
				cp.acService.log.Debugf("authentication failed, certificate frozen list: %v", err)
				continue
			}
			return chain
		}
	}
	return nil
}

func (cp *certACProvider) Module() string {
	return ModuleNameAccessControl
}

func (cp *certACProvider) Watch(chainConfig *config.ChainConfig) error {
	cp.acService.hashType = chainConfig.GetCrypto().GetHash()
	cp.acService.authType = chainConfig.AuthType
	err := cp.initTrustRootsForUpdatingChainConfig(chainConfig, cp.localOrg.id)
	if err != nil {
		return err
	}

	cp.acService.initResourcePolicy(chainConfig.ResourcePolicies, cp.localOrg.id)

	cp.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	cp.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	cp.acService.memberCache.Clear()
	cp.certCache.Clear()
	err = cp.initTrustMembers(chainConfig.TrustMembers)
	if err != nil {
		return err
	}
	return nil
}

func (cp *certACProvider) ContractNames() []string {
	return []string{syscontract.SystemContract_CERT_MANAGE.String()}
}

func (cp *certACProvider) Callback(contractName string, payloadBytes []byte) error {
	switch contractName {
	case syscontract.SystemContract_CERT_MANAGE.String():
		return cp.systemContractCallbackCertManagementCase(payloadBytes)
	default:
		cp.acService.log.Debugf("unwatched smart contract [%s]", contractName)
		return nil
	}
}

func (cp *certACProvider) initTrustRootsForUpdatingChainConfig(chainConfig *config.ChainConfig,
	localOrgId string) error {

	var orgNum int32
	orgList := sync.Map{}
	opts := bcx509.VerifyOptions{
		Intermediates: bcx509.NewCertPool(),
		Roots:         bcx509.NewCertPool(),
	}
	for _, orgRoot := range chainConfig.TrustRoots {
		org := &organization{
			id:                       orgRoot.OrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}

		for _, root := range orgRoot.Root {
			certificateChain, err := cp.buildCertificateChainForUpdatingChainConfig(root, orgRoot.OrgId, org)
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
				return fmt.Errorf(
					"update configuration failed, no trusted root (for %s): "+
						"please configure trusted root certificate or trusted public key whitelist",
					orgRoot.OrgId,
				)
			}
		}
		orgList.Store(org.id, org)
		orgNum++
	}
	atomic.StoreInt32(&cp.acService.orgNum, orgNum)
	cp.acService.orgList = &orgList
	cp.opts = opts
	localOrg := cp.acService.getOrgInfoByOrgId(localOrgId)
	if localOrg == nil {
		localOrg = &organization{
			id:                       localOrgId,
			trustedRootCerts:         map[string]*bcx509.Certificate{},
			trustedIntermediateCerts: map[string]*bcx509.Certificate{},
		}
	}
	cp.localOrg, _ = localOrg.(*organization)
	return nil
}

func (cp *certACProvider) buildCertificateChainForUpdatingChainConfig(root, orgId string,
	org *organization) ([]*bcx509.Certificate, error) {
	var certificates, certificateChain []*bcx509.Certificate

	pemBlock, rest := pem.Decode([]byte(root))
	for pemBlock != nil {
		cert, errCert := bcx509.ParseCertificate(pemBlock.Bytes)
		if errCert != nil {
			return nil, fmt.Errorf("update configuration failed, invalid certificate for organization %s", orgId)
		}
		if len(cert.Signature) == 0 {
			return nil, fmt.Errorf("update configuration failed, invalid certificate [SN: %s]", cert.SerialNumber)
		}

		certificates = append(certificates, cert)
		pemBlock, rest = pem.Decode(rest)
	}

	certificateChain = bcx509.BuildCertificateChain(certificates)
	return certificateChain, nil
}

func (cp *certACProvider) systemContractCallbackCertManagementCase(payloadBytes []byte) error {
	var payload common.Payload
	err := proto.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return fmt.Errorf("resolve payload failed: %v", err)
	}
	switch payload.Method {
	case syscontract.CertManageFunction_CERTS_FREEZE.String():
		return cp.systemContractCallbackCertManagementCertFreezeCase(&payload)
	case syscontract.CertManageFunction_CERTS_UNFREEZE.String():
		return cp.systemContractCallbackCertManagementCertUnfreezeCase(&payload)
	case syscontract.CertManageFunction_CERTS_REVOKE.String():
		return cp.systemContractCallbackCertManagementCertRevokeCase(&payload)
	default:
		cp.acService.log.Debugf("unwatched method [%s]", payload.Method)
		return nil
	}
}

//GetValidEndorsements filters all endorsement entries and returns all valid ones
func (cp *certACProvider) GetValidEndorsements(principal protocol.Principal) ([]*common.EndorsementEntry, error) {
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
