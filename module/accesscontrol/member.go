/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/pem"
	"fmt"

	"chainmaker.org/chainmaker-go/utils"
	bccrypto "chainmaker.org/chainmaker/common/crypto"
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

var _ protocol.Member = (*member)(nil)

type member struct {
	// the identity of this member (SKI of its certificate)
	id string

	// organization identity who owns this member
	orgId string

	// cert contains the x.509 certificate that signs the public key of this instance
	cert *bcx509.Certificate

	// this is the public key or certificate of this instance
	pk bccrypto.PublicKey

	// role of this member
	role []protocol.Role

	// authentication type: x509 certificate or plain public key
	identityType pbac.MemberType

	// hash type from chain configuration
	hashType string
}

func (m *member) GetMemberId() string {
	return m.id
}

func (m *member) GetOrgId() string {
	return m.orgId
}

func (m *member) GetRole() []protocol.Role {
	return m.role
}

func (m *member) GetSKI() []byte {
	if m.identityType == pbac.MemberType_CERT || m.identityType == pbac.MemberType_CERT_HASH {
		return m.cert.SubjectKeyId
	} else {
		return m.cert.Raw
	}
}

func (m *member) GetCertificate() (*bcx509.Certificate, error) {
	return m.cert, nil
}

func (m *member) Verify(hashType string, msg []byte, sig []byte) error {
	var opts bccrypto.SignOpts
	if m.identityType == pbac.MemberType_PUBLIC_KEY {
		opts.Hash = bccrypto.HashAlgoMap[hashType]
		opts.UID = bccrypto.CRYPTO_DEFAULT_UID
	} else {
		hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(m.cert.SignatureAlgorithm)
		if err != nil {
			return fmt.Errorf("invalid algorithm: %v", err)
		}
		opts.Hash = hashAlgo
		opts.UID = bccrypto.CRYPTO_DEFAULT_UID
	}
	ok, err := m.pk.VerifyWithOpts(msg, sig, &opts)
	if err != nil {
		return fmt.Errorf("fail to verify signature: [%v]", err)
	}
	if !ok {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func (m *member) GetMember() (*pbac.Member, error) {
	var pemStruct *pem.Block
	switch m.identityType {
	case pbac.MemberType_CERT:
		pemStruct = &pem.Block{Bytes: m.cert.Raw, Type: "CERTIFICATE"}
		certPEM := pem.EncodeToMemory(pemStruct)
		return &pbac.Member{
			OrgId:      m.orgId,
			MemberInfo: certPEM,
			MemberType: pbac.MemberType_CERT,
		}, nil
	case pbac.MemberType_CERT_HASH:
		id, err := utils.GetCertificateIdFromDER(m.cert.Raw, m.hashType)
		if err != nil {
			return nil, fmt.Errorf("fail to compute certificate identity")
		}
		return &pbac.Member{
			OrgId:      m.orgId,
			MemberInfo: id,
			MemberType: pbac.MemberType_CERT_HASH,
		}, nil
	case pbac.MemberType_PUBLIC_KEY:
		pemStruct = &pem.Block{Bytes: m.cert.Raw, Type: "PUBLIC KEY"}
		certPEM := pem.EncodeToMemory(pemStruct)
		return &pbac.Member{
			OrgId:      m.orgId,
			MemberInfo: certPEM,
			MemberType: pbac.MemberType_PUBLIC_KEY,
		}, nil
	}
	return nil, fmt.Errorf("member's identity type is unsupport")
}

type signingMember struct {
	// Extends Identity
	member

	// Sign the message
	sk bccrypto.PrivateKey
}

// When using certificate, the signature-hash algorithm suite is from the certificate, and the input hashType is ignored.
// When using public key instead of certificate, hashType is used to specify the hash algorithm while the signature algorithm is decided by the public key itself.
func (sm *signingMember) Sign(hashType string, msg []byte) ([]byte, error) {
	var opts bccrypto.SignOpts
	if sm.identityType == pbac.MemberType_PUBLIC_KEY {
		opts.Hash = bccrypto.HashAlgoMap[hashType]
		opts.UID = bccrypto.CRYPTO_DEFAULT_UID
	} else {
		hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(sm.cert.SignatureAlgorithm)
		if err != nil {
			return nil, fmt.Errorf("invalid algorithm: %v", err)
		}
		opts.Hash = hashAlgo
		opts.UID = bccrypto.CRYPTO_DEFAULT_UID
	}
	return sm.sk.SignWithOpts(msg, &opts)
}

func (m *member) satisfyPolicy(policy *policyWhiteList) error {
	switch policy.policyType {
	case MemberMode: // whitelist mode
		_, ok := policy.policyList[string(m.cert.Raw)]
		if ok {
			return nil
		} else {
			return fmt.Errorf("not a member for the claimed organization [%s]", m.orgId)
		}
	case IdentityMode: // attribute mode
		// TODO add policy
		return nil
	default:
		return fmt.Errorf("unknown authentication policy type")
	}
}
