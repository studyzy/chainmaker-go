/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"strings"

	"chainmaker.org/chainmaker/common/v2/cert"
	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/crypto/pkcs11"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/localconf/v2"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

var _ protocol.Member = (*certificateMember)(nil)

// an instance whose member type is a public key
type certificateMember struct {

	// the CommonName field of the certificate
	id string

	// organization identity who owns this member
	orgId string

	// the X.509 certificate used for authentication
	cert *bcx509.Certificate

	// role of this member
	role protocol.Role

	// hash algorithm for chains (It's not the hash algorithm that the certificate uses)
	hashType string

	// the certificate is compressed or not
	isCompressed bool
}

func (cm *certificateMember) GetMemberId() string {
	return cm.id
}

func (cm *certificateMember) GetOrgId() string {
	return cm.orgId
}

func (cm *certificateMember) GetRole() protocol.Role {
	return cm.role
}

func (cm *certificateMember) GetUid() string {
	return hex.EncodeToString(cm.cert.SubjectKeyId)
}

func (cm *certificateMember) Verify(hashType string, msg []byte, sig []byte) error {
	hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(cm.cert.SignatureAlgorithm)
	if err != nil {
		return fmt.Errorf("cert member verify failed: get hash from signature algorithm failed: [%s]", err.Error())
	}
	ok, err := cm.cert.PublicKey.VerifyWithOpts(msg, sig, &bccrypto.SignOpts{
		Hash: hashAlgo,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
	if err != nil {
		return fmt.Errorf("cert member verify signature failed: [%s]", err.Error())
	}
	if !ok {
		return fmt.Errorf("cert member verify signature failed: invalid signature")
	}
	return nil
}

func (cm *certificateMember) GetMember() (*pbac.Member, error) {
	if cm.isCompressed {
		id, err := utils.GetCertificateIdFromDER(cm.cert.Raw, cm.hashType)
		if err != nil {
			return nil, fmt.Errorf("get pb member failed: [%s]", err.Error())
		}
		return &pbac.Member{
			OrgId:      cm.id,
			MemberInfo: id,
			MemberType: pbac.MemberType_CERT_HASH,
		}, nil
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Bytes: cm.cert.Raw, Type: "CERTIFICATE"})
	return &pbac.Member{
		OrgId:      cm.orgId,
		MemberInfo: certPEM,
		MemberType: pbac.MemberType_CERT,
	}, nil
}

func newCertMemberFromParam(orgId, role, hashType string, isCompressed bool,
	certPEM []byte) (*certificateMember, error) {
	var (
		cert *bcx509.Certificate
		err  error
	)
	certBlock, rest := pem.Decode(certPEM)
	if certBlock == nil {
		cert, err = bcx509.ParseCertificate(rest)
		if err != nil {
			return nil, fmt.Errorf("new cert member failed, invalid certificate")
		}
	} else {
		cert, err = bcx509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("new cert member failed, invalid certificate")
		}
	}

	id, err := bcx509.GetExtByOid(bcx509.OidNodeId, cert.Extensions)
	if err != nil {
		id = []byte(cert.Subject.CommonName)
	}

	role = strings.ToUpper(role)

	return &certificateMember{
		id:           string(id),
		orgId:        orgId,
		role:         protocol.Role(role),
		cert:         cert,
		hashType:     hashType,
		isCompressed: isCompressed,
	}, nil
}

func newMemberFromCertPem(orgId, hashType string, certPEM []byte, isCompressed bool) (*certificateMember, error) {
	var member certificateMember
	member.isCompressed = isCompressed

	var cert *bcx509.Certificate
	var err error
	certBlock, rest := pem.Decode(certPEM)
	if certBlock == nil {
		cert, err = bcx509.ParseCertificate(rest)
		if err != nil {
			return nil, fmt.Errorf("new cert member failed, invalid certificate")
		}
	} else {
		cert, err = bcx509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("new cert member failed, invalid certificate")
		}
	}

	member.hashType = hashType
	member.orgId = orgId

	orgIdFromCert := ""
	if len(cert.Subject.Organization) > 0 {
		orgIdFromCert = cert.Subject.Organization[0]
	}
	if member.orgId == "" {
		member.orgId = orgIdFromCert
	}
	if orgIdFromCert != member.orgId {
		return nil, fmt.Errorf(
			"setup cert member failed, organization information in certificate "+
				"and in input parameter do not match [certificate: %s, parameter: %s]",
			orgIdFromCert,
			orgId,
		)
	}

	id, err := bcx509.GetExtByOid(bcx509.OidNodeId, cert.Extensions)
	if err != nil {
		id = []byte(cert.Subject.CommonName)
	}
	member.id = string(id)
	member.cert = cert
	ou := ""
	if len(cert.Subject.OrganizationalUnit) > 0 {
		ou = cert.Subject.OrganizationalUnit[0]
	}
	ou = strings.ToUpper(ou)
	member.role = protocol.Role(ou)
	return &member, nil
}

func newCertMemberFromPb(member *pbac.Member, acs *accessControlService) (*certificateMember, error) {

	if member.MemberType == pbac.MemberType_CERT {
		return newMemberFromCertPem(member.OrgId, acs.hashType, member.MemberInfo, false)
	}

	if member.MemberType == pbac.MemberType_CERT_HASH {
		return newMemberFromCertPem(member.OrgId, acs.hashType, member.MemberInfo, true)
	}

	return nil, fmt.Errorf("setup member failed, unsupport cert member type")
}

type signingCertMember struct {
	// Extends Identity
	certificateMember

	// Sign the message
	sk bccrypto.PrivateKey
}

// Sign When using certificate, the signature-hash algorithm suite is from the certificate
// and the input hashType is ignored.
func (scm *signingCertMember) Sign(hashType string, msg []byte) ([]byte, error) {
	hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(scm.cert.SignatureAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("sign failed: invalid algorithm: %s", err.Error())
	}

	return scm.sk.SignWithOpts(msg, &bccrypto.SignOpts{
		Hash: hashAlgo,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
}

func NewCertSigningMember(hashType string, member *pbac.Member, privateKeyPem,
	password string) (protocol.SigningMember, error) {

	certMember, err := newMemberFromCertPem(member.OrgId, hashType, member.MemberInfo, false)
	if err != nil {
		return nil, err
	}

	var sk bccrypto.PrivateKey
	nodeConfig := localconf.ChainMakerConfig.NodeConfig
	if nodeConfig.P11Config.Enabled {
		var p11Handle *pkcs11.P11Handle
		p11Handle, err = getP11Handle()
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%v]", err)
		}

		sk, err = cert.ParseP11PrivKey(p11Handle, []byte(privateKeyPem))
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%v]", err)
		}
	} else {
		sk, err = asym.PrivateKeyFromPEM([]byte(privateKeyPem), []byte(password))
		if err != nil {
			return nil, err
		}
	}

	return &signingCertMember{
		*certMember,
		sk,
	}, nil
}
