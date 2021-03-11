/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	bccrypto "chainmaker.org/chainmaker-go/common/crypto"
	bcx509 "chainmaker.org/chainmaker-go/common/crypto/x509"
	pbac "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/pem"
	"fmt"
	"github.com/gogo/protobuf/proto"
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
	identityType IdentityType

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
	if m.identityType == IdentityTypeCert {
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
	if m.identityType == IdentityTypePublicKey {
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

func (m *member) Serialize(isFullCert bool) ([]byte, error) {
	var serializedMember *pbac.SerializedMember
	if isFullCert {
		var pemStruct *pem.Block
		if m.identityType == IdentityTypePublicKey {
			pemStruct = &pem.Block{Bytes: m.cert.Raw, Type: "PUBLIC KEY"}
		} else {
			pemStruct = &pem.Block{Bytes: m.cert.Raw, Type: "CERTIFICATE"}
		}

		info := pem.EncodeToMemory(pemStruct)

		serializedMember = &pbac.SerializedMember{
			OrgId:      m.orgId,
			MemberInfo: info,
			IsFullCert: true,
		}
	} else {
		id, err := utils.GetCertificateIdFromDER(m.cert.Raw, m.hashType)
		if err != nil {
			return nil, fmt.Errorf("fail to compute certificate identity")
		}

		serializedMember = &pbac.SerializedMember{
			OrgId:      m.orgId,
			MemberInfo: id,
			IsFullCert: false,
		}
	}

	return proto.Marshal(serializedMember)
}

func (m *member) GetSerializedMember(isFullCert bool) (*pbac.SerializedMember, error) {
	if isFullCert {
		var pemStruct *pem.Block
		if m.identityType == IdentityTypePublicKey {
			pemStruct = &pem.Block{Bytes: m.cert.Raw, Type: "PUBLIC KEY"}
		} else {
			pemStruct = &pem.Block{Bytes: m.cert.Raw, Type: "CERTIFICATE"}
		}
		certPEM := pem.EncodeToMemory(pemStruct)
		return &pbac.SerializedMember{
			OrgId:      m.orgId,
			MemberInfo: certPEM,
			IsFullCert: true,
		}, nil
	} else {
		id, err := utils.GetCertificateIdFromDER(m.cert.Raw, m.hashType)
		if err != nil {
			return nil, fmt.Errorf("fail to compute certificate identity")
		}

		return &pbac.SerializedMember{
			OrgId:      m.orgId,
			MemberInfo: id,
			IsFullCert: false,
		}, nil
	}
}

type signingMember struct {
	// Extends Identity
	member

	// Sign the message
	sk bccrypto.PrivateKey
}

func (sm *signingMember) Sign(hashType string, msg []byte) ([]byte, error) {
	var opts bccrypto.SignOpts
	if sm.identityType == IdentityTypePublicKey {
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
