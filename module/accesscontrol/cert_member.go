package accesscontrol

import (
	"encoding/pem"
	"fmt"
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

	// hash type from chain configuration
	hashType string

	// the certificate is full or hash
	isFullCert bool
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

func (cm *certMember) Verify(msg []byte, sig []byte) error {

	hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(cm.cert.SignatureAlgorithm)
	if err != nil {
		return fmt.Errorf("cert member verify failed: invalid algorithm: %s", err.Error())
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

	if cm.isFullCert == false {
		id, err := utils.GetCertificateIdFromDER(cm.cert.Raw, cm.hashType)
		if err != nil {
			return nil, fmt.Errorf("get pb member failed: fail to compute certificate identity")
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

// When using certificate, the signature-hash algorithm suite is from the certificate, and the input hashType is ignored.
func (scm *signingCertMember) Sign(msg []byte) ([]byte, error) {
	hashAlgo, err := bcx509.GetHashFromSignatureAlgorithm(scm.cert.SignatureAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("sign failed: invalid algorithm: %s", err.Error())
	}
	return scm.sk.SignWithOpts(msg, &bccrypto.SignOpts{
		Hash: hashAlgo,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
}

func NewCertMember(member *pbac.Member, ac *accessControl) (*certMember, error) {
	if member.MemberType == pbac.MemberType_CERT {
		return newMemberFromCertPem(member.OrgId, string(member.MemberInfo), ac.hashType, true)
	}
	if member.MemberType == pbac.MemberType_CERT_HASH {
		certPEM, ok := ac.lookUpCertCache(string(member.MemberInfo))
		if !ok {
			return nil, fmt.Errorf("setup member failed, fail to look up certificate ID")
		}
		if certPEM == nil {
			return nil, fmt.Errorf("setup member failed, unknown certificate ID")
		}
		return newMemberFromCertPem(member.OrgId, string(certPEM), ac.hashType, false)
	}
	return nil, fmt.Errorf("setup member failed, unsupport cert member type")
}

func newMemberFromCertPem(orgId, certPEM string, hashType string, isFullCert bool) (*certMember, error) {
	var certMember certMember
	certMember.orgId = orgId
	certMember.isFullCert = isFullCert
	certMember.hashType = hashType

	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, fmt.Errorf("setup cert member failed, none public key or certificate given")
	}

	cert, err := bcx509.ParseCertificate(certBlock.Bytes)
	if err == nil {
		orgIdFromCert := cert.Subject.Organization[0]
		if orgIdFromCert != orgId {
			return nil, fmt.Errorf("setup cert member failed, organization information in certificate and in input parameter do not match [certificate: %s, parameter: %s]", orgIdFromCert, orgId)
		}
		id, err := bcx509.GetExtByOid(bcx509.OidNodeId, cert.Extensions)
		if err != nil {
			id = []byte(cert.Subject.CommonName)
		}
		certMember.id = string(id)
		certMember.cert = cert
		ou := ""
		if len(cert.Subject.OrganizationalUnit) > 0 {
			ou = cert.Subject.OrganizationalUnit[0]
		}
		ou = strings.ToUpper(ou)
		certMember.role = protocol.Role(ou)
		return &certMember, nil
	}
	return nil, fmt.Errorf("setup cert member failed, invalid public key or certificate")
}

var NilCertMemberProvider MemberProvider = (*certMemberProvider)(nil)

type certMemberProvider struct {
}

func (cmp *certMemberProvider) NewMember(member *pbac.Member, ac *accessControl) (protocol.Member, error) {
	return NewCertMember(member, ac)
}

func NewCertSigningMember(hashType string, member *pbac.Member, privateKeyPem string, password string) (protocol.SigningMember, error) {
	certMember, err := newMemberFromCertPem(member.OrgId, string(member.MemberInfo), hashType, true)
	if err != nil {
		return nil, err
	}
	var sk bccrypto.PrivateKey
	p11Config := localconf.ChainMakerConfig.NodeConfig.P11Config
	if p11Config.Enabled {
		p11Handle, err := getP11Handle()
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
