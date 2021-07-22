package accesscontrol

import (
	"fmt"

	bccrypto "chainmaker.org/chainmaker/common/crypto"
	pbac "chainmaker.org/chainmaker/pb-go/accesscontrol"
	"chainmaker.org/chainmaker/protocol"
)

var _ protocol.Member = (*pkMember)(nil)

// an instance whose member type is a certificate
type pkMember struct {

	// pem public key
	id string

	// organization identity who owns this member
	orgId string

	// the public key used for authentication
	pk bccrypto.PublicKey

	// role of this member
	role protocol.Role

	// hash type from chain configuration
	hashType string
}

func (pm *pkMember) GetMemberId() string {
	return pm.id
}

func (pm *pkMember) GetOrgId() string {
	return pm.orgId
}

func (pm *pkMember) GetRole() protocol.Role {
	return pm.role
}

func (pm *pkMember) Verify(msg []byte, sig []byte) error {

	hash, ok := bccrypto.HashAlgoMap[pm.hashType]
	if !ok {
		return fmt.Errorf("cert member verify signature failed: unsupport hash type")
	}
	ok, err := pm.pk.VerifyWithOpts(msg, sig, &bccrypto.SignOpts{
		Hash: hash,
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

func (pm *pkMember) GetMember() (*pbac.Member, error) {
	memberInfo, err := pm.pk.String()
	if err != nil {
		return nil, fmt.Errorf("get pb member failed: %s", err.Error())
	}
	return &pbac.Member{
		OrgId:      pm.orgId,
		MemberInfo: []byte(memberInfo),
		MemberType: pbac.MemberType_CERT,
	}, nil
}

type signingPkMember struct {
	// Extends Identity
	pkMember

	// Sign the message
	sk bccrypto.PrivateKey
}

// When using public key instead of certificate, hashType is used to specify the hash algorithm while the signature algorithm is decided by the public key itself.
func (spm *signingPkMember) Sign(msg []byte) ([]byte, error) {
	hash, ok := bccrypto.HashAlgoMap[spm.hashType]
	if !ok {
		return nil, fmt.Errorf("sign failed: unsupport hash type")
	}
	return spm.sk.SignWithOpts(msg, &bccrypto.SignOpts{
		Hash: hash,
		UID:  bccrypto.CRYPTO_DEFAULT_UID,
	})
}
