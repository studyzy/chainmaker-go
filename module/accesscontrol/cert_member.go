/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"chainmaker.org/chainmaker/pb-go/config"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/utils"
	bccrypto "chainmaker.org/chainmaker/common/crypto"
	"chainmaker.org/chainmaker/common/crypto/asym"
	"chainmaker.org/chainmaker/common/crypto/pkcs11"
	bcx509 "chainmaker.org/chainmaker/common/crypto/x509"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

var _ protocol.Member = (*certMember)(nil)

// an instance whose member type is a public key
type certMember struct {

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

func newCertMember(orgId, role, hashType string, isCompressed bool, certPEM []byte) (*certMember, error) {
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

	return &certMember{
		id:           string(id),
		orgId:        orgId,
		role:         protocol.Role(role),
		cert:         cert,
		hashType:     hashType,
		isCompressed: isCompressed,
	}, nil
}

func (cm *certMember) GetMemberId() string {
	return cm.id
}

func (cm *certMember) GetOrgId() string {
	return cm.orgId
}

func (cm *certMember) GetRole() protocol.Role {
	return cm.role
}

func (cm *certMember) Verify(hashType string, msg []byte, sig []byte) error {
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

func (cm *certMember) GetMember() (*pbac.Member, error) {
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

type signingCertMember struct {
	// Extends Identity
	certMember

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

func newCertMemberFromPb(member *pbac.Member, acs *accessControlService) (*certMember, error) {

	for _, v := range acs.trustMembers {
		certBlock, _ := pem.Decode([]byte(v.MemberInfo))
		if certBlock == nil {
			return nil, fmt.Errorf("new member failed, the trsut member cert is not PEM")
		}
		if v.MemberInfo == string(member.MemberInfo) {
			var isCompressed bool
			if member.MemberType == pbac.MemberType_CERT {
				isCompressed = false
			}
			return newCertMember(v.OrgId, v.Role, acs.hashType, isCompressed, []byte(v.MemberInfo))
		}
	}

	if member.MemberType == pbac.MemberType_CERT {
		certBlock, rest := pem.Decode(member.MemberInfo)
		if certBlock == nil {
			return newMemberFromCertPem(member.OrgId, acs.hashType, rest, false)
		}
		return newMemberFromCertPem(member.OrgId, acs.hashType, certBlock.Bytes, false)
	}

	if member.MemberType == pbac.MemberType_CERT_HASH {
		return newMemberFromCertPem(member.OrgId, acs.hashType, member.MemberInfo, true)
	}

	return nil, fmt.Errorf("setup member failed, unsupport cert member type")
}

func newMemberFromCertPem(orgId, hashType string, certPEM []byte, isCompressed bool) (*certMember, error) {
	var member certMember
	member.orgId = orgId
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
	orgIdFromCert := cert.Subject.Organization[0]
	if orgIdFromCert != orgId {
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

func NewCertSigningMember(hashType string, member *pbac.Member, privateKeyPem,
	password string) (protocol.SigningMember, error) {
	certMember, err := newMemberFromCertPem(member.OrgId, hashType, member.MemberInfo, false)
	if err != nil {
		return nil, err
	}
	var sk bccrypto.PrivateKey
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	if p11Config.Enabled {
		var p11Handle *pkcs11.P11Handle
		p11Handle, err = getP11Handle()
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%v]", err)
		}

		sk, err = pkcs11.NewPrivateKey(p11Handle, certMember.cert.PublicKey)
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
		certMember: *certMember,
		sk:         sk,
	}, nil
}

func InitCertSigningMember(chainConfig *config.ChainConfig, localOrgId, localPrivKeyFile, localPrivKeyPwd, localCertFile string) (
	protocol.SigningMember, error) {
	var (
		certMember *certMember
	)
	if localPrivKeyFile != "" && localCertFile != "" {
		certPEM, err := ioutil.ReadFile(localCertFile)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}

		isTrustMember := false
		for _, v := range chainConfig.TrustMembers {
			certBlock, _ := pem.Decode([]byte(v.MemberInfo))
			if certBlock == nil {
				return nil, fmt.Errorf("new member failed, the trsut member cert is not PEM")
			}
			if v.MemberInfo == string(certPEM) {
				certMember, err = newCertMember(v.OrgId, v.Role,
					chainConfig.Crypto.Hash, false, certPEM)
				if err != nil {
					return nil, fmt.Errorf("init signing member failed, init trust member failed: [%s]", err.Error())
				}
				isTrustMember = true
				break
			}
		}

		if !isTrustMember {
			certMember, err = newMemberFromCertPem(localOrgId, chainConfig.Crypto.Hash, certPEM, false)
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}
		}

		skPEM, err := ioutil.ReadFile(localPrivKeyFile)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
		}
		var sk bccrypto.PrivateKey
		p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
		if p11Config.Enabled {
			var p11Handle *pkcs11.P11Handle
			p11Handle, err = getP11Handle()
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}

			sk, err = pkcs11.NewPrivateKey(p11Handle, certMember.cert.PublicKey)
			if err != nil {
				return nil, fmt.Errorf("fail to initialize identity management service: [%s]", err.Error())
			}
		} else {
			sk, err = asym.PrivateKeyFromPEM(skPEM, []byte(localPrivKeyPwd))
			if err != nil {
				return nil, err
			}
		}

		return &signingCertMember{
			certMember: *certMember,
			sk:         sk,
		}, nil
	}
	return nil, nil
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
