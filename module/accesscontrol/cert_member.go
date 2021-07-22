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
